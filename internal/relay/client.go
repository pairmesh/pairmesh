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

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/codec"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/pairmesh/pairmesh/security"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type ClientState byte

const (
	ClientStateInit       ClientState = 0
	ClientStateConnecting ClientState = 1
	ClientStateConnected  ClientState = 2
)

// Client represents the relay server client which is used to interactive with relay server.
type Client struct {
	relayServer    protocol.RelayServer
	credentials    []byte
	nodeDHKey      noise.DHKey
	srvPubKey      security.DHPublic // of the relay server; not a machine or node key
	handshakeState *noise.HandshakeState
	cipher         noise.Cipher
	codec          *codec.RelayCodec
	handler        ClientHandler
	isPrimary      bool

	heartbeatInterval time.Duration
	lastHeartbeatAt   time.Time
	lastMeasuredLat   time.Duration

	conn     net.Conn
	closed   *atomic.Bool
	onClosed func() // Callback function
	state    ClientState
	die      chan struct{}
	chRead   chan codec.RawPacket
	chWrite  chan Packet

	// handshake notifier
	hsSignal chan struct{}
}

// NewClient returns a new client instance.
func NewClient(server protocol.RelayServer, credentials []byte, nodeDHKey noise.DHKey, srvPubKey security.DHPublic) *Client {
	s := &Client{
		relayServer: server,
		credentials: credentials,
		nodeDHKey:   nodeDHKey,
		srvPubKey:   srvPubKey,
		closed:      atomic.NewBool(false),
		state:       ClientStateInit,
		die:         make(chan struct{}, 1),
		hsSignal:    make(chan struct{}, 1),
		chRead:      make(chan codec.RawPacket, 64),
		chWrite:     make(chan Packet, 64),
		codec:       codec.NewCodec(),
		handler:     NewClientHandler(),
	}
	return s
}

// State returns the current state of the client.
func (c *Client) State() ClientState {
	return c.state
}

// SetState sets the state of the client.
func (c *Client) SetState(s ClientState) {
	// Handshake finished signal.
	if c.state == ClientStateConnecting && s == ClientStateConnected {
		close(c.hsSignal)
	}
	c.state = s
}

func (c *Client) SetHeartbeatInterval(interval time.Duration) {
	c.heartbeatInterval = interval
}

func (c *Client) SetLastHeartbeatAt(t time.Time) {
	c.lastHeartbeatAt = t
}

func (c *Client) SetLastMeasuredLat(lat time.Duration) {
	c.lastMeasuredLat = lat
}

func (c *Client) SetIsPrimary(is bool) {
	c.isPrimary = is
}

// HandshakeState returns the handshake state of noise protocol.
func (c *Client) HandshakeState() *noise.HandshakeState {
	return c.handshakeState
}

func (c *Client) Cipher() noise.Cipher {
	return c.cipher
}

func (c *Client) SetCipher(cipher noise.Cipher) {
	c.cipher = cipher
}

func (c *Client) Handler() ClientHandler {
	return c.handler
}

func (c *Client) RelayServer() protocol.RelayServer {
	return c.relayServer
}

// Connect connects to the relay server.
func (c *Client) Connect(ctx context.Context) error {
	if c.state != ClientStateInit {
		return errors.New("cannot connect remote MERP server due to state isn't init")
	}

	addr := fmt.Sprintf("%s:%d", c.relayServer.Host, c.relayServer.Port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c.conn = conn
	c.state = ClientStateConnecting

	go c.read(ctx)
	go c.write(ctx)

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
		PublicKey: c.nodeDHKey.Public,
		Message:   out,
		IsPrimary: c.isPrimary,
	}

	// Start handshake procedure.
	if err := c.Send(message.PacketType_Handshake, msg); err != nil {
		return err
	}

	select {
	case <-c.hsSignal:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) OnClosed(cb func()) {
	c.onClosed = cb
}

func (c *Client) Send(typ message.PacketType, msg proto.Message) error {
	if c.closed.Load() {
		return errors.New("cannot send message to closed client")
	}

	select {
	case c.chWrite <- Packet{Type: typ, Message: msg}:
		return nil
	default:
		return fmt.Errorf("send buffer excced: %c", c.conn.RemoteAddr())
	}
}

func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return errors.New("close a closed client")
	}
	close(c.die)
	if c.onClosed != nil {
		c.onClosed()
	}
	return c.conn.Close()
}

func (c *Client) read(ctx context.Context) {
	defer func() {
		c.Close()
		zap.L().Warn("Client connection terminated",
			zap.String("addr", fmt.Sprintf("%s:%d", c.relayServer.Host, c.relayServer.Port)))
	}()

	go func() {
		select {
		case <-ctx.Done():
			c.Close()
		case <-c.die:
		}
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
			if err := c.handler.Handle(c, p); err != nil {
				zap.L().Error("Handle relay server message failed", zap.Error(err))
				continue
			}
		}
	}
}

func (c *Client) write(ctx context.Context) {
	defer close(c.chWrite)

	// Default to 1 second
	heartbeatTimer := time.After(time.Second)
	for {
		select {
		case wp := <-c.chWrite:
			err := writePacketHelper(c.conn, wp, c.cipher, c.codec, 5*time.Second)
			if err != nil {
				zap.L().Error("Write message failed", zap.Error(err))
				_ = c.Close()
				return
			}

		case <-heartbeatTimer:
			if c.state != ClientStateConnected {
				heartbeatTimer = time.After(time.Second)
				continue
			}
			err := c.Send(message.PacketType_Heartbeat, &message.PacketHeartbeat{
				Timestamp: time.Now().UnixNano(),
			})
			if err != nil {
				zap.L().Error("Send heartbeat message failed", zap.Error(err))
			}
			heartbeatTimer = time.After(c.heartbeatInterval)

		case <-ctx.Done():
			return
		case <-c.die:
			return
		}
	}
}
