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
	"github.com/pairmesh/pairmesh/relay/api"
	"github.com/pairmesh/pairmesh/relay/config"
	"github.com/pairmesh/pairmesh/security"
	"go.uber.org/zap"
)

var startedAt = time.Now()

func retrievePortalKey(apiClient *api.Client, cfg *config.Config) (*rsa.PublicKey, error) {
	resp, err := apiClient.Keepalive(cfg, security.NewDHPublic(cfg.DHKey.ToNoiseDHKey().Public).String(), startedAt)
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

	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("The keepalive with portal is over")
			ticker.Stop()
			return

		case <-ticker.C:
			publicKey, err := retrievePortalKey(apiClient, cfg)
			if err != nil {
				zap.L().Error("Retrieve the latest portal server information failed", zap.Error(err))
				continue
			}
			server.SetRSAPublicKey(publicKey)
		}
	}
}
