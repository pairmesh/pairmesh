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
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/flynn/noise"
	gomock "github.com/golang/mock/gomock"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

// Handy util function to create a default server for testing.
func createServer(t *testing.T) *Server {
	port := 10042
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	duration := 5 * time.Second
	serverDHKey, err := noise.DH25519.GenerateKeypair(rand.Reader)
	assert.Nil(t, err)
	priv, err := rsa.GenerateKey(rand.Reader, 512)
	assert.Nil(t, err)
	s := NewServer(addr, duration, serverDHKey, &priv.PublicKey)
	return s
}

func TestGetHeartbeatInterval(t *testing.T) {
	s := createServer(t)

	duration := s.HeartbeatInterval()

	assert.Equal(t, duration, 5*time.Second)
}

func TestGetHandler(t *testing.T) {
	s := createServer(t)

	handler := s.Handler()

	assert.True(t, handler == s.handler)
}

func TestGetDHKey(t *testing.T) {
	s := createServer(t)

	dhKey := s.DHKey()

	assert.Equal(t, dhKey, s.dhKey)
}

func TestGetRSAPublicKey(t *testing.T) {
	s := createServer(t)

	publicKey := s.RSAPublicKey()

	assert.True(t, publicKey == s.publicKey)
}

func TestSetRSAPublicKey(t *testing.T) {
	s := createServer(t)

	oldPublicKey := s.RSAPublicKey()
	newPriv, err := rsa.GenerateKey(rand.Reader, 512)
	assert.Nil(t, err)

	newPublicKey := &newPriv.PublicKey
	s.SetRSAPublicKey(newPublicKey)

	assert.False(t, newPublicKey == oldPublicKey)
	assert.True(t, newPublicKey == s.publicKey)

}

func TestGetSessionFound(t *testing.T) {
	s := createServer(t)

	ses := &Session{}
	peerID := protocol.PeerID(42)
	s.sessions.Store(peerID, ses)

	storedSes := s.Session(peerID)

	assert.True(t, ses == storedSes)
}

func TestGetSessionNotFound(t *testing.T) {
	s := createServer(t)

	peerID := protocol.PeerID(42)

	storedSes := s.Session(peerID)

	assert.Nil(t, storedSes)
}

func TestForeachSession(t *testing.T) {
	s := createServer(t)

	peers := 100

	for id := 0; id < peers; id++ {
		ses := &Session{}
		peerID := protocol.PeerID(id)
		ses.peerID = peerID
		s.sessions.Store(peerID, ses)
	}

	peerIDMap := map[protocol.PeerID]bool{}

	f := func(ses *Session) {
		peerID := ses.PeerID()
		peerIDMap[peerID] = true
	}

	s.ForeachSession(f)

	for id := 0; id < peers; id++ {
		peerID := protocol.PeerID(id)
		assert.True(t, peerIDMap[peerID])
	}
}

// Basic happy case:
// Server is ramped up new, and then handshake with a new session.
func TestOnSessionHandshake(t *testing.T) {
	s := createServer(t)

	ses := &Session{}
	peerID := protocol.PeerID(42)
	ses.peerID = peerID

	s.OnSessionHandshake(ses)

	storedSes, exists := s.sessions.Load(peerID)
	assert.True(t, exists)
	// To assert two pointers equal to each other, the assert.True() should be the method,
	// rather than the assert.Equal().
	// https://github.com/stretchr/testify/issues/597
	assert.True(t, ses == storedSes)
}

// The scenario that the server is closed,
// therefore the session should not be handshaked.
func TestOnSessionHandshakeServerClosed(t *testing.T) {
	s := createServer(t)
	s.closed.Store(true)

	ses := &Session{}

	peerID := protocol.PeerID(42)
	ses.peerID = peerID

	s.OnSessionHandshake(ses)

	_, exists := s.sessions.Load(peerID)
	assert.False(t, exists)

}

// The scenario that the server already has a session stored,
// then trying to handshake with another session with same peerID.
// Test the old session is popped and closed while the new sessoin
// is inserted to the session map.
func TestOnSessionHandshakeSessionExisted(t *testing.T) {
	s := createServer(t)

	ctrl := gomock.NewController(t)
	transporter := NewMockSessionTransporter(ctrl)
	peerID := protocol.PeerID(42)
	oldSes := &Session{
		SessionTransporter: transporter,
		lifetimeHook:       s,
		closed:             atomic.NewBool(false),
		peerID:             peerID,
	}
	transporter.EXPECT().Close().Return(nil).Times(1)
	newSes := &Session{
		peerID: peerID,
	}

	// Store the oldSes into the server s.
	s.sessions.Store(peerID, oldSes)

	s.OnSessionHandshake(newSes)

	storedSes, exists := s.sessions.Load(peerID)
	assert.True(t, exists)
	assert.True(t, storedSes == newSes)
}

// The scenario that the peerID of the session is 0.
// In this case the session should not be handshaked.
func TestOnSessionHandshakeSessionPeerIDZero(t *testing.T) {
	s := createServer(t)

	ses := &Session{}
	peerID := protocol.PeerID(0)
	ses.peerID = peerID

	s.OnSessionHandshake(ses)

	_, exists := s.sessions.Load(peerID)
	assert.False(t, exists)
}

// Basic normal case:
// Session is closed so remove it from s.sessions.
func TestOnSessionClosed(t *testing.T) {
	s := createServer(t)

	storedSes := &Session{}
	deleteSes := &Session{}

	peerID := protocol.PeerID(42)

	s.sessions.Store(peerID, storedSes)

	deleteSes.peerID = peerID

	s.OnSessionClosed(deleteSes)

	_, exist := s.sessions.Load(peerID)
	assert.False(t, exist)
}

// Scenario that the server is already closed.
// In this case the onSessionClosed() function should return fast.
func TestOnSessionClosedServerClosed(t *testing.T) {
	s := createServer(t)
	s.closed.Store(true)

	deleteSes := &Session{}

	peerID := protocol.PeerID(42)

	deleteSes.peerID = peerID

	s.OnSessionClosed(deleteSes)

	_, exist := s.sessions.Load(peerID)
	assert.False(t, exist)
}

// Scenario that the session's peerID is 0.
// In this case the onSessionClosed() function should return fast.
func TestOnSessionClosedSessionPeerIDZero(t *testing.T) {
	s := createServer(t)

	deleteSes := &Session{}

	peerID := protocol.PeerID(0)

	deleteSes.peerID = peerID

	s.OnSessionClosed(deleteSes)

	_, exist := s.sessions.Load(peerID)
	assert.False(t, exist)
}

func TestClose(t *testing.T) {
	s := createServer(t)

	ctrl := gomock.NewController(t)

	peers := 100
	for id := 0; id < peers; id++ {
		peerID := protocol.PeerID(id)
		transporter := NewMockSessionTransporter(ctrl)
		ses := &Session{
			SessionTransporter: transporter,
			lifetimeHook:       s,
			closed:             atomic.NewBool(false),
			peerID:             peerID,
		}
		transporter.EXPECT().Close().Return(nil).Times(1)
		s.sessions.Store(peerID, ses)
	}

	err := s.Close()

	assert.Nil(t, err)
}
