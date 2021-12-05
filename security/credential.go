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
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"net"
	"time"

	"github.com/pairmesh/pairmesh/protocol"
)

const (
	credentialValidationHeaderSize = 25
)

// Credential returns a credential to identity the Peer, which contains
// the expiration time and the reclaimed IP address. All requests relevant
// to IP address, the credential is required. We will verify the triple:
// -> (PeerID, IP, Expiration).
// The schema of credential:
// |   PeerID    |  PeerID    |  Expiration  |  IPLen  |    IP     |
// |  8 bytes    | 8 bytes    |    8 bytes   |  1 byte |  Variant  |
// The credential delivered to the client will be encrypted by secret key.
// The secret key is a private key only hold by the gateway.
func Credential(privateKey *rsa.PrivateKey, userID protocol.UserID, peerID protocol.PeerID, ip net.IP, lease time.Duration) ([]byte, error) {
	data := make([]byte, credentialValidationHeaderSize+len(ip))
	binary.BigEndian.PutUint64(data[:8], uint64(userID))
	binary.BigEndian.PutUint64(data[8:16], uint64(peerID))
	expirationAt := time.Now().Add(lease).Unix()
	binary.BigEndian.PutUint64(data[16:24], uint64(expirationAt))
	data[credentialValidationHeaderSize-1] = byte(len(ip))
	copy(data[credentialValidationHeaderSize:credentialValidationHeaderSize+len(ip)], ip)

	digest := sha256.Sum256(data[:])
	signed, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return nil, err
	}
	credential := append(data[:], signed...)
	return credential, nil
}

// VerifyCredential verifies the expiration of the credential, returns the networkID, peerID and IP
func VerifyCredential(publicKey *rsa.PublicKey, credential []byte) (userID protocol.UserID, peerID protocol.PeerID, ip net.IP, valid bool) {
	// Illegal credentials
	if len(credential) <= credentialValidationHeaderSize+net.IPv4len {
		return 0, 0, nil, false
	}

	ipLen := credential[credentialValidationHeaderSize-1]

	digest := sha256.Sum256(credential[:credentialValidationHeaderSize+ipLen])
	err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], credential[credentialValidationHeaderSize+ipLen:])
	if err != nil {
		return 0, 0, nil, false
	}

	// expiration
	if time.Now().Unix() > int64(binary.BigEndian.Uint64(credential[16:credentialValidationHeaderSize-1])) {
		return 0, 0, nil, false
	}
	userID = protocol.UserID(binary.BigEndian.Uint64(credential[:8]))
	peerID = protocol.PeerID(binary.BigEndian.Uint64(credential[8:16]))
	ip = credential[credentialValidationHeaderSize : credentialValidationHeaderSize+ipLen]
	return userID, peerID, ip, true
}
