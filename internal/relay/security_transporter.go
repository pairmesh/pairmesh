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
	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/internal/codec"

	"net"
)

type securityTransporter struct {
	conn    net.Conn
	codec   *codec.RelayCodec
	chRead  chan codec.RawPacket
	chWrite chan Packet
	cipher  noise.Cipher
}

func newSecurityTransporter(conn net.Conn) securityTransporter {
	return securityTransporter{
		conn:    conn,
		codec:   codec.NewCodec(),
		chRead:  make(chan codec.RawPacket, 64),
		chWrite: make(chan Packet, 64),
	}
}

// Cipher returns the current session cipher.
func (s *securityTransporter) Cipher() noise.Cipher {
	return s.cipher
}

// SetCipher sets the session cipher
func (s *securityTransporter) SetCipher(cipher noise.Cipher) {
	s.cipher = cipher
}

func (s *securityTransporter) ReadQueue() <-chan codec.RawPacket {
	return s.chRead
}

func (s *securityTransporter) WriteQueue() chan<- Packet {
	return s.chWrite
}
