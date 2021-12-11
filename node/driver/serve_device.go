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
	"net"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/internal/logutil"
	"go.uber.org/zap"
)

func parseDst(b []byte) net.IP {
	if len(b) < 20 {
		return nil
	}
	return net.IPv4(b[16], b[17], b[18], b[19])
}

func (d *nodeDriver) serveDevRead(ctx context.Context) {
	defer d.wg.Done()

	npnp := net.IPv4(239, 255, 255, 250)

	buffer := make([]byte, constant.MaxBufferSize)
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Serve virtual device read goroutine stopped")
			return

		default:
			c, err := d.device.Read(buffer)
			if err != nil {
				continue
			}

			if !d.enable.Load() {
				continue
			}

			dst := parseDst(buffer[:c])
			if dst == nil {
				zap.L().Warn("Parse IPv4 header failed", zap.ByteString("data", buffer[:c]))
				continue
			}

			if dst.Equal(npnp) {
				continue
			}

			if logutil.IsEnableDevice() {
				zap.L().Debug("Read packet from device", zap.Stringer("to", dst))
			}

			// TODO: support broadcast
			destination := dst.String()
			t := d.mm.Tunnel(destination)
			if t == nil {
				continue
			}

			dataCopy := make([]byte, c)
			copy(dataCopy, buffer[:c])

			// Write pipeline back if the destination is the current virtual address (loopback)
			if destination == d.credential.address {
				d.chDevWrite <- dataCopy
				continue
			}

			t.Write(dataCopy)
		}
	}
}

func (d *nodeDriver) serveDevWrite(ctx context.Context) {
	defer d.wg.Done()

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Serve virtual device write goroutine stopped")
			return

		case data := <-d.chDevWrite:
			if !d.enable.Load() {
				continue
			}

			if logutil.IsEnableDevice() {
				dst := parseDst(data)
				if dst == nil {
					zap.L().Warn("Parse IPv4 header failed", zap.ByteString("data", data))
					continue
				}
				zap.L().Debug("Write packet into device", zap.Stringer("to", dst))
			}

			_, err := d.device.Write(data)
			if err != nil {
				zap.L().Error("Write data into virtual device failed", zap.Error(err))
			}
		}
	}
}
