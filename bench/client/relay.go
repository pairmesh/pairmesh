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

package client

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/bench/config"
	"github.com/pairmesh/pairmesh/bench/results"
	"github.com/pairmesh/pairmesh/bench/utils"
	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/pairmesh/pairmesh/security"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// RelayClient is the struct of relay client for benchmark
type RelayClient struct {
	cfg *config.ClientConfig
}

// NewRelayClient returns RelayClient struct with input config
func NewRelayClient(cfg *config.ClientConfig) *RelayClient {
	return &RelayClient{
		cfg: cfg,
	}
}

// Start starts relay test cases from client side
func (c *RelayClient) Start() error {
	// Since in benchmark test we don't have a portal to transmit credentials,
	// we will have to make credentials deterministic so that the relay side and
	// the client side share the same knowledge
	rng := utils.NewDetermRng()
	priv, err := rsa.GenerateKey(rng, 512)
	if err != nil {
		return fmt.Errorf("error generating private key: %s", err.Error())
	}

	serverDHKey, err := noise.DH25519.GenerateKeypair(rng)
	if err != nil {
		return fmt.
			Errorf("error generating server DH key from client side: %s", err.Error())
	}

	clientDHKey, err := noise.DH25519.GenerateKeypair(rng)
	if err != nil {
		return fmt.Errorf("error generating client DH key from client side: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := c.cfg

	relayServer := protocol.RelayServer{
		Host: cfg.Endpoint(),
		Port: int(cfg.Port()),
	}

	var wg sync.WaitGroup
	timer := time.NewTimer(time.Duration(c.cfg.Duration()) * time.Second)
	notify := make(chan struct{})
	res := results.NewResults()

	var i uint16
	for i = 0; uint16(i) < c.cfg.Clients(); i++ {
		wg.Add(1)
		go func(index uint16) {
			defer wg.Done()
			lres := results.NewResults()
			defer func() {
				res.Submit(&lres)
			}()

			// Generate mock credentials
			credentials, err := security.Credential(priv, protocol.UserID(1), protocol.PeerID(index), net.ParseIP("1.2.3.4"), time.Hour)
			if err != nil {
				zap.L().Error(fmt.Sprintf("error generating credentials: %s", err.Error()))
				return
			}
			zap.L().Info(fmt.Sprintf("[worker %d] test job started", index))

			trs := relay.NewClientTransporter(relayServer, credentials, clientDHKey, security.NewDHPublic(serverDHKey.Public))
			client := relay.NewClient(trs)
			go client.Serve(ctx)

			readCh := make(chan bool)
			client.Handler().On(message.PacketType__UnitTestResponse, func(s *relay.Client, typ message.PacketType, msg proto.Message) error {
				readCh <- true
				return nil
			})
			err = client.Connect(ctx)
			if err != nil {
				zap.L().Error(fmt.Sprintf("[worker %d] error connecting to relay server: %s", index, err.Error()))
				return
			}

			payload := utils.GenerateRandPayload(cfg.Payload())

			for {
				select {
				case <-notify:
					zap.L().Info(fmt.Sprintf("[worker %d] test job finished", index))
					return
				default:
					prevTime := time.Now()
					err = client.Send(message.PacketType__UnitTestRequest, &message.P_UnitTestRequest{Field: string(payload)})
					if err != nil {
						zap.L().Error(fmt.Sprintf("[worker %d] error sending packet to relay server: %s", index, err.Error()))
						return
					}
					<-readCh
					postTime := time.Now()
					delta := postTime.Sub(prevTime)
					lres.AddDataPoint(delta)
				}
			}
		}(i)
	}

	<-timer.C
	close(notify)
	wg.Wait()

	res.Report(c.cfg)

	return nil
}
