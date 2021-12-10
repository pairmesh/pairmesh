package security

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/flynn/noise"
)

const keySize = 32

// DHPrivate represents a curve25519 private key.
type DHPrivate [keySize]byte

// DHPublic represents a curve25519 public key.
type DHPublic [keySize]byte

// IsZero reports whether the DHPrivate p is the zero value.
func (k DHPrivate) IsZero() bool { return k == DHPrivate{} }
func (k DHPrivate) String() string {
	return base64.StdEncoding.EncodeToString(k[:])
}
func (k DHPrivate) ShortString() string {
	return "[" + base64.StdEncoding.EncodeToString(k[:])[:5] + "]"
}

// NewDHPrivate returns a new private key.
func NewDHPrivate(s []byte) DHPrivate {
	if len(s) > keySize {
		return DHPrivate{}
	}
	var x DHPrivate
	copy(x[:], s)
	return x
}

// B32 returns k as the *[32]byte type that's used by the
// golang.org/x/crypto packages. This allocates; it might
// not be appropriate for performance-sensitive paths.
func (k DHPrivate) B32() *[keySize]byte { return (*[keySize]byte)(&k) }

// IsZero reports whether DHPublic p is the zero value.
func (k DHPublic) IsZero() bool { return k == DHPublic{} }

// ShortString returns the Meshstep conventional debug representation
// of a public key: the first five base64 digits of the key, in square
// brackets.
func (k DHPublic) ShortString() string {
	return "[" + base64.StdEncoding.EncodeToString(k[:])[:5] + "]"
}
func (k DHPublic) String() string {
	return base64.StdEncoding.EncodeToString(k[:])
}

func (k DHPublic) MarshalText() ([]byte, error) {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(k)))
	base64.StdEncoding.Encode(buf, k[:])
	return buf, nil
}

func (k DHPublic) Bytes() []byte {
	b := make([]byte, keySize)
	copy(b, k[:])
	return b
}

func (k *DHPublic) UnmarshalText(txt []byte) error {
	if *k != (DHPublic{}) {
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

// B32 returns k as the *[32]byte type that's used by the
// golang.org/x/crypto packages. This allocates; it might
// not be appropriate for performance-sensitive paths.
func (k DHPublic) B32() *[keySize]byte { return (*[keySize]byte)(&k) }

// NewDHPublic returns a new public key.
func NewDHPublic(s []byte) DHPublic {
	if len(s) > keySize {
		return DHPublic{}
	}
	var x DHPublic
	copy(x[:], s)
	return x
}

// DHKey a noise.DHkey wrapper for meshstep
type DHKey struct {
	Public  DHPublic
	Private DHPrivate
}

// FromNoiseDHKey generate a DHKey from noise.DHKey
func FromNoiseDHKey(nKey noise.DHKey) *DHKey {
	return &DHKey{
		Public:  NewDHPublic(nKey.Public),
		Private: NewDHPrivate(nKey.Private),
	}
}
func (k *DHKey) IsZero() bool {
	return k.Public.IsZero()
}

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
		Public:  k.Public[:],
		Private: k.Private[:],
	}
}
