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

package server

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/pairmesh/pairmesh/relay/api"
	"github.com/pairmesh/pairmesh/relay/config"
	"go.uber.org/zap"
)

var startedAt = time.Now()

func keepaliveWithPortal(apiClient *api.Client, cfg *config.Config, peers []protocol.PeerID) (*rsa.PublicKey, error) {
	resp, err := apiClient.Keepalive(cfg, peers, startedAt)
	if err != nil {
		return nil, err
	}
	rawbytes, err := base64.RawStdEncoding.DecodeString(resp.PublicKey)
	if err != nil {
		return nil, err
	}
	return x509.ParsePKCS1PublicKey(rawbytes)
}

func keepalive(ctx context.Context, wg *sync.WaitGroup, server *relay.Server, apiClient *api.Client, cfg *config.Config) {
	defer wg.Done()
	const (
		tickerInterval = 5 * time.Minute
		syncInterval   = 10 * time.Minute
	)
	ticker := time.NewTicker(tickerInterval)
	events := server.Events()
	var peers []protocol.PeerID
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("The keepalive with portal is over")
			ticker.Stop()
			return

		case e := <-events:
			// Notify portal service if client closed.
			var peers []protocol.PeerID
			if e.Type == relay.EventTypeSessionClosed {
				peers = append(peers, e.Data.(*relay.EventSessionClosed).Session.PeerID())
			}
			// Batch all session closed events
			if size := len(events); size > 0 {
				for i := 0; i < size; i++ {
					e := <-events
					if e.Type == relay.EventTypeSessionClosed {
						peers = append(peers, e.Data.(*relay.EventSessionClosed).Session.PeerID())
					}
				}
			}
			err := apiClient.PeersOffline(peers)
			if err != nil {
				zap.L().Error("Notify portal service peers offline failed", zap.Error(err))
				continue
			}

		case <-ticker.C:
			peers = peers[:0]
			server.ForeachSession(func(s *relay.Session) {
				if s.IsPrimary() && time.Since(s.SyncAt()) > syncInterval {
					peers = append(peers, s.PeerID())
				}
			})
			publicKey, err := keepaliveWithPortal(apiClient, cfg, peers)
			if err != nil {
				zap.L().Error("Retrieve the latest portal server information failed", zap.Error(err))
				continue
			}
			now := time.Now()
			for _, peerId := range peers {
				s := server.Session(peerId)
				if s == nil {
					continue
				}
				s.SetSyncAt(now)
			}
			server.SetRSAPublicKey(publicKey)
		}
	}
}
