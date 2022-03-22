// Copyright 2021 PairMesh, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"encoding/base64"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/node/api"
	"github.com/pairmesh/pairmesh/node/config"
	"github.com/pairmesh/pairmesh/node/device"
	"github.com/pairmesh/pairmesh/node/mesh"
	"github.com/pairmesh/pairmesh/node/mesh/tunnel"
	"github.com/pairmesh/pairmesh/node/mesh/types"
	"github.com/pairmesh/pairmesh/node/monitor"
	"github.com/pairmesh/pairmesh/protocol"

	"github.com/libp2p/go-reuseport"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"inet.af/netaddr"
)

var _ Driver = &NodeDriver{}

type SummaryChangedCallback func()

type Driver interface {
	tunnel.FragmentCallback

	// Preflight initializes the driver state and check whether can the driver go forward.
	Preflight() error

	// Drive starts the driver and serve the application traffics.
	Drive(ctx context.Context)

	// Enable enables the driver to serve the application.
	Enable()

	// Disable disables the driver to serve the application and all packets
	// will be dropped if the device in disabled status.
	Disable()

	// Summarize summarizes the driver current state and returns the state and
	// mesh summary.
	Summarize() *Summary

	// Terminate closes the PairMesh engine.
	Terminate()
}

// NodeDriver implements the Driver interface.
// NodeDriver is mainly used to control the overall procedure of
// peerly application.
// 1. Initialize all resources in the Initialize function.
// 2. Some long-running routines:
//    - Renew credential goroutine will keep the credential up to date.
//    - Read tunnel device data goroutine will read the data from tunnel device.
//    - Write tunnel device data goroutine will write the data to the tunnel device.
// 3. Filter out data via filter.
type NodeDriver struct {
	// Concurrent-safe fields.
	wg           *sync.WaitGroup
	running      atomic.Bool // indicates whether the driver has been initialized.
	termed       atomic.Bool // indicates whether the driver has been terminated.
	enable       atomic.Bool
	chDevWrite   chan []byte
	externalAddr atomic.String

	// Read-only fields after initialized.
	apiClient *api.Client
	config    *config.Config
	peerID    protocol.PeerID
	userID    protocol.UserID
	name      string
	dialer    *net.Dialer
	mm        *mesh.Manager
	rm        *relay.Manager
	device    device.Device
	mon       *monitor.Monitor

	// Driver will keep updating local endpoints to the primary relay server.
	// The field primaryServerConnected is used to indicate the status
	// of connection to the primary relay server.
	// Note: It is only accessed by update endpoints thread.
	primaryServerConnected bool

	// mu is used to protect the following fields.
	mu         sync.Mutex //nolint ; should be used somewhere
	credential credential
}

// New constructs the engines instance.
func New(cfg *config.Config, dev device.Device, apiClient *api.Client) Driver {
	return &NodeDriver{
		wg:         &sync.WaitGroup{},
		enable:     *atomic.NewBool(true),
		chDevWrite: make(chan []byte, 512),
		apiClient:  apiClient,
		config:     cfg,
		device:     dev,
	}
}

func (d *NodeDriver) Preflight() error {
	hostname, err := os.Hostname()
	if err != nil {
		zap.L().Error("Retrieve the host name failed", zap.Error(err))
	}

	// Send a request to the portal service Preflight interface to
	// retrieve the initial data essential to initialize the driver.
	res, err := d.apiClient.Preflight(runtime.GOOS, hostname)
	if err != nil {
		return err
	}

	zap.L().Info("Preflight request successful")

	// We use a customized dialer to make sure all the traffic from
	// the current node has the same local address (host:port).
	udpAddr := &net.UDPAddr{IP: net.IPv4zero, Port: d.config.Port}
	d.dialer = &net.Dialer{
		Control:   reuseport.Control,
		LocalAddr: udpAddr,
	}

	// Parse and initialize the credential and there is a thread to renew it.
	// The credential will be a part of handshake with relay server to prove
	// the current node owns the IP address. The IP address is encoded in the
	// credential and signed by the portal service. The relay server will
	// verify the credential (PeerID/IP) via RSA public key.
	cred, err := base64.RawStdEncoding.DecodeString(res.Credential)
	if err != nil {
		return errors.WithMessage(err, "decode base64 credential")
	}
	d.credential = credential{
		address:   res.IPv4,
		RawBytes:  cred,
		Base64:    res.Credential,
		RenewedAt: time.Now(),
		Lease:     time.Duration(res.CredentialLease) * time.Second,
	}
	d.userID = res.UserID
	d.peerID = res.ID
	d.name = res.Name

	// Up the virtual device with the specified address which is allocated
	// by the portal service.
	err = d.device.Up(res.IPv4)
	if err != nil {
		return errors.WithMessage(err, "set device address")
	}

	zap.L().Info("Set virtual device address finished", zap.String("address", res.IPv4))

	// Preflight the monitor service which is used to discover external address.
	d.mon = monitor.New(d.dialer, res.PrimaryServer)

	// Register all relay clients into the relay manager
	d.rm = relay.NewManager(d.config.DHKey, d)
	d.rm.SetCredential(cred)

	vIPV4Addr, err := netaddr.ParseIP(res.IPv4)
	if err != nil {
		return errors.WithMessage(err, "parse ipv4 address")
	}

	nodeInfo := types.LocalPeer{
		Name:   res.Name,
		UserID: res.UserID,
		PeerID: res.ID,
		Key:    d.config.DHKey,
		VIPv4:  vIPV4Addr,
	}
	d.mm = mesh.NewManager(d.dialer, nodeInfo, d, d.rm, d.device.Router())

	zap.L().Info("Driver preflight finished")

	return nil
}

// Drive implement the Driver interface and will start all background threads
// to drive the application followup process.
func (d *NodeDriver) Drive(ctx context.Context) {
	if d.running.Swap(true) {
		zap.L().Error("The drive method had been called duplicated")
		return
	}

	// Begin virtual network interface traffic handling.
	d.wg.Add(2)
	go d.serveDevRead(ctx)
	go d.serveDevWrite(ctx)

	// Begin to periodically retrieve the peers graph.
	// Keep the relay server information up to date with the portal.
	d.wg.Add(1)
	go d.pullPeerGraph(ctx)

	d.wg.Add(1)
	go d.eventsMonitor(ctx)

	// Begin to monitoring the link change and external address change.
	d.wg.Add(1)
	go d.mon.Monitoring(ctx, d.wg)

	zap.L().Info("All background threads running")
}

// Enable implements the Driver interface
func (d *NodeDriver) Enable() {
	zap.L().Info("Enable driver and will handle all traffics")
	d.enable.Store(true)
}

// Disable implements the Driver interface
func (d *NodeDriver) Disable() {
	zap.L().Info("Disable driver and will drop all traffics")
	d.enable.Store(false)
}

// Summarize implements the Driver interface.
func (d *NodeDriver) Summarize() *Summary {
	if mockSummary {
		return d.mockSummarize()
	}

	status := "connecting"
	if d.primaryServerConnected {
		status = "connected"
	}
	var meshSummary *mesh.Summary
	if d.running.Load() {
		meshSummary = d.mm.Summarize()
	} else {
		meshSummary = &mesh.Summary{}
	}
	return &Summary{
		Enabled: d.enable.Load(),
		Status:  status,
		Profile: &Profile{
			UserID: uint64(d.userID),
			IPv4:   d.credential.address,
			Name:   d.name,
		},
		Mesh: meshSummary,
	}
}

// Terminate implements the Driver interface
func (d *NodeDriver) Terminate() {
	if !d.running.Load() {
		return
	}

	if d.termed.Swap(true) {
		return
	}

	if d.rm != nil {
		d.rm.Stop()
	}

	// Wait all goroutines exit
	d.wg.Wait()

	zap.L().Info("The Driver is powered off, see you again")
}

// MockDriver is a wrapper of NodeDriver for testing use.
type MockDriver struct {
	NodeDriver
}

func (d *MockDriver) SetPeerID(id protocol.PeerID) {
	d.NodeDriver.peerID = id
}
