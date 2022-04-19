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
	"math/rand"
	"net"
	"time"

	"github.com/pairmesh/pairmesh/internal/codec"
	"github.com/pairmesh/pairmesh/message"

	"github.com/flynn/noise"
	"google.golang.org/protobuf/proto"
)

type (
	// Packet is the fundamental struct representing a network packet, with type and message
	Packet struct {
		Type    message.PacketType
		Message proto.Message
	}
)

func writePacketHelper(conn net.Conn, wp Packet, cipher noise.Cipher, codec *codec.RelayCodec, timeout time.Duration) error {
	data, err := proto.Marshal(wp.Message)
	if err != nil {
		return err
	}

	nonce := rand.Uint32()

	// Don't encrypt handshake messages.
	if wp.Type != message.PacketType_Handshake && wp.Type != message.PacketType_HandshakeAck {
		// The handshake is cleartext so the cipher will be nil.
		u64Nonce := uint64(nonce)
		if cipher != nil {
			// TODO: in-place encrypt to reduce the memory allocation
			data = cipher.Encrypt(nil, u64Nonce, nil, data)
		}
	}

	payload, err := codec.Encode(nonce, wp.Type, data)
	if err != nil {
		return err
	}

	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	_, err = conn.Write(payload)
	return err
}
