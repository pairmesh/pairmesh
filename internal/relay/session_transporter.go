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
	"github.com/pairmesh/pairmesh/codec"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// SessionTransporter interface, together with sessionTransporterImpl struct,
// is an abstraction of network IO so that the network behaviors could be mocked, and
// therefore the session and relay server could be better tested.
type SessionTransporter interface {
	Cipher() noise.Cipher
	SetCipher(cipher noise.Cipher)
	PublicKey() []byte
	SetPublicKey(pk []byte)
	ReadQueue() <-chan codec.RawPacket
	WriteQueue() chan<- Packet
	TerminationQueue() <-chan struct{}
	Read(ctx context.Context)
	Write(ctx context.Context)
	Close() error
}

type sessionTransporterImpl struct {
	wg                *sync.WaitGroup
	conn              net.Conn
	codec             *codec.RelayCodec
	chRead            chan codec.RawPacket
	chWrite           chan Packet
	chTermination     chan struct{}
	cipher            noise.Cipher
	dhKey             noise.DHKey
	publicKey         []byte // DH public key
	heartbeatInterval time.Duration
	lifetimeHook      SessionLifetimeHook
	closed            *atomic.Bool
}

func newSessionTransporter(wg *sync.WaitGroup, conn net.Conn, heartbeatInterval time.Duration) *sessionTransporterImpl {
	return &sessionTransporterImpl{
		wg:                wg,
		conn:              conn,
		codec:             codec.NewCodec(),
		chRead:            make(chan codec.RawPacket, 128),
		chWrite:           make(chan Packet, 128),
		chTermination:     make(chan struct{}, 1),
		heartbeatInterval: heartbeatInterval,
		closed:            atomic.NewBool(false),
	}
}

// Cipher returns the current session cipher.
func (s *sessionTransporterImpl) Cipher() noise.Cipher {
	return s.cipher
}

// SetCipher sets the session cipher
func (s *sessionTransporterImpl) SetCipher(cipher noise.Cipher) {
	s.cipher = cipher
}

func (s *sessionTransporterImpl) PublicKey() []byte {
	return s.publicKey
}

func (s *sessionTransporterImpl) SetPublicKey(pk []byte) {
	s.publicKey = pk
}

func (s *sessionTransporterImpl) ReadQueue() <-chan codec.RawPacket {
	return s.chRead
}

func (s *sessionTransporterImpl) WriteQueue() chan<- Packet {
	return s.chWrite
}

func (s *sessionTransporterImpl) TerminationQueue() <-chan struct{} {
	return s.chTermination
}

func (s *sessionTransporterImpl) Read(ctx context.Context) {
	defer s.wg.Done()
	defer close(s.chRead)

	go func() {
		select {
		case <-ctx.Done():
			s.Close()
		case <-s.chTermination:
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
			s.chRead <- p
		}
	}
}

func (s *sessionTransporterImpl) Write(ctx context.Context) {
	defer s.wg.Done()
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

func (s *sessionTransporterImpl) Close() error {
	if s.closed.Swap(true) {
		return errors.New("close a closed session transporter")
	}
	close(s.chTermination)
	return s.conn.Close()
}
