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
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/bench/config"
	"github.com/pairmesh/pairmesh/bench/results"
	"github.com/pairmesh/pairmesh/bench/utils"
	"go.uber.org/zap"
)

const timeout = 3

// EchoClient is the client struct
type EchoClient struct {
	cfg *config.ClientConfig
}

// NewEchoClient returns EchoClient struct with input config
func NewEchoClient(cfg *config.ClientConfig) *EchoClient {
	return &EchoClient{
		cfg: cfg,
	}
}

// Start function starts echo test cases from client side
func (c *EchoClient) Start() error {
	rand.Seed(time.Now().UnixNano())

	cfg := c.cfg

	var wg sync.WaitGroup
	timer := time.NewTimer(time.Duration(cfg.Duration()) * time.Second)
	notify := make(chan struct{})

	res := results.NewResults()

	var i uint16
	for i = 0; uint16(i) < cfg.Clients(); i++ {
		wg.Add(1)
		go func(index uint16) {
			defer wg.Done()

			zap.L().Info(fmt.Sprintf("[worker %d] test job started", index))
			addr := fmt.Sprintf("%s:%d", cfg.Endpoint(), cfg.Port())
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				zap.L().Error(fmt.Sprintf("[worker %d] error dialing to endpoint: %s", index, err.Error()))
				return
			}

			lres := results.NewResults()

			defer func() {
				conn.Close()
				res.Submit(&lres)
			}()

			payload := utils.GenerateRandPayload(cfg.Payload())

			var thres uint16
			if cfg.IsBounce() {
				thres = cfg.Payload()
			} else {
				thres = 2 // just the length of "OK"
			}

			for {
				select {
				case <-notify:
					zap.L().Info(fmt.Sprintf("[worker %d] test job finished", index))
					return
				default:
					err = conn.SetDeadline(time.Now().Add(timeout * time.Second))
					if err != nil {
						zap.L().Error(fmt.Sprintf("[worker %d] error setting deadline to connection: %s", index, err.Error()))
						return
					}
					prevTime := time.Now()
					_, err := conn.Write(payload)
					if err != nil {
						zap.L().Error(fmt.Sprintf("[worker %d] error writing to server: %s", index, err.Error()))
						return
					}

					// TODO: definitely need a smarter way of handling Read
					// This is actually related to how to
					// accurately measure Read(). Sometimes when buf is too small, the following
					// Read() would immediately return with empty buf.
					var rcount uint16 = 0
					for {
						buf := make([]byte, 512)
						_, err = conn.Read(buf)
						if err != nil {
							zap.L().Error(fmt.Sprintf("[worker %d] error reading from server: %s", index, err.Error()))
							return
						}
						for i := 0; i < len(buf); i++ {
							if buf[i] == byte(0) {
								break
							}
							rcount++
						}

						if rcount == thres {
							break
						}
					}

					postTime := time.Now()
					delta := postTime.Sub(prevTime)
					lres.AddDataPoint(delta)
				}
			}
		}(i)
	}

	// time.C only sends one signal, without closing
	// Therefore it could not be shared between goroutines
	// So just use notify channel to represent the timer
	<-timer.C
	close(notify)
	wg.Wait()

	res.Report(cfg)

	return nil
}
