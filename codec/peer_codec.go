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

package codec

import (
	"encoding/binary"
	"errors"
	"math/rand"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"google.golang.org/protobuf/proto"
)

// EncodeMessage encodes a given message with certain type, cipher and peerID together to make a byte slice
func EncodeMessage(typ message.PacketType, cipher noise.Cipher, peerID protocol.PeerID, msg proto.Message) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return Encode(typ, cipher, peerID, data), nil
}

// Encode encodes byte slice payload into binary format
func Encode(typ message.PacketType, cipher noise.Cipher, peerID protocol.PeerID, payload []byte) []byte {
	nonce := rand.Uint32()
	encrypted := cipher.Encrypt(nil, uint64(nonce), nil, payload)

	// Packet format:
	// | nonce(4bytes) | type(2bytes) | peerID(8bytes) | payload |
	buffer := make([]byte, len(encrypted)+constant.FragmentHeaderSize)
	binary.BigEndian.PutUint32(buffer[:constant.HeaderNonceSize], nonce)
	newtyp := uint16(typ) ^ uint16(nonce&(1<<16-1))
	binary.BigEndian.PutUint16(buffer[constant.HeaderNonceSize:constant.PacketHeaderSize], newtyp)
	newpid := uint64(peerID) ^ (uint64(nonce) | (^uint64(nonce) << 32))
	binary.BigEndian.PutUint64(buffer[constant.FragmentHeaderSize-8:constant.FragmentHeaderSize], newpid)
	copy(buffer[constant.FragmentHeaderSize:], encrypted)

	return buffer
}

// Decode decodes byte slice into formatted packet and peer id
func Decode(data []byte) (uint32, message.PacketType, protocol.PeerID, []byte, error) {
	if len(data) < constant.FragmentHeaderSize {
		return 0, message.PacketType(0), protocol.PeerID(0), nil, errors.New("invalid fragment data")
	}

	nonce := binary.BigEndian.Uint32(data[:constant.HeaderNonceSize])
	oldtyp := binary.BigEndian.Uint16(data[constant.HeaderNonceSize:constant.PacketHeaderSize])
	newtyp := message.PacketType(oldtyp ^ uint16(nonce&(1<<16-1)))
	oldpid := binary.BigEndian.Uint64(data[constant.FragmentHeaderSize-8 : constant.FragmentHeaderSize])
	newpid := protocol.PeerID((oldpid) ^ (uint64(nonce) | (^uint64(nonce) << 32)))

	return nonce, newtyp, newpid, data[constant.FragmentHeaderSize:], nil
}
