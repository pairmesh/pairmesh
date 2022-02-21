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

package mesh

import (
	"net"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/node/device"
	"github.com/pairmesh/pairmesh/node/mesh/peer"
	"github.com/pairmesh/pairmesh/node/mesh/tunnel"
	"github.com/pairmesh/pairmesh/node/mesh/types"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/protocol"

	"github.com/flynn/noise"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"inet.af/netaddr"
)

// Manager is used to manage all tunnels connected to the current node.
type Manager struct {
	dialer   *net.Dialer
	self     types.LocalPeer
	rm       *relay.Manager
	router   device.Router
	callback tunnel.FragmentCallback
	networks atomic.Value // An atomic value of: []protocol.Network

	// Peers table.
	mu    sync.RWMutex
	peers map[protocol.PeerID]*peer.Peer
	index map[string]*peer.Peer // index by address.

	// Cache the summary
	lastChangedAt time.Time
	cachedSummary *Summary
}

func NewManager(dialer *net.Dialer, self types.LocalPeer, callback tunnel.FragmentCallback, rm *relay.Manager, router device.Router) *Manager {
	m := &Manager{
		dialer:   dialer,
		self:     self,
		rm:       rm,
		router:   router,
		callback: callback,

		index: map[string]*peer.Peer{},
		peers: map[protocol.PeerID]*peer.Peer{},
	}
	return m
}

func (m *Manager) markChanged() {
	m.lastChangedAt = time.Now()
}

func (m *Manager) Tunnel(dest string) *tunnel.Tunnel {
	// TODO: support subnet NAT.
	m.mu.RLock()
	p, found := m.index[dest]
	m.mu.RUnlock()
	if !found {
		return nil
	}

	return p.Tunnel()
}

// Peer returns the communication Tunnel corresponding to the destination.
func (m *Manager) Peer(peerID protocol.PeerID) *peer.Peer {
	m.mu.RLock()
	p, found := m.peers[peerID]
	m.mu.RUnlock()
	if !found {
		return nil
	}
	return p
}

// Summarize returns the mesh network summary.
func (m *Manager) Summarize() *Summary {
	if m.cachedSummary != nil && m.cachedSummary.LastChangedAt == m.lastChangedAt {
		return m.cachedSummary
	}

	var myDevices []Device
	selfUserID := m.self.UserID
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.peers {
		peerInfo := p.PeerInfo()
		if peerInfo.UserID == selfUserID {
			state := StateRelay
			if t := p.Tunnel(); t != nil && t.ReachableEndpoint() != nil {
				state = StateP2P
			}
			myDevices = append(myDevices, Device{
				Name:   peerInfo.Name,
				IPv4:   peerInfo.IPv4,
				Status: state,
			})
		}
	}

	// Networks
	var myNetworks []Network
	var networks []protocol.Network
	if v := m.networks.Load(); v != nil {
		networks = v.([]protocol.Network)
	}
	for _, n := range networks {
		network := Network{
			ID:   uint64(n.ID),
			Name: n.Name,
		}
		for _, peerID := range n.Peers {
			p, ok := m.peers[peerID]
			if !ok {
				continue
			}
			state := StateRelay
			if t := p.Tunnel(); t != nil && t.ReachableEndpoint() != nil {
				state = StateP2P
			}
			peerInfo := p.PeerInfo()
			network.Devices = append(network.Devices, Device{
				Name:   peerInfo.Name,
				IPv4:   peerInfo.IPv4,
				Status: state,
			})
		}
		myNetworks = append(myNetworks, network)
	}

	summary := &Summary{
		LastChangedAt: m.lastChangedAt,
		MyDevices:     myDevices,
		Networks:      myNetworks,
	}
	m.cachedSummary = summary

	return summary
}

func (m *Manager) Tick() {
	m.probePeers()
}

// Update updates the latest networks and peers information.
func (m *Manager) Update(latestNetworks []protocol.Network, latestPeers []protocol.Peer) error {
	if logutil.IsEnableRelay() {
		zap.L().Debug("Update peers", zap.Any("peers", latestPeers))
	}

	// Belongs to all the networks
	var selfNetworks []*message.PacketSyncPeer_Network
	for _, n := range latestNetworks {
		selfNetworks = append(selfNetworks, &message.PacketSyncPeer_Network{
			Name: n.Name,
			ID:   uint64(n.ID),
		})
	}

	m.networks.Store(latestNetworks)
	m.self.Networks = selfNetworks

	routerCfg := &device.Config{LocalAddress: m.self.VIPv4}

	// NOTE: We must merge the peer information before sending probe request to
	// the relay server because we may receive the probe response when the `Update`
	// call doesn't finish. If so, the handler of probe response cannot find the
	// peer information from the peers set.
	//
	// Merge the latest peers information with previous existing.
	m.mu.Lock()
	peers := map[protocol.PeerID]*peer.Peer{}
	for _, latestPeer := range latestPeers {
		p, ok := m.peers[latestPeer.ID]
		if !ok {
			p = peer.New(latestPeer)
		}
		peers[latestPeer.ID] = p

		// Skip the current device.
		if p.ID() != m.self.PeerID {
			addr, err := netaddr.ParseIP(latestPeer.IPv4)
			if err != nil {
				return errors.WithMessage(err, "parse ipv4 address in Update")
			}
			prefix := netaddr.IPPrefixFrom(addr, 32)
			routerCfg.Routes = append(routerCfg.Routes, prefix)
		}
	}
	index := map[string]*peer.Peer{}
	for _, p := range peers {
		index[p.IPv4()] = p
	}

	// Close outdated remote peers.
	for _, p := range m.peers {
		_, ok := peers[p.ID()]
		if !ok {
			p.Close()
		}
	}

	// Update the local peers cache.
	m.peers = peers
	m.index = index
	m.mu.Unlock()

	// Update the router configuration to allow traffics to the remote peers.
	m.router.Set(routerCfg)

	// TODO: check peers changed more accurately.
	m.markChanged()

	m.probePeers()

	return nil
}

func (m *Manager) probePeers() {
	// Group peers by relay server ID.
	m.mu.RLock()
	probeGroup := map[protocol.ServerID][]uint64{}
	for _, p := range m.peers {
		// Skip the current device.
		if p.ID() == m.self.PeerID {
			continue
		}
		// Skip the peers recently probed.
		if !p.IsNeedProbe() {
			continue
		}
		serverID := p.PrimaryServerID()
		probeGroup[serverID] = append(probeGroup[serverID], uint64(p.ID()))
	}
	m.mu.RUnlock()

	// Probe all peers.
	for serverID, peerIDs := range probeGroup {
		client := m.rm.RelayServerClient(serverID)
		if client == nil {
			continue
		}
		probeRequest := &message.PacketProbeRequest{
			Peers: peerIDs,
		}

		probeRequestAt := time.Now()
		err := client.Send(message.PacketType_ProbeRequest, probeRequest)
		if err != nil {
			continue
		}

		// Update the probe request timestamp.
		m.mu.Lock()
		for _, pid := range peerIDs {
			peerID := protocol.PeerID(pid)
			m.peers[peerID].SetLastProbeRequestAt(probeRequestAt)
		}
		m.mu.Unlock()
	}
}

// ProbeResult handles the probe response from relay server.
func (m *Manager) ProbeResult(probe *message.PacketProbeResponse) {
	var catchupPeers []*peer.Peer

	m.mu.RLock()
	// Online peers which established connection with its primary relay server.
	for _, pid := range probe.OnlinePeers {
		peerID := protocol.PeerID(pid)
		p, found := m.peers[peerID]
		if !found {
			zap.L().Warn("Peer not found", zap.Any("peer_id", peerID))
			continue
		}

		// Update the peer probe status and add to catchup list if need.
		p.SetProbeStatus(true)
		if p.IsNeedCatchup() {
			catchupPeers = append(catchupPeers, p)
		}
	}

	// Offline peers which didn't establish connection with its primary relay server.
	for _, pid := range probe.OfflinePeers {
		peerID := protocol.PeerID(pid)
		p, found := m.peers[peerID]
		if !found {
			zap.L().Warn("Peer not found", zap.Any("peer_id", peerID))
			continue
		}
		p.SetProbeStatus(false)
	}
	m.mu.RUnlock()

	if len(catchupPeers) == 0 {
		return
	}

	pc := m.rm.PrimaryRelayServerClient()
	if pc == nil {
		zap.L().Error("Primary relay server doesn't connected")
		return
	}

	rs := pc.RelayServer()
	peerInfo := &message.PacketSyncPeer_PeerInfo{
		UserID:    uint64(m.self.UserID),
		PeerID:    uint64(m.self.PeerID),
		IPv4:      m.self.VIPv4.String(),
		Name:      m.self.Name,
		PublicKey: m.self.Key.Public,
		PrimaryServer: &message.PacketSyncPeer_RelayServer{
			ID:        uint64(rs.ID),
			Name:      rs.Name,
			Region:    rs.Region,
			Host:      rs.Host,
			Port:      uint32(rs.Port),
			PublicKey: rs.PublicKey,
		},
		Networks: m.self.Networks,
	}

	for _, catchupPeer := range catchupPeers {
		client := m.rm.RelayServerClient(catchupPeer.PrimaryServerID())
		if client == nil {
			continue
		}

		syncPeer := &message.PacketSyncPeer{
			DstPeerID: uint64(catchupPeer.ID()),
			Purpose:   message.PacketSyncPeer_Catchup,
			Peer:      peerInfo,
		}

		err := client.Send(message.PacketType_SyncPeer, syncPeer)
		if err != nil {
			zap.L().Error("Send catchup failed", zap.Error(err))
			continue
		}
		catchupPeer.SetLastSendCatchupAt(time.Now())
	}
}

func (m *Manager) PeerCatchup(syncPeer *message.PacketSyncPeer) error {
	peerInfo := syncPeer.Peer
	if peerInfo == nil {
		// This is tolerable scenario. We just return early
		zap.L().Debug("Received request to do PeerCatchup but peerInfo in packet is empty")
		return nil
	}

	routerCfg := &device.Config{LocalAddress: m.self.VIPv4}

	// Update the latest peer information.
	m.mu.Lock()
	peerID := protocol.PeerID(peerInfo.PeerID)
	p, ok := m.peers[peerID]
	if !ok {
		p = peer.New(protocol.Peer{
			ID:       protocol.PeerID(peerInfo.PeerID),
			UserID:   protocol.UserID(peerInfo.UserID),
			Name:     peerInfo.Name,
			IPv4:     peerInfo.IPv4,
			ServerID: protocol.ServerID(peerInfo.PrimaryServer.ID),
			Active:   true,
		})
		addr, err := netaddr.ParseIP(peerInfo.IPv4)
		if err != nil {
			return errors.WithMessage(err, "parse ipv4 address in PeerCatchup")
		}
		prefix := netaddr.IPPrefixFrom(addr, 32)
		routerCfg.Routes = append(routerCfg.Routes, prefix)
		m.peers[peerID] = p
		m.index[peerInfo.IPv4] = p

		// We treat the newly added peer as probed one.
		p.SetProbeStatus(true)
	}
	m.mu.Unlock()

	// The encrypt/decrypt cipher of tunnels are the same.
	sharedKey, err := noise.DH25519.DH(m.self.Key.Private, peerInfo.PublicKey)
	if err != nil {
		zap.L().Error("Exchange shared key failed", zap.Error(err))
		return err
	}
	fixSizeKey := [32]byte{}
	copy(fixSizeKey[:], sharedKey)
	cipher := noise.CipherChaChaPoly.Cipher(fixSizeKey)
	rcGetter := m.relayClientGetter(protocol.ServerID(peerInfo.PrimaryServer.ID))

	p.SetCatchupAt(time.Now())
	p.SetTunnel(tunnel.New(rcGetter, m.dialer, m.self, protocol.PeerID(peerInfo.PeerID), m.callback, cipher))

	// Add to router if the peer is newly added.
	if len(routerCfg.Routes) > 0 {
		m.router.Add(routerCfg)
	}

	// Update the network topology information.
	// FIXME: remove peer from existing network if the peerInfo.Network is empty/nil.
	if len(peerInfo.Networks) > 0 {
		var networks []protocol.Network
		if n := m.networks.Load(); n != nil {
			networks = n.([]protocol.Network)
		}

		// Put local networks in a Hashmap for network ID matching
		localNwkMap := make(map[protocol.NetworkID]protocol.Network)
		for _, localNwk := range networks {
			localNwkMap[localNwk.ID] = localNwk
		}
		// Start network and peer matching
		for _, network := range peerInfo.Networks {
			exist, ok := localNwkMap[protocol.NetworkID(network.ID)]
			if ok {
				var peerFound bool
				for _, pid := range exist.Peers {
					if peerID == pid {
						peerFound = true
						break
					}
				}
				if !peerFound {
					// Add peer to existing network
					exist.Peers = append(exist.Peers, peerID)
				}
			} else {
				networks = append(networks, protocol.Network{
					ID:    protocol.NetworkID(network.ID),
					Name:  network.Name,
					Peers: []protocol.PeerID{peerID},
				})
			}
		}
		m.networks.Store(networks)
	}

	relayClient := m.rm.RelayServerClient(protocol.ServerID(peerInfo.PrimaryServer.ID))
	if relayClient == nil {
		zap.L().Error("The remote peer primary relay server doesn't connect", zap.Any("peer_id", peerID))
		return err
	}

	ack := &message.PacketSyncPeer{
		DstPeerID: uint64(peerID),
		Purpose:   message.PacketSyncPeer_CatchupAck,
		Peer: &message.PacketSyncPeer_PeerInfo{
			PeerID:    uint64(m.self.PeerID),
			PublicKey: m.self.Key.Public,
		},
	}
	err = relayClient.Send(message.PacketType_SyncPeer, ack)
	if err != nil {
		zap.L().Error("Response catchup ack failed", zap.Error(err))
		return err
	}

	return nil
}

func (m *Manager) PeerCatchupAck(syncPeer *message.PacketSyncPeer) {
	peerInfo := syncPeer.Peer
	if peerInfo == nil {
		return
	}

	m.mu.Lock()
	peerID := protocol.PeerID(syncPeer.Peer.PeerID)
	p, ok := m.peers[peerID]
	m.mu.Unlock()

	if !ok {
		zap.L().Error("Peer not found", zap.Any("peer_id", peerID))
		return
	}

	// The encrypt/decrypt cipher of tunnels are the same.
	sharedKey, err := noise.DH25519.DH(m.self.Key.Private, peerInfo.PublicKey)
	if err != nil {
		zap.L().Error("Exchange shared key failed", zap.Error(err))
		return
	}
	fixSizeKey := [32]byte{}
	copy(fixSizeKey[:], sharedKey)
	cipher := noise.CipherChaChaPoly.Cipher(fixSizeKey)
	rcGetter := m.relayClientGetter(p.PrimaryServerID())

	p.SetTunnel(tunnel.New(rcGetter, m.dialer, m.self, protocol.PeerID(peerInfo.PeerID), m.callback, cipher))
	p.SetCatchupAt(time.Now())
}

func (m *Manager) PeerEndpoints(syncPeer *message.PacketSyncPeer) {
	peerInfo := syncPeer.Peer
	if peerInfo == nil {
		return
	}

	m.mu.RLock()
	peerID := protocol.PeerID(peerInfo.PeerID)
	p, ok := m.peers[peerID]
	m.mu.RUnlock()

	if !ok {
		zap.L().Error("Peer not found", zap.Any("peer_id", peerID))
		return
	}

	t := p.Tunnel()
	if t == nil {
		return
	}

	t.SetRemoteEndpoints(syncPeer.Endpoints)
}

// SyncEndpoints synchronize the latest endpoints to the remote peers which had established
// P2P connection with the local peer.
func (m *Manager) SyncEndpoints(endpoints []string) {
	var needSyncPeers []*peer.Peer

	m.mu.RLock()
	for _, p := range m.peers {
		t := p.Tunnel()
		if t == nil {
			continue
		}
		t.SetLocalEndpoints(endpoints)
		if !t.IsDisco() {
			continue
		}
		needSyncPeers = append(needSyncPeers, p)
	}
	m.mu.RUnlock()

	for _, p := range needSyncPeers {
		// Send the pair request packet to the remote peer.
		relayClient := m.rm.RelayServerClient(p.PrimaryServerID())
		if relayClient == nil {
			// Fallback to the local peer primary client
			relayClient = m.rm.PrimaryRelayServerClient()
		}
		if relayClient == nil {
			zap.L().Error("Cannot find the relay server for remote peer", zap.Any("peer_id", p.ID()))
			continue
		}

		syncPeer := &message.PacketSyncPeer{
			DstPeerID: uint64(p.ID()),
			Purpose:   message.PacketSyncPeer_EndpointsChanged,
			Peer:      &message.PacketSyncPeer_PeerInfo{PeerID: uint64(m.self.PeerID)},
			Endpoints: endpoints,
		}

		err := relayClient.Send(message.PacketType_SyncPeer, syncPeer)
		if err != nil {
			zap.L().Error("Send endpoints changed sync message failed", zap.Error(err))
		}
	}
}

func (m *Manager) relayClientGetter(serverID protocol.ServerID) tunnel.RelayClientGetter {
	return func() *relay.Client {
		// Send the pair request packet to the remote peer.
		relayClient := m.rm.RelayServerClient(serverID)
		if relayClient == nil {
			// Fallback to the local peer primary client
			relayClient = m.rm.PrimaryRelayServerClient()
		}
		return relayClient
	}
}
