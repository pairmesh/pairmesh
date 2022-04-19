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
	"testing"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/stretchr/testify/assert"
)

type RandomInc byte

func (r *RandomInc) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(*r)
		*r = (*r) + 1
	}
	return len(p), nil
}

func TestEncode(t *testing.T) {
	key, err := noise.DH25519.GenerateKeypair(new(RandomInc))
	assert.Nil(t, err)

	// The encrypt/decrypt cipher of tunnels are the same.
	sharedKey, err := noise.DH25519.DH(key.Private, key.Public)
	assert.Nil(t, err)

	fixSizeKey := [32]byte{}
	copy(fixSizeKey[:], sharedKey)
	cipher := noise.CipherChaChaPoly.Cipher(fixSizeKey)

	peerID := protocol.PeerID(1234567)
	data := []byte{1, 2, 3, 4, 5}

	encoded := Encode(message.PacketType_Handshake, cipher, peerID, data)
	nonce, typ, pid, payload, err := Decode(encoded)
	assert.Nil(t, err)

	assert.Equal(t, typ, message.PacketType_Handshake)
	assert.Equal(t, pid, peerID)

	decrypted, err := cipher.Decrypt(nil, uint64(nonce), nil, payload)
	assert.Nil(t, err)
	assert.Equal(t, decrypted, data)
}
