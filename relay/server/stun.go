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
	"errors"
	"net"
	"os"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/internal/stun"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/relay/config"
	"go.uber.org/zap"
)

func serveSTUN(ctx context.Context, cfg *config.Config, wg *sync.WaitGroup) {
	defer wg.Done()

	addr := &net.UDPAddr{
		Port: cfg.STUNPort,
	}
	udpConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		zap.L().Fatal("Open STUN listener is failed", zap.Error(err))
	}
	zap.L().Info("The STUN server is running", zap.Any("addr", udpConn.LocalAddr()))

	var buf [64 << 10]byte

	serveProtocol := func() error {
		_ = udpConn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, remote, err := udpConn.ReadFromUDP(buf[:])
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return nil
			}
			return err
		}

		if logutil.IsEnableRelay() {
			zap.L().Debug("Read UDP packet", zap.Stringer("remote", remote))
		}

		data := buf[:n]
		if !stun.Is(data) {
			return errors.New("not stun packet")
		}

		txid, err := stun.ParseBindingRequest(data)
		if err != nil {
			return err
		}

		res := stun.Response(txid, remote.IP, uint16(remote.Port))
		_, err = udpConn.WriteToUDP(res, remote)

		return err
	}

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("The STUN server is over")
			return

		default:
			err := serveProtocol()
			if err != nil {
				zap.L().Error("RelayServer stun failed", zap.Error(err))
			}
		}
	}
}
