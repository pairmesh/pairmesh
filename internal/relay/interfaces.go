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
	"crypto/rsa"
	"time"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/internal/codec"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"google.golang.org/protobuf/proto"
)

type (
	// SessionCallback is server side callback function when there is certain type of message arrives
	SessionCallback func(s *Session, typ message.PacketType, msg proto.Message) error

	// SessionHandler is the server side handler interface that features On and Handle functions
	SessionHandler interface {
		On(typ message.PacketType, cb SessionCallback)
		Handle(s *Session, packet codec.RawPacket) error
	}

	// ClientCallback is client side callback function when there is certain type of message arrives
	ClientCallback func(s *Client, typ message.PacketType, msg proto.Message) error

	// ClientHandler is the client side handler interface that features On and Handle functions
	ClientHandler interface {
		On(typ message.PacketType, cb ClientCallback)
		Handle(s *Client, packet codec.RawPacket) error
	}

	// SessionLifetimeHook is hook interface to specifically handle OnSessionHandshake and OnSessionClosed
	SessionLifetimeHook interface {
		OnSessionHandshake(ses *Session)
		OnSessionClosed(ses *Session)
	}

	// SessionManager is manager interface to handle session metadata
	SessionManager interface {
		HeartbeatInterval() time.Duration
		DHKey() noise.DHKey
		RSAPublicKey() *rsa.PublicKey
		Session(peerID protocol.PeerID) *Session
	}

	// PeerRouter is router interface that adds and removes peer route
	PeerRouter interface {
		AddMERPPeerRoute(protocol.PeerID, string, *Client)
		RemoveMERPPeerRoute(protocol.PeerID, string, *Client)
	}
)

// MaxPacketSize is the maximum size of a packet sent over MERP.
// (This only includes the data bytes visible to nicesock, not
// including its on-wire framing overhead)
const (
	MaxPacketSize = 64 << 10

	// ProtocolVersion is bumped whenever there's a wire-incompatible change.
	//   * version 1: received packets have src addrs in frameRecvPacket at beginning
	ProtocolVersion = 1
)
