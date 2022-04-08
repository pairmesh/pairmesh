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

package monitor

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/internal/stun"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/protocol"

	"github.com/pkg/errors"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const eventBufferSize = 256

var (
	// ErrNoSTUNServer is the error message with there is no STUN server found
	ErrNoSTUNServer = errors.New("no STUN server found")
)

// Monitor is used to monitoring the link change.
type Monitor struct {
	dialer       *net.Dialer
	events       chan Event
	stunServer   atomic.Value // An atomic value of type: protocol.RelayServer
	externalAddr atomic.Value // An atomic value of type: string (cached external address)
}

// New returns the monitor instance which is used to detect the external address
// of the current node.
func New(dialer *net.Dialer, stunServer protocol.RelayServer) *Monitor {
	mon := &Monitor{
		dialer: dialer,
		events: make(chan Event, eventBufferSize),
	}
	mon.stunServer.Store(stunServer)
	return mon
}

// SetSTUNServer sets the latest STUN server (the same as the primary relay server).
func (m *Monitor) SetSTUNServer(stunServer protocol.RelayServer) {
	m.stunServer.Store(stunServer)
}

// ExternalAddress returns the external address of current node.
func (m *Monitor) ExternalAddress() string {
	val := m.externalAddr.Load()
	if val == nil {
		return ""
	}
	return val.(string)
}

// Events returns the events channel
func (m *Monitor) Events() <-chan Event {
	return m.events
}

func (m *Monitor) event(event Event) {
	// Is it ok to block monitoring thread?
	m.events <- event
}

// Monitoring is the job to monitor detectExternalAddressTimer and handle the monitoring task
func (m *Monitor) Monitoring(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	const defaultInterval = 30 * time.Second

	// fastDetectInterval indicates the relay server not ready.
	const fastDetectInterval = 2 * time.Second

	detectExternalAddressTimer := time.After(0)

	for {
		select {
		case <-detectExternalAddressTimer:
			var interval = defaultInterval
			externalAddr, err := m.DetectExternalAddress(ctx, false)
			if err == ErrNoSTUNServer {
				interval = fastDetectInterval
			}
			detectExternalAddressTimer = time.After(interval)
			if err != nil {
				zap.L().Error("Detect external address failed", zap.Error(err))
				continue
			}
			if old := m.externalAddr.Load(); old == nil || externalAddr != old.(string) {
				if logutil.IsEnableRelay() {
					zap.L().Debug("Detected external address", zap.String("address", externalAddr))
				}

				m.externalAddr.Store(externalAddr)
				m.event(Event{
					Type: EventTypeExternalAddressChanged,
					Data: EventExternalAddressChanged{
						ExternalAddress: externalAddr,
					},
				})
			}

		case <-ctx.Done():
			zap.L().Info("Monitoring goroutine finished", zap.Error(ctx.Err()))
			return
		}
	}
}

// DetectExternalAddress detect the external address of the current node.
func (m *Monitor) DetectExternalAddress(ctx context.Context, persistent bool) (string, error) {
	val := m.stunServer.Load()
	if val == nil {
		return "", ErrNoSTUNServer
	}

	type inflightRecord struct {
		txID stun.TxID
		time time.Time
	}

	var inflight = map[stun.TxID]inflightRecord{}
	// GC inflights if it's count great-than 20 (only retain the first haft)
	gcInflight := func() {
		if len(inflight) > 20 {
			records := make([]inflightRecord, 0, len(inflight))
			for _, inflight := range inflight {
				records = append(records, inflight)
			}

			// Sort them by the time
			sort.Slice(records, func(i, j int) bool {
				return records[i].time.Before(records[j].time)
			})

			// Delete the first half.
			for _, record := range records[len(records)/2:] {
				delete(inflight, record.txID)
			}
		}
	}

	stunServer := val.(protocol.RelayServer)
	address := fmt.Sprintf("%s:%d", stunServer.Host, stunServer.STUNPort)
	stunConn, err := m.dialer.DialContext(ctx, "udp", address)
	if err != nil {
		return "", err
	}
	defer stunConn.Close()

	udpConn := stunConn.(*net.UDPConn)
	writeSTUNPacket := func() error {
		gcInflight()
		txID := stun.NewTxID()
		inflight[txID] = inflightRecord{txID: txID, time: time.Now()}
		_, err := udpConn.Write(stun.Request(txID))
		return err
	}

	buffer := make([]byte, constant.MaxBufferSize)
	for {
		_ = udpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, remote, err := udpConn.ReadFromUDP(buffer)
		if errors.Is(err, os.ErrDeadlineExceeded) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()

			default:
				// Write STUN packet again if the STUN response cannot be received in
				// a reasonable duration.
				if werr := writeSTUNPacket(); werr != nil {
					return "", werr
				}
				continue
			}
		}

		if err != nil {
			return "", err
		}
		if remote.String() != address {
			zap.L().Warn("Ignore STUN packet due to remote address mismatch", zap.String("remote", remote.String()))
			continue
		}

		txID, addr, port, err := stun.ParseResponse(buffer[:n])
		if err != nil {
			zap.L().Error("Parser STUN response failed", zap.Error(err))
			continue
		}

		_, found := inflight[txID]
		if !found {
			zap.L().Warn("Receive unrecognized STUN message")
			continue
		}

		externalAddr := fmt.Sprintf("%s:%d", net.IP(addr).String(), port)

		if persistent {
			m.externalAddr.Store(externalAddr)
		}

		return externalAddr, nil
	}
}
