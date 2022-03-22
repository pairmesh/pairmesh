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
	"time"

	"github.com/pairmesh/pairmesh/protocol"
	"go.uber.org/zap"
)

// pullPeerGraph updates peers graph from portal service periodically.
func (d *NodeDriver) pullPeerGraph(ctx context.Context) {
	defer d.wg.Done()

	pullInterval := 1800 * time.Second
	tickInterval := 5 * time.Second
	uniqHash := ""
	pullTimer := time.After(0)
	tickTimer := time.After(0)
	for {
		select {
		case <-pullTimer:
			res, err := d.apiClient.PeerGraph(uniqHash)

			// Update the latest pullInterval from portal service.
			if res != nil && res.UpdateInterval > 0 {
				pullInterval = time.Duration(res.UpdateInterval) * time.Second
			}
			pullTimer = time.After(pullInterval)
			if err != nil {
				zap.L().Error("Retrieve the latest relay server information failed", zap.Error(err))
				continue
			}
			if res.NotModified {
				continue
			}

			// Update the latest unique hash.
			uniqHash = res.UniqueHash

			// Since response is already formatted so that res.Peers is sorted by peerID,
			// using binary search to find the serverID by matching peerID.
			primaryServerID := d.FindServerIDWithPeerID(res)

			if primaryServerID == 0 {
				zap.L().Error("Illegal peer graph response, cannot find primary server id")
				continue
			}

			var primaryServer protocol.RelayServer
			for _, r := range res.RelayServers {
				if r.ID == primaryServerID {
					primaryServer = r
					break
				}
			}
			if primaryServer.ID == 0 {
				zap.L().Error("Illegal peer graph response, cannot find primary server by id",
					zap.Any("server_id", primaryServerID))
				continue
			}

			d.mon.SetSTUNServer(primaryServer)
			d.rm.SetPrimaryServerID(primaryServerID)
			d.rm.Update(ctx, res.RelayServers)
			err = d.mm.Update(res.Networks, res.Peers)
			if err != nil {
				zap.L().Error("Error updating peers to network")
			}

		case <-tickTimer:
			d.rm.Tick(ctx)
			d.mm.Tick()
			tickTimer = time.After(tickInterval)

		case <-ctx.Done():
			zap.L().Info("Relay server information updater stopped")
			return
		}
	}
}

func (d *NodeDriver) FindServerIDWithPeerID(res *protocol.PeerGraphResponse) protocol.ServerID {
	var serverID protocol.ServerID

	if len(res.Peers) == 0 {
		return serverID
	}

	leftIndex := 0
	rightIndex := len(res.Peers) - 1
	var midIndex int
	var currPeer protocol.Peer

	for leftIndex <= rightIndex {
		midIndex = (leftIndex + rightIndex) / 2
		currPeer = res.Peers[midIndex]
		if currPeer.ID < d.peerID {
			leftIndex = midIndex + 1
		} else if currPeer.ID > d.peerID {
			rightIndex = midIndex - 1
		} else {
			serverID = currPeer.ServerID
			break
		}
	}

	return serverID
}

func (d *NodeDriver) SetPeerID(id protocol.PeerID) {
	d.peerID = id
}
