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
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/internal/codec"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// SessionTransporter interface, together with sessionTransporterImpl struct,
// is an abstraction of network IO so that the network behaviors could be mocked, and
// therefore the session and relay server could be better tested.
type SessionTransporter interface {
	Cipher() noise.Cipher
	SetCipher(cipher noise.Cipher)
	ReadQueue() <-chan codec.RawPacket
	WriteQueue() chan<- Packet
	Read(ctx context.Context)
	Write(ctx context.Context)
	Close() error
}

type sessionTransporterImpl struct {
	securityTransporter

	wg                *sync.WaitGroup
	chTermination     chan struct{}
	heartbeatInterval time.Duration
	closed            *atomic.Bool
}

func newSessionTransporter(wg *sync.WaitGroup, conn net.Conn, heartbeatInterval time.Duration) *sessionTransporterImpl {
	return &sessionTransporterImpl{
		wg:                  wg,
		securityTransporter: newSecurityTransporter(conn),
		chTermination:       make(chan struct{}, 1),
		heartbeatInterval:   heartbeatInterval,
		closed:              atomic.NewBool(false),
	}
}

// Read implements the SessionTransporter interface.
// We assume that the ctx is same as Write function, so we can ignore it.
func (s *sessionTransporterImpl) Read(_ context.Context) {
	defer func() {
		if e := recover(); e != nil {
			zap.L().Error("Read thread panicked", zap.Reflect("error", e))
		}

		s.wg.Done()
		_ = s.Close()
		close(s.chRead)
	}()

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
			s.chRead <- p
		}
	}
}

func (s *sessionTransporterImpl) Write(ctx context.Context) {
	defer func() {
		if e := recover(); e != nil {
			zap.L().Error("Write thread panicked", zap.Reflect("error", e))
		}

		s.wg.Done()
		_ = s.Close()
		close(s.chWrite)
	}()

	for {
		select {
		case wp := <-s.chWrite:
			err := writePacketHelper(s.conn, wp, s.cipher, s.codec, s.heartbeatInterval)
			if err != nil {
				zap.L().Error("Write message failed", zap.Error(err))
				return
			}

		case <-s.chTermination:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *sessionTransporterImpl) Close() error {
	if s.closed.Swap(true) {
		return errors.New("close a closed session transporter")
	}

	zap.L().Warn("Session connection transporter terminated", zap.Stringer("addr", s.conn.RemoteAddr()))

	close(s.chTermination)
	return s.conn.Close()
}
