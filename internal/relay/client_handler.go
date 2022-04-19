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
	"errors"
	"fmt"
	"time"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/internal/codec"
	"github.com/pairmesh/pairmesh/internal/codec/serde"
	"github.com/pairmesh/pairmesh/message"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type (
	clientHandler struct {
		callbacks map[message.PacketType]ClientCallback
	}
)

// NewClientHandler generates, initiates and returns a ClientHandler struct
func NewClientHandler() ClientHandler {
	h := &clientHandler{
		callbacks: map[message.PacketType]ClientCallback{},
	}
	h.On(message.PacketType_HandshakeAck, h.onHandshakeAck)
	h.On(message.PacketType_Heartbeat, h.onHeartbeat)
	return h
}

// On registers a callback function with a certain message packet type
func (h *clientHandler) On(typ message.PacketType, cb ClientCallback) {
	h.callbacks[typ] = cb
}

// Handle implements the SessionHandler interface
func (h *clientHandler) Handle(c *Client, packet codec.RawPacket) error {
	typ := packet.Type
	if c.State() == ClientTransporterStateConnecting && typ != message.PacketType_HandshakeAck {
		return fmt.Errorf("only handshake message valid when connecting status but got: %s", typ)
	}

	cb, ok := h.callbacks[typ]
	if !ok {
		return fmt.Errorf("message %s cannot be handled", typ)
	}

	cipher := c.Cipher()
	if c.State() == ClientTransporterStateConnected && cipher == nil {
		return errors.New("cipher shouldn't be empty when running state")
	}

	payload := packet.Payload
	if cipher != nil {
		decrypted, err := cipher.Decrypt(nil, uint64(packet.Nonce), nil, payload)
		if err != nil {
			return err
		}
		payload = decrypted
	}

	msg, err := serde.Deserialize(typ, payload)
	if err != nil {
		return err
	}

	return cb(c, typ, msg)
}

func (h *clientHandler) onHandshakeAck(c *Client, _ message.PacketType, msg proto.Message) error {
	ack := msg.(*message.PacketHandshakeAck)

	// We ignore the ds returned value to simplify the noise the protocol
	interval, es, _, err := c.HandshakeState().ReadMessage(nil, ack.Message)
	if err != nil {
		zap.L().Error("Read noise handshake state failed", zap.Error(err))
		return err
	}

	parsed, err := time.ParseDuration(string(interval))
	if err != nil {
		zap.L().Error("Parse duration failed", zap.String("duration", string(interval)))
		parsed = constant.HeartbeatInterval
	}

	c.SetCipher(es.Cipher())
	c.SetState(ClientTransporterStateConnected)
	c.SetHeartbeatInterval(parsed)

	zap.L().Info("Relay client noise protocol handshake is finished")
	return nil
}

func (h *clientHandler) onHeartbeat(c *Client, _ message.PacketType, msg proto.Message) error {
	heartbeat := msg.(*message.PacketHeartbeat)
	ts := heartbeat.Timestamp
	t := time.Unix(ts/int64(time.Second), ts%int64(time.Second))
	c.SetLatency(time.Since(t) / 2)
	return nil
}
