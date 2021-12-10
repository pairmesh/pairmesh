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

	"github.com/pairmesh/pairmesh/cmd/pairrelay/api"
	"github.com/pairmesh/pairmesh/cmd/pairrelay/config"
	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/internal/relay"
	"go.uber.org/zap"
)

// Serve run the Relay & STUN services
func Serve(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config) {
	zap.L().Info("Relay server is starting up...")

	apiClient := api.NewClient(cfg.Portal.URL, cfg.Portal.Key)

	// Start first keepalive ticker to retrieve the latest information of portal service.
	publicKey, err := retrievePortalKey(apiClient, cfg)
	if err != nil {
		zap.L().Fatal("Connect to portal service failed", zap.Error(err))
	}

	// Preflight the relay server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := relay.NewServer(addr, constant.HeartbeatInterval, cfg.DHKey, publicKey)

	// Register the packet customized callback.
	registerCallback(server)

	// Start the keepalive goroutine to keep alive with the portal service.
	wg.Add(1)
	go keepalive(ctx, wg, server, apiClient, cfg)

	// Start serve STUN service to assist the PairMesh node detect their external address.
	wg.Add(1)
	go serveSTUN(ctx, cfg, wg)

	zap.L().Error("Relay server ready to serve", zap.String("addr", addr))

	if err := server.Serve(ctx); err != nil {
		zap.L().Error("Serve relay server failed", zap.Error(err))
	}

	zap.L().Info("The relay server is shutdown gracefully")
}
