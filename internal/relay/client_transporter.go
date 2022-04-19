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

package relay

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/pairmesh/pairmesh/internal/codec"
	"go.uber.org/atomic"
	"go.uber.org/zap"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/pairmesh/pairmesh/security"
)

// ClientTransporterState represents the state of client transporter
type ClientTransporterState byte

// ClientTransporterState represents the state of client transporter
const (
	ClientTransporterStateInit       ClientTransporterState = 0
	ClientTransporterStateConnecting ClientTransporterState = 1
	ClientTransporterStateConnected  ClientTransporterState = 2
)

// ClientTransporter interface, together with clientTransporterImpl struct,
// is an abstraction of network IO so that the network behaviors could be mocked.
type ClientTransporter interface {
	RelayServer() protocol.RelayServer
	State() ClientTransporterState
	SetState(s ClientTransporterState)
	Cipher() noise.Cipher
	SetCipher(cipher noise.Cipher)
	SetHeartbeatInterval(interval time.Duration)
	SetIsPrimary(is bool)
	HandshakeState() *noise.HandshakeState
	ReadQueue() <-chan codec.RawPacket
	WriteQueue() chan<- Packet
	Connect(ctx context.Context) error
	Read(ctx context.Context)
	Write(ctx context.Context)
	Close() error
}

type clientTransporterImpl struct {
	securityTransporter

	relayServer       protocol.RelayServer
	credentials       []byte
	die               chan struct{}
	nodeDHKey         noise.DHKey
	srvPubKey         security.DHPublic // of the relay server; not a machine or node key
	state             ClientTransporterState
	handshakeState    *noise.HandshakeState
	heartbeatInterval time.Duration
	isPrimary         bool
	closed            *atomic.Bool
	hsSignal          chan struct{} // handshake notifier
}

// NewClientTransporter generates a clientTransporterImpl struct with given server and credentials
func NewClientTransporter(server protocol.RelayServer, credentials []byte, nodeDHKey noise.DHKey, srvPubKey security.DHPublic) *clientTransporterImpl {
	return &clientTransporterImpl{
		securityTransporter: newSecurityTransporter(nil),
		relayServer:         server,
		credentials:         credentials,
		die:                 make(chan struct{}, 1),
		nodeDHKey:           nodeDHKey,
		srvPubKey:           srvPubKey,
		state:               ClientTransporterStateInit,
		closed:              atomic.NewBool(false),
		hsSignal:            make(chan struct{}, 1),
	}
}

// RelayServer returns the relay server information of the client transporter
func (c *clientTransporterImpl) RelayServer() protocol.RelayServer {
	return c.relayServer
}

// State returns the state of the client transporter
func (c *clientTransporterImpl) State() ClientTransporterState {
	return c.state
}

// SetState sets the state of the client transporter
func (c *clientTransporterImpl) SetState(s ClientTransporterState) {
	// Handshake finished signal.
	if c.state == ClientTransporterStateConnecting && s == ClientTransporterStateConnected {
		close(c.hsSignal)
	}

	c.state = s
}

// SetHeartbeatInterval sets the heartbeat interval of the client transporter
func (c *clientTransporterImpl) SetHeartbeatInterval(interval time.Duration) {
	c.heartbeatInterval = interval
}

// SetIsPrimary sets whether this client transporter is primary
func (c *clientTransporterImpl) SetIsPrimary(is bool) {
	c.isPrimary = is
}

// HandshakeState returns the handshake state of the client transporter
func (c *clientTransporterImpl) HandshakeState() *noise.HandshakeState {
	return c.handshakeState
}

// Connect connects to the relay server.
func (c *clientTransporterImpl) Connect(ctx context.Context) error {

	if c.state != ClientTransporterStateInit {
		return errors.New("cannot connect remote MERP server due to state isn't init")
	}

	addr := fmt.Sprintf("%s:%d", c.relayServer.Host, c.relayServer.Port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c.conn = conn
	c.state = ClientTransporterStateConnecting

	go c.Read(ctx)
	go c.Write(ctx)

	noiseCfg := noise.Config{
		CipherSuite:   security.CipherSuite,
		Pattern:       security.HandshakePatternNN,
		Initiator:     true,
		StaticKeypair: c.nodeDHKey,
	}
	state, err := noise.NewHandshakeState(noiseCfg)
	if err != nil {
		return err
	}

	c.handshakeState = state

	// Make `credentials` as the handshake payload message.
	out, _, _, err := c.handshakeState.WriteMessage(make([]byte, 0, 128), c.credentials)
	if err != nil {
		return err
	}

	msg := &message.PacketHandshake{
		Message:   out,
		IsPrimary: c.isPrimary,
	}

	c.chWrite <- Packet{
		Type:    message.PacketType_Handshake,
		Message: msg,
	}

	select {
	case <-c.hsSignal:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Read is the job to read data from connection, decode it and pushes to chRead
func (c *clientTransporterImpl) Read(ctx context.Context) {
	defer func() {
		if e := recover(); e != nil {
			zap.L().Error("Read thread panicked", zap.Reflect("error", e))
		}

		_ = c.Close()
		close(c.chRead)
	}()

	buffer := make([]byte, bufferSize)
	for {
		n, err := c.conn.Read(buffer)
		if err != nil {
			zap.L().Error("Read relay server message failed", zap.Error(err))
			return
		}

		output, err := c.codec.Decode(buffer[:n])
		if err != nil {
			zap.L().Error("Codec relay server message failed", zap.Error(err))
			return
		}
		for _, p := range output {
			c.chRead <- p
		}
	}
}

// Write is the job to detect data from chWrite, and sends it through connection
func (c *clientTransporterImpl) Write(ctx context.Context) {
	defer func() {
		if e := recover(); e != nil {
			zap.L().Error("Write thread panicked", zap.Reflect("error", e))
		}

		_ = c.Close()
		close(c.chWrite)
	}()

	// Default to 1 second
	heartbeatTimer := time.After(time.Second)
	for {
		select {
		case wp := <-c.chWrite:
			err := writePacketHelper(c.conn, wp, c.cipher, c.codec, 5*time.Second)
			if err != nil {
				zap.L().Error("Write message failed", zap.Error(err))
				return
			}

		case <-heartbeatTimer:
			if c.state != ClientTransporterStateConnected {
				heartbeatTimer = time.After(time.Second)
				continue
			}
			if c.closed.Load() {
				zap.L().Error("Cannot send heartbeat message since the client transporter is closed.")
				continue
			}
			c.chWrite <- Packet{
				Type: message.PacketType_Heartbeat,
				Message: &message.PacketHeartbeat{
					Timestamp: time.Now().UnixNano(),
				},
			}
			heartbeatTimer = time.After(c.heartbeatInterval)

		case <-ctx.Done():
			return
		case <-c.die:
			return
		}
	}
}

// Close actually closes the client transporter
func (c *clientTransporterImpl) Close() error {
	if c.closed.Swap(true) {
		return errors.New("close a closed client")
	}

	zap.L().Warn("Client connection transporter terminated", zap.Stringer("addr", c.conn.RemoteAddr()))

	close(c.die)
	return c.conn.Close()
}
