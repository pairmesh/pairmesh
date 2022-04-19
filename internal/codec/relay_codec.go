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

package codec

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/message"
)

type (
	// RelayCodec is the struct for message codec
	RelayCodec struct {
		buf   *bytes.Buffer
		size  int
		typ   message.PacketType
		nonce uint32
	}

	// RawPacket is the struct for a row network packet
	RawPacket struct {
		Type    message.PacketType
		Nonce   uint32
		Payload []byte
	}
)

// NewCodec returns a new RelayCodec instance
func NewCodec() *RelayCodec {
	return &RelayCodec{
		buf:  bytes.NewBuffer(nil),
		size: -1,
	}
}

// Encode encodes the protobuf message in relay command form into bytes.
func (c *RelayCodec) Encode(nonce uint32, typ message.PacketType, data []byte) ([]byte, error) {
	payloadSize := uint32(len(data))

	// Command packet format:
	// | nonce(4bytes) | type(2bytes) | payload size (4bytes) | payload |
	buffer := make([]byte, constant.PacketHeaderSize+payloadSize)

	// nonce
	from := 0
	to := constant.HeaderNonceSize
	binary.BigEndian.PutUint32(buffer[:to], nonce)

	// type
	from = constant.HeaderNonceSize
	to = from + constant.HeaderPacketTypeSize
	typeNumber := uint16(typ) ^ uint16(nonce&(1<<16-1))
	binary.BigEndian.PutUint16(buffer[from:to], typeNumber)

	// payload size
	from = constant.HeaderNonceSize + constant.HeaderPacketTypeSize
	to = constant.PacketHeaderSize
	binary.BigEndian.PutUint32(buffer[from:to], payloadSize)

	// payload
	from = constant.PacketHeaderSize
	copy(buffer[from:], data)

	return buffer, nil
}

// Decode decodes the protobuf message into MERP command.
func (c *RelayCodec) Decode(input []byte) ([]RawPacket, error) {
	c.buf.Write(input)

	// Check the buffer size to ensure at-least have one message.
	if c.buf.Len() < constant.PacketHeaderSize {
		return nil, nil
	}

	readHeader := func() {
		header := c.buf.Next(constant.PacketHeaderSize)

		// nonce
		from := 0
		to := constant.HeaderNonceSize
		nonce := binary.BigEndian.Uint32(header[:to])

		// type
		from = constant.HeaderNonceSize
		to = from + constant.HeaderPacketTypeSize
		typeNumber := binary.BigEndian.Uint16(header[from:to])
		typ := message.PacketType(typeNumber ^ uint16(nonce&(1<<16-1)))

		// frame size
		from = constant.HeaderNonceSize + constant.HeaderPacketTypeSize
		to = constant.PacketHeaderSize

		c.size = int(binary.BigEndian.Uint32(header[from:to]))
		c.typ = typ
		c.nonce = nonce
	}

	// Negative size means there is no reading message.
	if c.size < 0 {
		readHeader()
	}

	var output []RawPacket

	// Read all messages.
	for c.size > 0 && c.size <= c.buf.Len() {
		if c.size > constant.MaxMessageSize {
			return nil, errors.New("message size exceed")
		}

		// RawPacket content
		buffer := c.buf.Next(c.size)
		payload := make([]byte, len(buffer))
		copy(payload, buffer)

		output = append(output, RawPacket{
			Type:    c.typ,
			Nonce:   c.nonce,
			Payload: payload,
		})

		// Read the next command header.
		if c.buf.Len() >= constant.PacketHeaderSize {
			readHeader()
		} else {
			c.size = -1
		}
	}

	return output, nil
}
