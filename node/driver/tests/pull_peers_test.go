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

package pull_peers_test

import (
	"testing"

	"github.com/pairmesh/pairmesh/node/driver"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/stretchr/testify/assert"
)

func setupDriver() *driver.MockDriver {
	var mockDrv = driver.MockDriver{
		NodeDriver: driver.NodeDriver{},
	}
	mockDrv.SetPeerID(protocol.PeerID(42))
	return &mockDrv
}

func TestFindServerIdWithPeerId(t *testing.T) {
	a := assert.New(t)

	mockDrv := setupDriver()

	peersArray := []protocol.Peer{
		{
			ID:       protocol.PeerID(1),
			ServerID: protocol.ServerID(101),
		},
		{
			ID:       protocol.PeerID(42),
			ServerID: protocol.ServerID(142),
		},
		{
			ID:       protocol.PeerID(50),
			ServerID: protocol.ServerID(150),
		},
	}
	resp := protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID := mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(142))
}

func TestFindServerIdWithPeerIdEmptyArray(t *testing.T) {
	a := assert.New(t)

	mockDrv := setupDriver()

	peersArray := []protocol.Peer{}
	resp := protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID := mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(0))
}

func TestFindServerIdWithPeerIdNotFound(t *testing.T) {
	a := assert.New(t)

	mockDrv := setupDriver()

	peersArray := []protocol.Peer{
		{
			ID:       protocol.PeerID(1),
			ServerID: protocol.ServerID(101),
		},
		{
			ID:       protocol.PeerID(50),
			ServerID: protocol.ServerID(103),
		},
	}
	resp := protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID := mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(0))
}

func TestFindServerIdWithPeerIdBigListFound(t *testing.T) {
	a := assert.New(t)

	mockDrv := setupDriver()

	// Test case peers: [(0, 100), (1, 101), ... (99, 199)]
	peersArray := []protocol.Peer{}
	for i := 0; i < 100; i++ {
		peersArray = append(peersArray, protocol.Peer{
			ID:       protocol.PeerID(i),
			ServerID: protocol.ServerID(i + 100),
		})
	}

	resp := protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID := mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(142))

	// Test case peers: [(42, 142), (43, 143), ... (141, 241)]
	peersArray = []protocol.Peer{}
	for i := 42; i < 142; i++ {
		peersArray = append(peersArray, protocol.Peer{
			ID:       protocol.PeerID(i),
			ServerID: protocol.ServerID(i + 100),
		})
	}
	resp = protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID = mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(142))

	// Test case peers: [(0, 100), (1, 101), ... (42, 142)]
	peersArray = []protocol.Peer{}
	for i := 0; i < 43; i++ {
		peersArray = append(peersArray, protocol.Peer{
			ID:       protocol.PeerID(i),
			ServerID: protocol.ServerID(i + 100),
		})
	}
	resp = protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID = mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(142))
}

func TestFindServerIdWithPeerIdBigListNotFound(t *testing.T) {
	a := assert.New(t)

	mockDrv := setupDriver()

	// Test case peers: [(0, 100) ... (41, 141), (43, 143)... (99, 199)]
	peersArray := []protocol.Peer{}
	for i := 0; i < 100; i++ {
		if i == 42 {
			continue
		}
		peersArray = append(peersArray, protocol.Peer{
			ID:       protocol.PeerID(i),
			ServerID: protocol.ServerID(i + 100),
		})
	}

	resp := protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID := mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(0))

	// Test case peers: [(43, 143), (43, 143), ... (141, 241)]
	peersArray = []protocol.Peer{}
	for i := 43; i < 142; i++ {
		peersArray = append(peersArray, protocol.Peer{
			ID:       protocol.PeerID(i),
			ServerID: protocol.ServerID(i + 100),
		})
	}
	resp = protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID = mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(0))

	// Test case peers: [(0, 100), (1, 101), ... (41, 141)]
	peersArray = []protocol.Peer{}
	for i := 0; i < 42; i++ {
		peersArray = append(peersArray, protocol.Peer{
			ID:       protocol.PeerID(i),
			ServerID: protocol.ServerID(i + 100),
		})
	}
	resp = protocol.PeerGraphResponse{
		Peers: peersArray,
	}
	serverID = mockDrv.FindServerIDWithPeerID(&resp)
	a.Equal(serverID, protocol.ServerID(0))
}
