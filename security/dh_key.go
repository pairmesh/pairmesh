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

package security

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/flynn/noise"
)

const keySize = 32

// DHKeyBytes is the alias of a byte slice, representing DHKey raw data
type DHKeyBytes [keySize]byte

// DHPrivate represents a curve25519 private key.
type DHPrivate struct{ DHKeyBytes }

// DHPublic represents a curve25519 public key.
type DHPublic struct{ DHKeyBytes }

// B32 returns k as the *[32]byte type that's used by the
// golang.org/x/crypto packages. This allocates; it might
// not be appropriate for performance-sensitive paths.
func (k DHKeyBytes) B32() *[keySize]byte { return (*[keySize]byte)(&k) }

// IsZero reports whether the DHPrivate p is the zero value.
func (k DHKeyBytes) IsZero() bool { return k == DHKeyBytes{} }
func (k DHKeyBytes) String() string {
	return base64.StdEncoding.EncodeToString(k[:])
}

// ShortString returns the PairMesh conventional debug representation
// of a public key: the first five base64 digits of the key, in square
// brackets.
func (k DHKeyBytes) ShortString() string {
	return "[" + base64.StdEncoding.EncodeToString(k[:])[:5] + "]"
}

// MarshalText encodes the key into Marshal text and returns it
func (k DHKeyBytes) MarshalText() ([]byte, error) {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(k)))
	base64.StdEncoding.Encode(buf, k[:])
	return buf, nil
}

// Bytes copies the key to a buffer and returns the buffer
func (k DHKeyBytes) Bytes() []byte {
	b := make([]byte, keySize)
	copy(b, k[:])
	return b
}

// Len returns length of the key
func (k DHKeyBytes) Len() int {
	return len(k)
}

// UnmarshalText decodes the key into given txt field
func (k *DHKeyBytes) UnmarshalText(txt []byte) error {
	if *k != (DHKeyBytes{}) {
		return errors.New("refusing to unmarshal into non-zero key.DHPublic")
	}
	n, err := base64.StdEncoding.Decode(k[:], txt)
	if err != nil {
		return err
	}
	if n != keySize {
		return fmt.Errorf("short decode of %d; want keySize", n)
	}
	return nil
}

// NewDHPrivate returns a new private key.
func NewDHPrivate(s []byte) DHPrivate {
	if len(s) > keySize {
		return DHPrivate{}
	}
	var x DHKeyBytes
	copy(x[:], s)
	return DHPrivate{DHKeyBytes: x}
}

// NewDHPublic returns a new public key.
func NewDHPublic(s []byte) DHPublic {
	if len(s) > keySize {
		return DHPublic{}
	}
	var x DHKeyBytes
	copy(x[:], s)
	return DHPublic{DHKeyBytes: x}
}

// DHKey a noise.DHkey wrapper for PairMesh
type DHKey struct {
	Public  DHPublic  `json:"public"`
	Private DHPrivate `json:"private"`
}

// FromNoiseDHKey generate a DHKey from noise.DHKey
func FromNoiseDHKey(nKey noise.DHKey) *DHKey {
	return &DHKey{
		Public:  NewDHPublic(nKey.Public),
		Private: NewDHPrivate(nKey.Private),
	}
}

// IsZero returns whether DH public key is zero
func (k *DHKey) IsZero() bool {
	return k.Public.IsZero()
}

// Equals returns whether a DHKey equals to another one
func (k *DHKey) Equals(k2 *DHKey) bool {
	if k2 == nil {
		return false
	}
	if k.Public != k2.Public {
		return false
	}
	return k.Private == k2.Private
}

// ToNoiseDHKey re-generate noise.DHKey from DHKey
func (k *DHKey) ToNoiseDHKey() noise.DHKey {
	return noise.DHKey{
		Public:  k.Public.DHKeyBytes[:],
		Private: k.Private.DHKeyBytes[:],
	}
}
