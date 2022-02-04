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
	"fmt"
	"sync"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/relay/api"
	"github.com/pairmesh/pairmesh/relay/config"
	"go.uber.org/zap"
)

// Serve run the Relay & STUN services
func Serve(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config) error {
	zap.L().Info("Relay server is starting up...")

	apiClient := api.NewClient(cfg.Portal.URL, cfg.Portal.Key)

	// Start first keepalive ticker to retrieve the latest information of portal service.
	publicKey, err := keepaliveWithPortal(apiClient, cfg, nil)
	if err != nil {
		return err
	}

	// Preflight the relay server
	addr := fmt.Sprintf(":%d", cfg.Port)
	server := relay.NewServer(addr, constant.HeartbeatInterval, cfg.DHKey.ToNoiseDHKey(), publicKey)

	// Register the packet customized callback.
	registerCallback(server)

	// Start the keepalive goroutine to keep alive with the portal service.
	wg.Add(1)
	go keepalive(ctx, wg, server, apiClient, cfg)

	// Start serve STUN service to assist the PairMesh node detect their external address.
	wg.Add(1)
	go serveSTUN(ctx, cfg, wg)

	zap.L().Info("Relay server ready to serve", zap.String("addr", addr))

	return server.Serve(ctx)
}
