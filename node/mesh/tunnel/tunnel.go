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

package tunnel

import (
	"math"
	"math/rand"
	"net"
	"sort"
	"time"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/internal/codec"
	"github.com/pairmesh/pairmesh/internal/codec/serde"
	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/node/mesh/types"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/protocol"

	"github.com/flynn/noise"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// ZeroTime is the default time initialized with zero value
var ZeroTime = time.Time{}

type (
	// RelayClientGetter is a function to get relay client
	RelayClientGetter func() *relay.Client

	// Tunnel represents the remote conn and maintain the conn state.
	Tunnel struct {
		// Read-only fields
		rcGetter  RelayClientGetter
		dialer    *net.Dialer
		callback  FragmentCallback
		localPeer types.LocalPeer
		peerID    protocol.PeerID
		cipher    noise.Cipher

		disco          atomic.Bool
		closed         atomic.Bool
		localEndpoints atomic.Value // An atomic value of type []string
		endpoints      atomic.Value // An atomic value of type []*Endpoint
		endpointsCh    chan []string
		die            chan struct{}

		pairCounter int
		nextPairAt  time.Time
		lastSendAt  time.Time
		lastRecvAt  time.Time
	}
)

// New generates and returns a Tunnel struct with given parameters
func New(rcGetter RelayClientGetter, dialer *net.Dialer, localPeer types.LocalPeer, peerID protocol.PeerID, callback FragmentCallback, cipher noise.Cipher) *Tunnel {
	t := &Tunnel{
		rcGetter:    rcGetter,
		dialer:      dialer,
		callback:    callback,
		localPeer:   localPeer,
		peerID:      peerID,
		cipher:      cipher,
		endpointsCh: make(chan []string, 2),
		die:         make(chan struct{}),
	}
	return t
}

// Cipher returns the noise.Cipher.
func (t *Tunnel) Cipher() noise.Cipher {
	return t.cipher
}

// IsDisco returns whether the tunnel is discovered
func (t *Tunnel) IsDisco() bool {
	return t.disco.Load()
}

// SetLocalEndpoints sets the given endpoints to the tunnel
func (t *Tunnel) SetLocalEndpoints(endpoints []string) {
	t.localEndpoints.Store(endpoints)
}

// SetRemoteEndpoints sets given remote endpoints to the tunnel
func (t *Tunnel) SetRemoteEndpoints(endpoints []string) {
	t.endpointsCh <- endpoints
	if !t.disco.Load() {
		go t.discovery()
	}
}

// Write writes input data through the tunnel
func (t *Tunnel) Write(data []byte) {
	if logutil.IsEnablePeer() {
		zap.L().Debug("Send fragment", zap.Any("peer", t.localPeer.PeerID))
	}

	t.lastSendAt = time.Now()

	endpoint := t.ReachableEndpoint()
	if endpoint != nil {
		encoded := codec.Encode(message.PacketType_Fragment, t.cipher, t.localPeer.PeerID, data)
		endpoint.Write(encoded)
		return
	}

	// Forward fragment using relay server.

	relayClient := t.rcGetter()
	if relayClient == nil {
		zap.L().Warn("Drop data due to primary relay server client not ready", zap.Reflect("peer", t.peerID))
		return
	}

	nonce := rand.Uint32()
	encrypted := t.cipher.Encrypt(nil, uint64(nonce), nil, data)
	if logutil.IsEnableRelay() {
		zap.L().Debug("Relay data due to no UDP endpoint", zap.Int("length", len(encrypted)))
	}

	packet := &message.PacketForward{
		SrcPeerID: uint64(t.localPeer.PeerID),
		DstPeerID: uint64(t.peerID),
		Nonce:     nonce,
		Fragment:  encrypted,
	}

	err := relayClient.Send(message.PacketType_Forward, packet)
	if err != nil {
		zap.L().Error("Relay message failed", zap.Error(err))
	}

	// No endpoints available if the tunnel is discoverying means all endpoints cannot reachable
	// to the remote peer and wait newly endpoints change events.
	if !t.disco.Load() && t.lastSendAt.After(t.nextPairAt) {
		localEndpoints := t.localEndpoints.Load()
		if localEndpoints == nil {
			return
		}

		endpoints := localEndpoints.([]string)
		if len(endpoints) == 0 {
			return
		}

		// Exponential backoff time.
		syncPeer := &message.PacketSyncPeer{
			DstPeerID: uint64(t.peerID),
			Purpose:   message.PacketSyncPeer_PairRequest,
			Peer:      &message.PacketSyncPeer_PeerInfo{PeerID: uint64(t.localPeer.PeerID)},
			Endpoints: endpoints,
		}

		err := relayClient.Send(message.PacketType_SyncPeer, syncPeer)
		if err != nil {
			zap.L().Error("Relay message failed", zap.Error(err))
			return
		}

		// Try to establish P2P connection.
		t.pairCounter++
		retryInterval := time.Duration(math.Pow(float64(t.pairCounter), 2)) * time.Second
		t.nextPairAt = t.lastSendAt.Add(retryInterval)
	}
}

// OnUDPPacket implements the UDPPacketCallback interface. Which handles all UDP packets received from all tunnels.
func (t *Tunnel) OnUDPPacket(udpConn *net.UDPConn, data []byte) {
	if len(data) < constant.PacketHeaderSize {
		return
	}

	t.lastRecvAt = time.Now()

	nonce, packetType, peerID, payload, err := codec.Decode(data)
	if err != nil {
		zap.L().Error("Decode fragment failed", zap.Error(err))
		return
	}

	if logutil.IsEnablePeer() {
		zap.L().Debug("Receive packet", zap.Stringer("type", packetType), zap.Stringer("remote", udpConn.RemoteAddr()))
	}

	if packetType != message.PacketType_Fragment && packetType != message.PacketType_Discovery {
		zap.L().Warn("Unrecognized packet type", zap.Stringer("packet_type", packetType))
		return
	}

	decrypted, err := t.cipher.Decrypt(nil, uint64(nonce), nil, payload)
	if err != nil {
		zap.L().Error("Decrypt fragment failed", zap.Any("peer_id", peerID), zap.Error(err))
		return
	}

	switch packetType {
	case message.PacketType_Discovery:
		discovery, err := serde.Deserialize(message.PacketType_Discovery, decrypted)
		if err != nil {
			zap.L().Error("Deserialize fragment failed", zap.Stringer("packet_type", packetType), zap.Error(err))
			return
		}
		t.onDiscovery(udpConn, discovery.(*message.PacketDiscovery))

	case message.PacketType_Fragment:
		dataCopy := make([]byte, len(decrypted))
		copy(dataCopy, decrypted)
		t.callback.OnFragment(dataCopy)
	}
}

// ReachableEndpoint returns reachable endpoint, based on last discovery time
func (t *Tunnel) ReachableEndpoint() *Endpoint {
	val := t.endpoints.Load()
	if val == nil {
		return nil
	}

	reachable := val.([]*Endpoint)
	for i := range reachable {
		if time.Since(reachable[i].lastSeen) < 2*constant.DiscoveryDuration {
			return reachable[i]
		}
	}
	return nil
}

// cloneEndpoints does shallow copy of endpoints slice.
func (t *Tunnel) cloneEndpoints() []*Endpoint {
	val := t.endpoints.Load()
	var endpoints []*Endpoint
	if val != nil {
		endpoints = val.([]*Endpoint)
	}

	if len(endpoints) == 0 {
		return endpoints
	}

	// Clone the endpoints peerinfo slice and filter out the timeout endpoint.
	cloned := make([]*Endpoint, 0, len(endpoints))
	for i := range endpoints {
		if time.Since(endpoints[i].lastSeen) > 2*constant.DiscoveryDuration {
			continue
		}
		cloned = append(cloned, endpoints[i])
	}

	return cloned
}

func (t *Tunnel) storeEndpoints(endpoints []*Endpoint) {
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].latency < endpoints[j].latency
	})
	t.endpoints.Store(endpoints)
}

func (t *Tunnel) discovery() {
	if t.disco.Swap(true) {
		return
	}
	defer t.disco.Store(false)

	// The duration will be reset according to the current state of this connection
	nextDiscoTick := time.After(0)

	var currentEndpoints []string

	for {
		select {
		case eps := <-t.endpointsCh:
			currentEndpoints = eps
			nextDiscoTick = time.After(0)

		case <-nextDiscoTick:
			nextDiscoTick = time.After(constant.DiscoveryDuration)
			if len(currentEndpoints) < 1 {
				continue
			}

			endpoints := t.cloneEndpoints()
			byAddress := map[string]*Endpoint{}
			for _, e := range endpoints {
				byAddress[e.address] = e
			}

			// onDiscovery P2P connection via UDP address.
			for _, addr := range currentEndpoints {
				endpoint, found := byAddress[addr]
				if !found {
					conn, err := t.dialer.Dial("udp", addr)
					if err != nil {
						zap.L().Error("Dial remote address failed", zap.String("address", addr), zap.Error(err))
						continue
					}
					endpoint = newEndpoint(conn.(*net.UDPConn), time.Since(ZeroTime), ZeroTime, t)
					endpoint.serve()
					endpoints = append(endpoints, endpoint)
				}
				t.discoveryEndpoint(endpoint.udpConn)
				// Delete the address from the byAddress and make sure the remains should be pruned
				delete(byAddress, addr)
			}
			t.storeEndpoints(endpoints)

			// Prune the endpoints which are not belong to the current tunnel.
			for _, ep := range byAddress {
				ep.cancelFn()
			}

		case <-t.die:
			return
		}
	}
}

func (t *Tunnel) discoveryEndpoint(udpConn *net.UDPConn) {
	msg := &message.PacketDiscovery{
		SenderPeerID: uint64(t.localPeer.PeerID),
		Timestamp:    time.Now().UnixMicro(),
	}
	encoded, err := codec.EncodeMessage(message.PacketType_Discovery, t.cipher, t.localPeer.PeerID, msg)
	if err != nil {
		zap.L().Error("Encode discovery message failed", zap.Error(err))
		return
	}

	if logutil.IsEnablePeer() {
		zap.L().Debug("Send discovery message", zap.Stringer("remote", udpConn.RemoteAddr()))
	}

	_, err = udpConn.Write(encoded)
	if err != nil {
		zap.L().Error("Write discovery message failed", zap.Error(err))
	}
}

func (t *Tunnel) onDiscovery(udpConn *net.UDPConn, discovery *message.PacketDiscovery) {
	if logutil.IsEnablePeer() {
		zap.L().Debug("Discovery message", zap.Stringer("msg", discovery))
	}

	// onDiscovery message sent by self.

	if protocol.PeerID(discovery.SenderPeerID) != t.localPeer.PeerID {
		// Echo the discovery packet to the sender peer.
		data, err := codec.EncodeMessage(message.PacketType_Discovery, t.cipher, t.localPeer.PeerID, discovery)
		if err != nil {
			zap.L().Error("Encode discovery echo message failed", zap.Error(err))
			return
		}
		_, err = udpConn.Write(data)
		if err != nil {
			zap.L().Error("Write discovery echo message failed", zap.Error(err))
		}
	}

	val := t.endpoints.Load()
	var endpoints []*Endpoint
	if val != nil {
		endpoints = val.([]*Endpoint)
	}

	var found bool
	if len(endpoints) == 0 {
		zap.L().Warn("Endpoints empty")
		return
	}

	remoteAddr := udpConn.RemoteAddr().String()
	for i := range endpoints {
		if endpoints[i].address == remoteAddr {
			endpoints[i].latency = time.Since(time.UnixMicro(discovery.Timestamp)) / 2
			endpoints[i].lastSeen = time.Now()
			found = true
			break
		}
	}

	if found {
		return
	}

	// Found new discovery peer.
	latency := time.Since(time.UnixMicro(discovery.Timestamp)) / 2
	endpoint := newEndpoint(udpConn, latency, time.Now(), t)
	endpoints = append(endpoints, endpoint)
	endpoint.serve()
	t.storeEndpoints(endpoints)
}

// Close closes the current conn and clear status.
func (t *Tunnel) Close() {
	if t.closed.Swap(true) {
		return
	}
	zap.L().Info("Close tunnel", zap.Any("peer_id", t.peerID))

	endpoints := t.endpoints.Load()
	if endpoints != nil {
		for _, e := range endpoints.([]*Endpoint) {
			e.cancelFn()
		}
	}

	close(t.die)
}
