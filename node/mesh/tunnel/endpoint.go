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

package tunnel

import (
	"context"
	"net"
	"time"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/internal/logutil"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// Endpoint represents the record of discovering peerinfo via message.PacketType_Discovery.
type Endpoint struct {
	served atomic.Bool

	latency  time.Duration
	lastSeen time.Time
	address  string
	udpConn  *net.UDPConn
	callback UDPPacketCallback
	cancelFn context.CancelFunc
	chWrite  chan []byte
}

func newEndpoint(udpConn *net.UDPConn, latency time.Duration, lastSeen time.Time, callback UDPPacketCallback) *Endpoint {
	return &Endpoint{
		latency:  latency,
		lastSeen: lastSeen,
		address:  udpConn.RemoteAddr().String(),
		udpConn:  udpConn,
		callback: callback,
		chWrite:  make(chan []byte, 128),
	}
}

func (c *Endpoint) Write(data []byte) {
	c.chWrite <- data
}

func (c *Endpoint) serve() {
	if c.served.Swap(true) {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFn = cancel

	go c.serveRead(ctx)
	go c.serveWrite(ctx)
}

func (c *Endpoint) serveRead(ctx context.Context) {
	defer c.udpConn.Close()

	buffer := make([]byte, constant.MaxBufferSize)
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Serve endpoint connection read goroutine stopped", zap.String("address", c.address))
			return

		default:
			n, err := c.udpConn.Read(buffer)
			if err != nil {
				continue
			}

			if logutil.IsEnablePeer() {
				zap.L().Debug("Read packet from peer", zap.String("from", c.address))
			}

			c.callback.OnUDPPacket(c.udpConn, buffer[:n])
		}
	}
}

func (c *Endpoint) serveWrite(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Serve endpoint connection write goroutine stopped", zap.String("address", c.address))
			return

		case data := <-c.chWrite:
			if logutil.IsEnablePeer() {
				zap.L().Debug("Write packet in peer", zap.String("to", c.address))
			}

			_, err := c.udpConn.Write(data)
			if err != nil {
				zap.L().Error("Write data to peer failed", zap.Error(err))
			}
		}
	}
}
