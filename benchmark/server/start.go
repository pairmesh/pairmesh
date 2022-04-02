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
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"

	"go.uber.org/zap"
)

const bufferSize = 512

// Start function starts a simple echo server as server side of pairbenchmark
// If cfg.isBounce is true, it echoes whatever it hears from incoming connection
// Otherwise, it always echoes "OK" as minimal backward payload
func Start(cfg *Config) error {
	zap.L().Info("Starting pairbenchmark server")
	addr := fmt.Sprintf("%s:%d", "0.0.0.0", cfg.Port())
	s, err := net.Listen("tcp", addr)
	if err != nil {
		zap.L().Error("Error setting up server")
		return err
	}

	zap.L().Info(fmt.Sprintf("Started pairbenchmark server on port %d", cfg.Port()))

	defer s.Close()

	for {
		conn, err := s.Accept()
		if err != nil {
			zap.L().Error("Error accepting incoming connection")
			return err
		}

		go func(conn net.Conn, bounce bool) {
			defer conn.Close()
			for {
				buffer := make([]byte, bufferSize)
				if _, err = conn.Read(buffer); err != nil {
					if err != io.EOF {
						zap.L().Error(fmt.Sprintf("Error reading from connection: %s", err.Error()))
					}
					return
				}
				// It's okay to see broken pipe errors since the client side could have already closed the connection
				if bounce {
					if _, err = conn.Write(buffer); err != nil {
						if !errors.Is(err, syscall.EPIPE) {
							zap.L().Error(fmt.Sprintf("Error writing echo to connection: %s", err.Error()))
						}
						return
					}
				} else {
					if _, err = conn.Write([]byte("OK")); err != nil {
						if !errors.Is(err, syscall.EPIPE) {
							zap.L().Error(fmt.Sprintf("Error writing OK to connection: %s", err.Error()))
						}
						return
					}
				}

			}
		}(conn, cfg.IsBounce())
	}
}
