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
	"github.com/pairmesh/pairmesh/codec"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"google.golang.org/protobuf/proto"
	"inet.af/netaddr"
)

type (
	SessionCallback func(s *Session, typ message.PacketType, msg proto.Message) error
	SessionHandler  interface {
		On(typ message.PacketType, cb SessionCallback)
		Handle(s *Session, packet codec.RawPacket) error
	}

	ClientCallback func(s *Client, typ message.PacketType, msg proto.Message) error
	ClientHandler  interface {
		On(typ message.PacketType, cb ClientCallback)
		Handle(s *Client, packet codec.RawPacket) error
	}

	SessionManager interface {
		HeartbeatInterval() time.Duration
		DHKey() noise.DHKey
		RSAPublicKey() *rsa.PublicKey
		Session(peerID protocol.PeerID) *Session
	}

	PeerRouter interface {
		AddMERPPeerRoute(protocol.PeerID, string, *Client)
		RemoveMERPPeerRoute(protocol.PeerID, string, *Client)
	}

	WriteRequest struct {
		b      []byte // copied; ownership passed to receiver
		addr   netaddr.IPPort
		peerID protocol.PeerID
	}

	// ReadResult is the type sent by runMERPClient to receiveIPv4
	// when a MERP packet is available.
	ReadResult struct {
		region string
		n      int // length of data received
		src    protocol.PeerID

		// copyBuf is called to copy the data to dst.  It returns how
		// much data was copied, which will be n if dst is large
		// enough. copyBuf can only be called once.
		// If copyBuf is nil, that's a signal from the sender to ignore
		// this message.
		copyBuf func(dst []byte) int
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

	keepAlive = 60 * time.Second
)
