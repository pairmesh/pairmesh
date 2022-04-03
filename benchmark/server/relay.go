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

package server

import (
	"context"
	"crypto/rsa"
	mrand "math/rand"

	"fmt"
	"time"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/benchmark/config"
	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/message"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type RelayServer struct {
	cfg *config.ServerConfig
}

// NewRelayServer returns RelayServer struct with input config
func NewRelayServer(cfg *config.ServerConfig) *RelayServer {
	return &RelayServer{
		cfg: cfg,
	}
}

// Start starts relay server for benchmarking
func (s *RelayServer) Start() error {
	// Since in benchmark test we don't have a portal to transmit credentials,
	// we will have to make credentials deterministic so that the relay side and
	// the client side share the same knowledge
	mrand.Seed(1)
	source := mrand.NewSource(1)
	rng := mrand.New(source)
	priv, err := rsa.GenerateKey(rng, 512)
	if err != nil {
		return fmt.Errorf("error generating private key: %s", err.Error())
	}

	serverDHKey, err := noise.DH25519.GenerateKeypair(rng)
	if err != nil {
		return fmt.Errorf("error creating server DHKey: %s", err.Error())
	}

	addr := fmt.Sprintf("0.0.0.0:%d", s.cfg.Port())
	server := relay.NewServer(addr, 5*time.Second, serverDHKey, &priv.PublicKey)
	// Register customize callback
	server.Handler().On(message.PacketType__UnitTestRequest, func(s *relay.Session, typ message.PacketType, msg proto.Message) error {
		res := &message.P_UnitTestResponse{Field: msg.(*message.P_UnitTestRequest).Field}
		return s.Send(message.PacketType__UnitTestResponse, res)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zap.L().Info(fmt.Sprintf("Starting pairbench relay server at port %d", s.cfg.Port()))

	err = server.Serve(ctx)
	if err != nil {
		return fmt.Errorf("errors starting relay: %s", err.Error())
	}
	return nil
}
