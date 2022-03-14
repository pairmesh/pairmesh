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

	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"go.uber.org/atomic"
)

const bufferSize = 512

type SessionState byte

const (
	SessionStateInit    SessionState = 0
	SessionStateRunning SessionState = 1
)

// Session maintains the connection Session between relay server/client.
type Session struct {
	SessionTransporter

	// Read-only fields for concurrent safe.
	userID          protocol.UserID
	peerID          protocol.PeerID
	vaddress        net.IP // Virtual address allocated by Peerly
	isPrimary       bool
	state           SessionState
	closed          *atomic.Bool
	lifetimeHook    SessionLifetimeHook
	handler         SessionHandler
	lastHeartbeatAt time.Time // Update to the latest heartbeat time periodically.
	lastSyncAt      time.Time // Update the latest sync time while keepalive with portal service successfully.
}

// newSession returns a Session.
func newSession(transporter SessionTransporter, lifetimeHook SessionLifetimeHook, handler SessionHandler) *Session {
	return &Session{
		SessionTransporter: transporter,
		state:              SessionStateInit,
		lifetimeHook:       lifetimeHook,
		handler:            handler,
		closed:             atomic.NewBool(false),
	}
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

func (s *Session) IsPrimary() bool {
	return s.isPrimary
}

func (s *Session) SetIsPrimary(is bool) {
	s.isPrimary = is
}

func (s *Session) SyncAt() time.Time {
	return s.lastSyncAt
}

func (s *Session) SetSyncAt(t time.Time) {
	s.lastSyncAt = t
}

func (s *Session) LifetimeHook() SessionLifetimeHook {
	return s.lifetimeHook
}

// Send sends message to the pairmesh client-side via the client session.
func (s *Session) Send(typ message.PacketType, msg proto.Message) error {
	if s.closed.Load() {
		return errors.New("cannot send message to closed session")
	}

	select {
	case s.WriteQueue() <- Packet{Type: typ, Message: msg}:
		return nil
	default:
		return fmt.Errorf("write buffer exceed: %d", s.peerID)
	}
}

func (s *Session) Serve(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case p, ok := <-s.ReadQueue():
			if !ok {
				return
			}
			err := s.handler.Handle(s, p)
			if err != nil {
				zap.L().Error("Handle message failed", zap.Stringer("type", p.Type), zap.Error(err))
				continue
			}

		}
	}
}

// Close closes the current session.
func (s *Session) Close() error {
	if s.closed.Swap(true) {
		return errors.New("close a closed Session")
	}
	if err := s.SessionTransporter.Close(); err != nil {
		return err
	}
	s.lifetimeHook.OnSessionClosed(s)
	return nil
}

// String implements the fmt.Stringer interface.
func (s *Session) String() string {
	return fmt.Sprintf("State=%v", s.state)
}
