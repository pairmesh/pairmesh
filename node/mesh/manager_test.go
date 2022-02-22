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
	"testing"

	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/stretchr/testify/assert"
)

func setupManager() *Manager {
	manager := Manager{}
	networks := []protocol.Network{}
	manager.networks.Store(networks)
	return &manager
}

func findPeerInNetwork(manager *Manager, networkID uint64, peerID uint64) bool {
	var networks []protocol.Network
	if v := manager.networks.Load(); v != nil {
		networks = v.([]protocol.Network)
	}
	for _, network := range networks {
		if network.ID == protocol.NetworkID(networkID) {
			for _, currID := range network.Peers {
				if currID == protocol.PeerID(peerID) {
					return true
				}
			}
		}
	}
	return false
}

func TestUpdateNetworkTopologyWithPeer(t *testing.T) {
	a := assert.New(t)
	manager := setupManager()
	networks := []protocol.Network{
		{
			ID:    protocol.NetworkID(1),
			Peers: []protocol.PeerID{1, 2, 3, 42, 100},
		},
		{
			ID:    protocol.NetworkID(2),
			Peers: []protocol.PeerID{1, 2},
		},
		{
			ID:    protocol.NetworkID(4),
			Peers: []protocol.PeerID{42},
		},
	}
	manager.networks.Store(networks)

	peerNetworks := []*message.PacketSyncPeer_Network{
		{ID: 2},
		{ID: 3},
		{ID: 4},
	}

	peerInfo := message.PacketSyncPeer_PeerInfo{
		PeerID:   42,
		Networks: peerNetworks,
	}

	manager.updateNetworkTopologyWithPeer(&peerInfo)
	// Peer 42 should be removed from network 1 and added to network 2, 3,
	// and skip 4 since it's already there:
	// NetworkID  |  PeerIDs
	//         1  |        1, 2, 3, 100
	//         2  |        1, 2, 42
	//         3  |       42
	//         4  |       42
	for _, networkID := range []uint64{1, 2, 3, 100} {
		a.True(findPeerInNetwork(manager, 1, networkID))
	}
	a.False(findPeerInNetwork(manager, 1, 42))
	for _, networkID := range []uint64{1, 2, 42} {
		a.True(findPeerInNetwork(manager, 2, networkID))
	}
	a.True(findPeerInNetwork(manager, 3, 42))
	a.True(findPeerInNetwork(manager, 4, 42))
}

func TestUpdateNetworkTopologyWithPeerEmptyManagerNetwork(t *testing.T) {
	a := assert.New(t)
	manager := setupManager()

	peerNetworks := []*message.PacketSyncPeer_Network{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	peerInfo := message.PacketSyncPeer_PeerInfo{
		PeerID:   42,
		Networks: peerNetworks,
	}

	manager.updateNetworkTopologyWithPeer(&peerInfo)
	// Peer 42 together with new network 1, 2, 3 should be added to manager.
	// NetworkID | PeerIDs
	//         1 |      42
	//         2 |      42
	//         3 |      42
	a.True(findPeerInNetwork(manager, 1, 42))
	a.True(findPeerInNetwork(manager, 2, 42))
	a.True(findPeerInNetwork(manager, 3, 42))
	a.False(findPeerInNetwork(manager, 1, 1))
}

func TestUpdateNetworkTopologyWithPeerEmptyPeerInfo(t *testing.T) {
	a := assert.New(t)
	manager := setupManager()

	networks := []protocol.Network{
		{
			ID:    protocol.NetworkID(1),
			Peers: []protocol.PeerID{1, 2, 3, 42, 100},
		},
		{
			ID:    protocol.NetworkID(2),
			Peers: []protocol.PeerID{42},
		},
		{
			ID:    protocol.NetworkID(3),
			Peers: []protocol.PeerID{41},
		},
	}
	manager.networks.Store(networks)

	peerNetworks := []*message.PacketSyncPeer_Network{}

	peerInfo := message.PacketSyncPeer_PeerInfo{
		PeerID:   42,
		Networks: peerNetworks,
	}

	manager.updateNetworkTopologyWithPeer(&peerInfo)
	// Peer 42 should be removed from network 1 and 2.
	// NetworkID | PeerIDs
	//         1 |       1, 2, 3, 100
	//         2 |       N/A
	//         3 |       41
	for _, networkID := range []uint64{1, 2, 3, 100} {
		a.True(findPeerInNetwork(manager, 1, networkID))
	}
	a.False(findPeerInNetwork(manager, 1, 42))
	a.False(findPeerInNetwork(manager, 2, 42))
	a.True(findPeerInNetwork(manager, 3, 41))
	a.False(findPeerInNetwork(manager, 3, 42))
}
