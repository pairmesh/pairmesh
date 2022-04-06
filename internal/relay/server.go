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
	"crypto/rsa"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/protocol"

	"github.com/flynn/noise"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// Server represents the relay server, which is used to accept connection incoming
// from pairmesh relay clients.
// The server only play the Session manager role: create/remove Session.
type Server struct {
	addr string

	running           *atomic.Bool
	closed            *atomic.Bool
	dhKey             noise.DHKey
	publicKey         *rsa.PublicKey
	wg                *sync.WaitGroup
	heartbeatInterval time.Duration
	handler           SessionHandler
	// sessions only contain Session which completed handshake.
	// mscfg.PeerID -> *Session
	sessions sync.Map
}

// NewServer returns a new Server instance according to the serve vaddress and heartbeat
// interval.
func NewServer(addr string, heartbeatInterval time.Duration, dhKey noise.DHKey, publicKey *rsa.PublicKey) *Server {
	s := &Server{
		addr:              addr,
		running:           atomic.NewBool(false),
		closed:            atomic.NewBool(false),
		dhKey:             dhKey,
		publicKey:         publicKey,
		wg:                &sync.WaitGroup{},
		heartbeatInterval: heartbeatInterval,
		sessions:          sync.Map{},
	}
	s.handler = NewSessionHandler(s)
	return s
}

// HeartbeatInterval implements the SessionManager interface.
func (s *Server) HeartbeatInterval() time.Duration {
	return s.heartbeatInterval
}

func (s *Server) Handler() SessionHandler {
	return s.handler
}

// DHKey implements the handler.SessionManager interface
func (s *Server) DHKey() noise.DHKey {
	return s.dhKey
}

// RSAPublicKey implements the handler.SessionManager interface
func (s *Server) RSAPublicKey() *rsa.PublicKey {
	return s.publicKey
}

func (s *Server) SetRSAPublicKey(key *rsa.PublicKey) {
	s.publicKey = key
}

// Session implements the handler.SessionManager interface
func (s *Server) Session(peerID protocol.PeerID) *Session {
	v, found := s.sessions.Load(peerID)
	if !found {
		return nil
	}
	return v.(*Session)
}

func (s *Server) ForeachSession(fn func(*Session)) {
	s.sessions.Range(func(_, value interface{}) bool {
		fn(value.(*Session))
		return true
	})
}

// Serve starts to serve the server process.
func (s *Server) Serve(ctx context.Context) error {
	if s.running.Swap(true) {
		return errors.New("serve a running server")
	}

	cfg := net.ListenConfig{}
	listener, err := cfg.Listen(ctx, "tcp", s.addr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		zap.L().Info("Listener ready to close")
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			zap.L().Error("Accept incoming connection failed", zap.Error(err))
			return err
		}

		// Create a Session to maintain the Session state.
		trs := newSessionTransporter(s.wg, conn, s.heartbeatInterval)
		ses := newSession(trs, s, s.handler)

		s.wg.Add(3)
		go trs.Read(ctx)
		go trs.Write(ctx)
		go ses.Serve(ctx, s.wg)
	}
}

func (s *Server) Close() error {
	if s.closed.Swap(true) {
		return errors.New("close a closed server")
	}

	s.sessions.Range(func(key, value interface{}) bool {
		ses := value.(*Session)
		if err := ses.Close(); err != nil {
			zap.L().Error("Close Session failed", zap.Error(err), zap.Stringer("session", ses))
		}
		return true
	})

	s.wg.Wait()

	return nil
}

// OnSessionHandshake implements the SessionLifetimeHook interface
func (s *Server) OnSessionHandshake(ses *Session) {
	if s.closed.Load() {
		return
	}

	// Handshake session always has a non-zero peerID.
	if ses.peerID == 0 {
		return
	}

	if logutil.IsEnablePeer() {
		zap.L().Debug("New session handshake successfully", zap.Reflect("peerId", ses.PeerID()), zap.Bool("isPrimary", ses.IsPrimary()))
	}

	// Close the old session if new connection established.
	old, exists := s.sessions.Load(ses.peerID)
	if exists {
		oldSes := old.(*Session)
		zap.L().Warn("Close old session", zap.Reflect("peerID", oldSes.peerID))
		oldSes.Close()
	}

	// Add or update with the new session.
	s.sessions.Store(ses.peerID, ses)
}

// OnSessionClosed implements the SessionLifetimeHook interface
func (s *Server) OnSessionClosed(ses *Session) {
	if s.closed.Load() {
		return
	}

	// Non-zero peerID session should not be appearance in sessions.
	if ses.peerID == 0 {
		return
	}
	s.sessions.Delete(ses.peerID)
}
