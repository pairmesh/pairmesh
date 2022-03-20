// Copyright 2022 PairMesh, Inc.
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
	"crypto/rand"
	"testing"

	"github.com/pairmesh/pairmesh/security"
	"github.com/stretchr/testify/assert"
)

func TestSecurityTransporter_Cipher(t *testing.T) {
	key, err := security.CipherSuite.GenerateKeypair(rand.Reader)
	assert.Nil(t, err)

	dhk, err := security.CipherSuite.DH(key.Private, key.Public)
	assert.Nil(t, err)

	fixSizeKey := [32]byte{}
	copy(fixSizeKey[:], dhk)
	trs := newSecurityTransporter(nil)
	cipher := security.CipherSuite.Cipher(fixSizeKey)
	trs.SetCipher(cipher)

	// The cipher is not a comparable object, so we encrypt/decrypt data to identify they are same.
	nonce := uint64(12345)
	text := []byte("hello world!")
	encrypted := cipher.Encrypt(nil, nonce, nil, text)
	decrypted, err := cipher.Decrypt(nil, nonce, nil, encrypted)
	assert.Nil(t, err)

	assert.Equal(t, text, decrypted)
}
