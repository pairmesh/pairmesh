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
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/protocol"

	"github.com/pairmesh/pairmesh/codec"
	"github.com/pairmesh/pairmesh/message"

	"github.com/flynn/noise"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const bufferSize = 512

type SessionState byte

const (
	SessionStateInit    SessionState = 0
	SessionStateRunning SessionState = 1
)

// Session maintains the connection Session between relay server/client.
type Session struct {
	userID            protocol.UserID
	peerID            protocol.PeerID
	dhKey             noise.DHKey
	cipher            noise.Cipher
	publicKey         []byte // DH public key
	vaddress          net.IP // Virtual address allocated by Peerly
	conn              net.Conn
	state             SessionState
	codec             *codec.RelayCodec
	closed            *atomic.Bool
	die               chan struct{}
	handler           SessionHandler
	chWrite           chan Packet
	heartbeatInterval time.Duration
	lastHeartbeatAt   time.Time
	callback          struct {
		onHandshake func(ses *Session)
		onClosed    func(ses *Session)
	}

	userData sync.Map
}

// newSession returns a Session.
func newSession(conn net.Conn, heartbeatInterval time.Duration, handler SessionHandler) *Session {
	return &Session{
		conn:              conn,
		state:             SessionStateInit,
		codec:             codec.NewCodec(),
		closed:            atomic.NewBool(false),
		die:               make(chan struct{}, 1),
		chWrite:           make(chan Packet, 64),
		heartbeatInterval: heartbeatInterval,
		handler:           handler,
	}
}

// Set sets the user data associated with the current session.
func (s *Session) Set(key, val interface{}) {
	s.userData.Store(key, val)
}

// Get gets the user data associated with the current session.
func (s *Session) Get(key interface{}) (interface{}, bool) {
	return s.userData.Load(key)
}

// State returns the current session state.
func (s *Session) State() SessionState {
	return s.state
}

// SetState sets the session state.
func (s *Session) SetState(state SessionState) {
	s.state = state
}

// UserID returns the user id of the current session.
func (s *Session) UserID() protocol.UserID {
	return s.userID
}

// SetUserID sets the session userID
func (s *Session) SetUserID(userID protocol.UserID) {
	s.userID = userID
}

// PeerID returns the peer id of the current session.
func (s *Session) PeerID() protocol.PeerID {
	return s.peerID
}

// SetPeerID sets the session peerID
func (s *Session) SetPeerID(peerID protocol.PeerID) {
	s.peerID = peerID
}

// Cipher returns the current session cipher.
func (s *Session) Cipher() noise.Cipher {
	return s.cipher
}

// SetCipher sets the session cipher
func (s *Session) SetCipher(cipher noise.Cipher) {
	s.cipher = cipher
}

func (s *Session) PublicKey() []byte {
	return s.publicKey
}

func (s *Session) SetPublicKey(pk []byte) {
	s.publicKey = pk
}

// VAddress returns the virtual address allocated by portal service.
func (s *Session) VAddress() net.IP {
	return s.vaddress
}

// SetVAddress sets the virtual address allocated by portal service.
// The virtual address is coming from the credential which is encoded in handshake
// message and signed by portal service. So it cannot be counterfeit.
func (s *Session) SetVAddress(addr net.IP) {
	s.vaddress = addr
}

func (s *Session) SetHeartbeatAt(t time.Time) {
	s.lastHeartbeatAt = t
}

// Send sends message to the pairmesh client-side via the client session.
func (s *Session) Send(typ message.PacketType, msg proto.Message) error {
	if s.closed.Load() {
		return errors.New("cannot send message to closed session")
	}

	select {
	case s.chWrite <- Packet{Type: typ, Message: msg}:
		return nil
	default:
		return fmt.Errorf("write buffer exceed: %s", s.conn.RemoteAddr())
	}
}

// Close closes the current session.
func (s *Session) Close() error {
	if s.closed.Swap(true) {
		return errors.New("close a closed Session")
	}

	close(s.die)
	s.callback.onClosed(s)

	return s.conn.Close()
}

// String implements the fmt.Stringer interface.
func (s *Session) String() string {
	return fmt.Sprintf("State=%v", s.state)
}

func (s *Session) read(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	go func() {
		select {
		case <-ctx.Done():
			s.Close()
		case <-s.die:
		}
	}()

	defer s.Close()

	buffer := make([]byte, bufferSize)
	for {
		_ = s.conn.SetReadDeadline(time.Now().Add(2 * s.heartbeatInterval))
		n, err := s.conn.Read(buffer)
		if err != nil {
			zap.L().Error("Read connection failed", zap.Error(err))
			return
		}

		output, err := s.codec.Decode(buffer[:n])
		if err != nil {
			zap.L().Error("Decode packet failed", zap.Error(err))
			return
		}

		for _, p := range output {
			err := s.handler.Handle(s, p)
			if err != nil {
				zap.L().Error("Handle message failed", zap.Stringer("type", p.Type), zap.Error(err))
				continue
			}
		}
	}
}

func (s *Session) write(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(s.chWrite)

	for {
		select {
		case wp := <-s.chWrite:
			err := writePacketHelper(s.conn, wp, s.cipher, s.codec, s.heartbeatInterval)
			if err != nil {
				zap.L().Error("Write message failed", zap.Error(err))
				_ = s.Close()
				return
			}

		case <-ctx.Done():
			return
		}
	}
}
