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
	"fmt"

	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/protocol"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func (d *NodeDriver) OnForward(_ *relay.Client, typ message.PacketType, msg proto.Message) error {
	forward := msg.(*message.PacketForward)

	if logutil.IsEnablePeer() {
		zap.L().Debug("Receive packet", zap.Stringer("type", typ), zap.Stringer("payload", forward))
	}

	if protocol.PeerID(forward.DstPeerID) != d.peerID {
		return errors.Errorf("destination peer id %d not match", forward.DstPeerID)
	}

	peer := d.mm.Peer(protocol.PeerID(forward.SrcPeerID))
	if peer == nil {
		return errors.Errorf("peer closed (peerID %d)", forward.SrcPeerID)
	}

	t := peer.Tunnel()
	if t == nil {
		return fmt.Errorf("no peer catchup ack received (peer id: %d)", forward.SrcPeerID)
	}

	decrypted, err := t.Cipher().Decrypt(nil, uint64(forward.Nonce), nil, forward.Fragment)
	if err != nil {
		return errors.WithMessage(err, "decrypt fragment failed")
	}

	// Write the forwarded message into the device pipeline.
	dataCopy := make([]byte, len(decrypted))
	copy(dataCopy, decrypted)
	d.chDevWrite <- dataCopy

	return nil
}

func (d *NodeDriver) OnSyncPeer(_ *relay.Client, _ message.PacketType, msg proto.Message) error {
	syncPeer := msg.(*message.PacketSyncPeer)
	if protocol.PeerID(syncPeer.DstPeerID) != d.peerID {
		return errors.Errorf("destination peer id %d not match", syncPeer.DstPeerID)
	}
	zap.L().Debug("On sync peer", zap.Stringer("purpose", syncPeer.Purpose), zap.Stringer("msg", syncPeer))

	switch syncPeer.Purpose {
	case message.PacketSyncPeer_Catchup:
		// If the message.PairRequest is called by the peer side, we may have no
		// idea about its primary relay server.
		if ps := syncPeer.Peer.PrimaryServer; ps != nil {
			relayServer := protocol.RelayServer{
				ID:        protocol.ServerID(ps.ID),
				Name:      ps.Name,
				Region:    ps.Region,
				Host:      ps.Host,
				Port:      int(ps.Port),
				PublicKey: ps.PublicKey,
			}
			d.rm.AddServer(relayServer)
		}
		err := d.mm.PeerCatchup(syncPeer)
		if err != nil {
			zap.L().Error("Peer catchup failed", zap.Error(err))
			return err
		}

	case message.PacketSyncPeer_CatchupAck:
		d.mm.PeerCatchupAck(syncPeer)

	case message.PacketSyncPeer_PairRequest:
		d.mm.PeerEndpoints(syncPeer)

		peer := d.mm.Peer(protocol.PeerID(syncPeer.Peer.PeerID))

		// Send the pair request packet to the remote peer.
		relayClient := d.rm.RelayServerClient(peer.PrimaryServerID())
		if relayClient == nil {
			// Fallback to the local peer primary client
			relayClient = d.rm.PrimaryRelayServerClient()
		}
		if relayClient == nil {
			zap.L().Error("Cannot find the relay server for remote peer", zap.Any("peer_id", syncPeer.Peer.PeerID))
			return nil
		}

		// If the external addr doesn't be detected, we just send the local interface's addresses to
		// the remote peer. And the newly detected endpoints will send to the remote while external
		// address changed (see: events_monitor.go).
		var endpoints []string
		if externalAddr := d.externalAddr.Load(); externalAddr != "" {
			endpoints = append(endpoints, externalAddr)
		}
		for _, l := range d.localAddresses() {
			endpoints = append(endpoints, fmt.Sprintf("%s:%d", l, d.config.Port))
		}

		syncPeerRes := &message.PacketSyncPeer{
			DstPeerID: syncPeer.Peer.PeerID,
			Purpose:   message.PacketSyncPeer_PairResponse,
			Peer: &message.PacketSyncPeer_PeerInfo{
				PeerID: syncPeer.DstPeerID,
			},
			Endpoints: endpoints,
		}

		err := relayClient.Send(message.PacketType_SyncPeer, syncPeerRes)
		if err != nil {
			zap.L().Error("Send the pair response failed", zap.Error(err))
		}

	case message.PacketSyncPeer_PairResponse, message.PacketSyncPeer_EndpointsChanged:
		d.mm.PeerEndpoints(syncPeer)
	}

	return nil
}

func (d *NodeDriver) OnProbeResponse(_ *relay.Client, _ message.PacketType, msg proto.Message) error {
	probe := msg.(*message.PacketProbeResponse)

	zap.L().Debug("On probe result", zap.Stringer("msg", probe))
	d.mm.ProbeResult(probe)
	return nil
}
