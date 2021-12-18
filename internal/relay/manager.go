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
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/security"

	"google.golang.org/protobuf/proto"

	"github.com/pairmesh/pairmesh/internal/logutil"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"

	"go.uber.org/atomic"

	"github.com/flynn/noise"
	"go.uber.org/zap"
)

const eventBufferSize = 256

type (
	PacketCallback interface {
		// OnForward handles the messages forwarded by the relay server
		OnForward(s *Client, typ message.PacketType, msg proto.Message) error
		OnSyncPeer(s *Client, typ message.PacketType, msg proto.Message) error
		OnProbeResponse(s *Client, typ message.PacketType, msg proto.Message) error
	}

	// Manager maintains the relay clients and keep heartbeat with the
	// relay server. When a peer cannot connect to the remote peer, the lowest
	// latency relay server will be used to relay the traffic.
	Manager struct {
		closed     atomic.Bool
		staticKey  noise.DHKey
		credential atomic.Value
		primary    atomic.Uint64 // primary represents the home relay server id of the current node.
		callback   PacketCallback
		events     chan Event

		wg      *sync.WaitGroup
		clients sync.Map // protocol.ServerID -> *relay.Client
		pending sync.Map // protocol.ServerID -> protocol.RelayServer
	}
)

// NewManager returns the relay manager
func NewManager(staticKey noise.DHKey, callback PacketCallback) *Manager {
	return &Manager{
		staticKey: staticKey,
		callback:  callback,
		events:    make(chan Event, eventBufferSize),
		wg:        &sync.WaitGroup{},
	}
}

func (m *Manager) SetCredential(credential []byte) {
	m.credential.Store(credential)
}

// PrimaryServerID returns the primary relay server id.
func (m *Manager) PrimaryServerID() protocol.ServerID {
	return protocol.ServerID(m.primary.Load())
}

// SetPrimaryServerID sets the primary relay server id.
func (m *Manager) SetPrimaryServerID(id protocol.ServerID) {
	m.primary.Store(uint64(id))
}

// PrimaryRelayServerClient returns the primary relay server client of
// current node.
// A nil value will be returned if the primary relay server didn't connect.
func (m *Manager) PrimaryRelayServerClient() *Client {
	priServerID := protocol.ServerID(m.primary.Load())
	if priServerID == 0 {
		return nil
	}
	v, ok := m.clients.Load(priServerID)
	if !ok {
		return nil
	}
	return v.(*Client)
}

// RelayServerClient returns the relay server client corresponding server id.
// A nil value will be returned if the primary relay server didn't connect.
func (m *Manager) RelayServerClient(id protocol.ServerID) *Client {
	v, ok := m.clients.Load(id)
	if !ok {
		return nil
	}
	return v.(*Client)
}

// Events returns a channel which will recording clients connected/closed event.
// Some events will be dropped if the channel is full.
func (m *Manager) Events() <-chan Event {
	return m.events
}

func (m *Manager) event(e Event) {
	// Is it ok to block connect thread?
	m.events <- e
}

func (m *Manager) connect(ctx context.Context, r protocol.RelayServer) bool {
	_, ok := m.clients.Load(r.ID)
	if ok {
		return true
	}

	address := fmt.Sprintf("%s:%d", r.Host, r.Port)
	if logutil.IsEnablePortal() {
		zap.L().Debug("Add relay server", zap.String("address", address))
	}

	publicKey, err := base64.StdEncoding.DecodeString(r.PublicKey)
	if err != nil {
		zap.L().Error("Unmarshal public key of relay server failed", zap.String("vaddress", address), zap.Error(err))
		return false
	}

	client := NewClient(r, m.credential.Load().([]byte), m.staticKey, security.NewDHPublic(publicKey))
	if err := client.Connect(ctx); err != nil {
		zap.L().Error("Connect to relay server failed", zap.String("vaddress", address), zap.Error(err))
		return false
	}

	m.clients.Store(r.ID, client)

	m.event(Event{
		Type: EventTypeClientConnected,
		Data: EventClientConnected{
			RelayServer: r,
			Client:      client,
		},
	})

	// Proxy the callback of client to the driver callback.
	client.Handler().On(message.PacketType_Forward, m.callback.OnForward)
	client.Handler().On(message.PacketType_SyncPeer, m.callback.OnSyncPeer)
	client.Handler().On(message.PacketType_ProbeResponse, m.callback.OnProbeResponse)

	// Avoid closure problem.
	capture := r
	client.OnClosed(func() {
		// Don't reconnect when manager stopped.
		if m.closed.Load() {
			return
		}

		// Don't reconnect if it is not a managed client.
		if _, found := m.clients.Load(capture.ID); !found {
			return
		}

		m.event(Event{
			Type: EventTypeClientClosed,
			Data: EventClientClosed{
				RelayServer: r,
				Client:      client,
			},
		})

		// Remove the client from the clients list.
		m.clients.Delete(capture.ID)

		// Reconnect to the relay server if the connection closed.
		m.pending.Store(capture.ID, capture)
	})

	return true
}

// Tick ticks the relay client manager to flush the pending connecting relay servers.
func (m *Manager) Tick(ctx context.Context) {
	m.pending.Range(func(key, value interface{}) bool {
		id := key.(protocol.ServerID)
		r := value.(protocol.RelayServer)
		connected := m.connect(ctx, r)
		if connected {
			m.pending.Delete(id)
		}
		return true
	})
}

func (m *Manager) AddServer(relayServer protocol.RelayServer) {
	_, found := m.clients.Load(relayServer.ID)
	if found {
		return
	}

	// Try to connect first.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	connected := m.connect(ctx, relayServer)
	if connected {
		return
	}

	// Add the pending list to try again later.
	_, found = m.pending.Load(relayServer.ID)
	if found {
		return
	}

	m.pending.Store(relayServer.ID, relayServer)
}

// Update updates the relay clients maintained by the relay manager.
func (m *Manager) Update(ctx context.Context, relayServers []protocol.RelayServer) {
	existing := map[protocol.ServerID]struct{}{}
	for _, r := range relayServers {
		existing[r.ID] = struct{}{}

		connected := m.connect(ctx, r)
		if connected {
			continue
		}

		m.pending.Store(r.ID, r)
	}

	// Prune the redundant clients.
	m.clients.Range(func(key, value interface{}) bool {
		id := key.(protocol.ServerID)
		_, found := existing[id]
		if !found {
			// MUST delete first to avoid auto-reconnect when closing client connection.
			m.clients.Delete(id)
			m.pending.Delete(id)

			client := value.(*Client)
			err := client.Close()
			if err != nil {
				zap.L().Error("Close relay server connection failed", zap.Error(err))
			}
		}
		return true
	})
}

// Stop stops the relay manager and all the relay clients.
func (m *Manager) Stop() {
	if m.closed.Swap(true) {
		return
	}

	m.clients.Range(func(key, value interface{}) bool {
		id := key.(protocol.ServerID)
		client := value.(*Client)
		if err := client.Close(); err != nil {
			zap.L().Error("Close relay server connection failed", zap.Any("relay_server_id", id))
		}
		return true
	})

	m.wg.Wait()
	zap.L().Info("The relay manager is stopped")
}
