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

package driver

import (
	"context"
	"fmt"

	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/node/monitor"
	"go.uber.org/zap"
)

// eventsMonitor updates the local endpoints to the primary relay server periodically.
func (d *deviceDriver) eventsMonitor(ctx context.Context) {
	defer d.wg.Done()

	for {
		select {
		case e := <-d.rm.Events():
			// Only care about the relay server status of connected/closed.
			switch e.Type {
			case relay.EventTypeClientConnected:
				event := e.Data.(relay.EventClientConnected)
				priID := d.rm.PrimaryServerID()
				if priID != event.RelayServer.ID {
					continue
				}
				d.primaryServerConnected = true

			case relay.EventTypeClientClosed:
				event := e.Data.(relay.EventClientClosed)
				priID := d.rm.PrimaryServerID()
				if priID != event.RelayServer.ID {
					continue
				}
				d.primaryServerConnected = false
			}

		case e := <-d.mon.Events():
			switch e.Type {
			case monitor.EventTypeExternalAddressChanged:
				event := e.Data.(monitor.EventExternalAddressChanged)
				zap.L().Info("ExternalAddressChanged", zap.String("address", event.ExternalAddress))
				d.externalAddr.Store(event.ExternalAddress)

				// Latest endpoints.
				endpoints := []string{event.ExternalAddress}
				for _, l := range d.localAddresses() {
					endpoints = append(endpoints, fmt.Sprintf("%s:%d", l, d.config.Port))
				}

				d.mm.SyncEndpoints(endpoints)
			}

		case <-ctx.Done():
			zap.L().Info("Local events monitor stopped")
			return
		}
	}
}
