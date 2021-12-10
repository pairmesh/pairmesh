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

package security_test

import (
	"crypto/rand"
	"crypto/rsa"
	"net"
	"testing"
	"time"

	"github.com/pairmesh/pairmesh/protocol"
	"github.com/pairmesh/pairmesh/security"
	"github.com/stretchr/testify/assert"
)

func TestCredential(t *testing.T) {
	a := assert.New(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 512)
	a.Nil(err)

	var (
		userID = protocol.UserID(88888)
		peerID = protocol.PeerID(12345678)
		ip     = net.IP{0x01, 0x02, 0x03, 0x04}
	)

	credential, err := security.Credential(privateKey, userID, peerID, ip, time.Second*10)
	a.Nil(err)
	a.NotNil(credential)

	userID2, peerID2, ip2, valid := security.VerifyCredential(&privateKey.PublicKey, credential)
	a.True(valid)
	a.Equal(peerID2, peerID)
	a.Equal(userID2, userID)
	a.Equal(ip2, ip)
}

func TestCredentialIPv6(t *testing.T) {
	a := assert.New(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 512)
	a.Nil(err)

	var (
		userID = protocol.UserID(88888)
		peerID = protocol.PeerID(12345678)
		ip     = net.IPv6zero
	)

	credential, err := security.Credential(privateKey, userID, peerID, ip, time.Second*10)
	a.Nil(err)
	a.NotNil(credential)

	userID2, peerID2, ip2, valid := security.VerifyCredential(&privateKey.PublicKey, credential)
	a.True(valid)
	a.Equal(peerID2, peerID)
	a.Equal(userID2, userID)
	a.Equal(ip2, ip)
}

func TestCredential2(t *testing.T) {
	a := assert.New(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 512)
	a.Nil(err)

	var (
		userID = protocol.UserID(88888)
		peerID = protocol.PeerID(12345678)
		ip     = net.IP{0x01, 0x02, 0x03, 0x04}
	)

	credential, err := security.Credential(privateKey, userID, peerID, ip, -time.Second*10)
	a.Nil(err)
	a.NotNil(credential)

	userID2, peerID2, ip2, valid := security.VerifyCredential(&privateKey.PublicKey, credential)
	a.False(valid)
	a.Zero(peerID2)
	a.Zero(userID2)
	a.Nil(ip2)
}
