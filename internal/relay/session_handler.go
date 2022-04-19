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

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/internal/codec"
	"github.com/pairmesh/pairmesh/internal/codec/serde"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/security"
	"google.golang.org/protobuf/proto"
)

type (
	sessionHandler struct {
		sm        SessionManager
		callbacks map[message.PacketType]SessionCallback
	}
)

// NewSessionHandler generates and returns a SessionHandler struct
func NewSessionHandler(sm SessionManager) SessionHandler {
	h := &sessionHandler{
		sm:        sm,
		callbacks: map[message.PacketType]SessionCallback{},
	}
	h.On(message.PacketType_Handshake, h.onHandshake)
	h.On(message.PacketType_Heartbeat, h.onHeartbeat)
	return h
}

// On registers a callback function to sessionHandler with certain message packet type
func (h *sessionHandler) On(typ message.PacketType, cb SessionCallback) {
	h.callbacks[typ] = cb
}

// Handle implements the SessionHandler interface
func (h *sessionHandler) Handle(s *Session, packet codec.RawPacket) error {
	typ := packet.Type
	if s.State() == SessionStateInit && typ != message.PacketType_Handshake {
		return fmt.Errorf("only handshake message valid, but receive %s", typ)
	}

	cb, ok := h.callbacks[typ]
	if !ok {
		return fmt.Errorf("message %s cannot be handled", typ)
	}

	cipher := s.Cipher()
	if s.State() == SessionStateRunning && cipher == nil {
		return errors.New("cipher shouldn't be empty when running state")
	}

	payload := packet.Payload
	if cipher != nil {
		decrypted, err := cipher.Decrypt(nil, uint64(packet.Nonce), nil, payload)
		if err != nil {
			return fmt.Errorf("decrypt the payload is failed: %w", err)
		}
		payload = decrypted
	}

	msg, err := serde.Deserialize(typ, payload)
	if err != nil {
		return err
	}

	return cb(s, typ, msg)
}

// onHandshake handles the message.PacketHandshake and assign some values for session instance.
func (h *sessionHandler) onHandshake(s *Session, _ message.PacketType, msg proto.Message) error {
	hs := msg.(*message.PacketHandshake)

	config := noise.Config{
		CipherSuite:   security.CipherSuite,
		Pattern:       security.HandshakePatternNN,
		StaticKeypair: h.sm.DHKey(),
		Initiator:     false,
	}

	state, err := noise.NewHandshakeState(config)
	if err != nil {
		return err
	}

	// Because we will not append the payload into the handshake message.
	// So we can make the `out` parameter to be nil and ignore the result.
	credentials, _, _, err := state.ReadMessage(nil, hs.Message)
	if err != nil {
		return err
	}

	// Validate the credential.
	userID, peerID, ip, valid := security.VerifyCredential(h.sm.RSAPublicKey(), credentials)
	if !valid {
		return errors.New("invalid credentials")
	}

	// Because we use the handshake NN pattern (see: https://noiseprotocol.org/noise.html)
	// So there must be completed in the first RTT. The two cipher should be non-nil.
	// To simplify the implementation, we ignore the ds returned value
	interval := h.sm.HeartbeatInterval()
	out, es, _, err := state.WriteMessage(make([]byte, 0, 128), []byte(interval.String()))
	if err != nil {
		return err
	}

	s.SetCipher(es.Cipher())
	s.SetState(SessionStateRunning)
	s.SetUserID(userID)
	s.SetPeerID(peerID)
	s.SetVAddress(ip)
	s.SetIsPrimary(hs.IsPrimary)
	s.LifetimeHook().OnSessionHandshake(s)

	// Construct the response message which is used to acknowledge handshake.
	res := &message.PacketHandshakeAck{Message: out}

	return s.Send(message.PacketType_HandshakeAck, res)
}

// onHeartbeat echo the heartbeat message.
func (h *sessionHandler) onHeartbeat(s *Session, typ message.PacketType, msg proto.Message) error {
	ts := msg.(*message.PacketHeartbeat).Timestamp
	s.SetHeartbeatAt(time.Unix(ts/int64(time.Second), ts%int64(time.Second)))
	return s.Send(typ, msg)
}
