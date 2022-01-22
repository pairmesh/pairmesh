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

package server

import (
	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type (
	callbacks struct {
		server *relay.Server
	}
)

func registerCallback(server *relay.Server) {
	callback := &callbacks{server: server}
	server.Handler().On(message.PacketType_Forward, callback.onForward)
	server.Handler().On(message.PacketType_SyncPeer, callback.onSyncPeer)
	server.Handler().On(message.PacketType_ProbeRequest, callback.onProbe)
}

func (h *callbacks) onForward(self *relay.Session, _ message.PacketType, msg proto.Message) error {
	forward := msg.(*message.PacketForward)
	if logutil.IsEnableRelay() {
		zap.L().Debug("On forward", zap.Stringer("msg", forward), zap.Any("peer_id", self.PeerID()))
	}

	peerSession := h.server.Session(protocol.PeerID(forward.DstPeerID))
	if peerSession == nil {
		zap.L().Error("Peer session not found", zap.Any("peer_id", forward.DstPeerID))
		return nil
	}

	return peerSession.Send(message.PacketType_Forward, forward)
}

func (h *callbacks) onProbe(self *relay.Session, _ message.PacketType, msg proto.Message) error {
	probe := msg.(*message.PacketProbeRequest)
	if logutil.IsEnableRelay() {
		zap.L().Debug("On probe", zap.Stringer("msg", probe), zap.Any("peer_id", self.PeerID()))
	}

	res := &message.PacketProbeResponse{}
	for _, peerID := range probe.Peers {
		peerSession := h.server.Session(protocol.PeerID(peerID))
		if peerSession == nil {
			res.OfflinePeers = append(res.OfflinePeers, peerID)
		} else {
			res.OnlinePeers = append(res.OnlinePeers, peerID)
		}
	}

	return self.Send(message.PacketType_ProbeResponse, res)
}

func (h *callbacks) onSyncPeer(self *relay.Session, _ message.PacketType, msg proto.Message) error {
	syncPeer := msg.(*message.PacketSyncPeer)
	if logutil.IsEnableRelay() {
		zap.L().Debug("On sync peer", zap.Stringer("msg", syncPeer), zap.Any("peer_id", self.PeerID()))
	}

	peerSession := h.server.Session(protocol.PeerID(syncPeer.DstPeerID))
	if peerSession == nil {
		zap.L().Error("Peer session not found", zap.Any("peer_id", syncPeer.DstPeerID))
		return nil
	}

	return peerSession.Send(message.PacketType_SyncPeer, syncPeer)
}
