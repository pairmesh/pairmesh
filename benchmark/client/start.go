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

	"go.uber.org/zap"
)

const timeout = 3

// Start function connects to endpoint host and start performance testing
func Start(cfg *Config) error {
	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup
	timer := time.NewTimer(time.Duration(cfg.Duration()) * time.Second)
	notify := make(chan interface{})

	results := NewResults()

	var i uint16
	for i = 0; uint16(i) < cfg.Clients(); i++ {
		wg.Add(1)
		go func(
			wg *sync.WaitGroup,
			index uint16,
			cfg *Config,
			notify <-chan interface{},
			results *Results,
		) {
			defer wg.Done()

			zap.L().Info(fmt.Sprintf("[worker %d] test job started", index))
			addr := fmt.Sprintf("%s:%d", cfg.Endpoint(), cfg.Port())
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				zap.L().Error(fmt.Sprintf("[worker %d] error dialing to endpoint: %s", index, err.Error()))
				return
			}

			lresults := NewResults()

			defer func() {
				conn.Close()
				results.Submit(&lresults)
			}()

			payload := generateRandPayload(cfg.Payload())

			for {
				select {
				case <-notify:
					zap.L().Info(fmt.Sprintf("[worker %d] test job finished", index))
					return
				default:
					buf := make([]byte, 2*cfg.Payload())
					conn.SetDeadline(time.Now().Add(timeout * time.Second))
					prev_time := time.Now()
					if _, err := conn.Write(payload); err != nil {
						zap.L().Error(fmt.Sprintf("[worker %d] error writing to server: %s", index, err.Error()))
						return
					}

					if _, err = conn.Read(buf); err != nil {
						zap.L().Error(fmt.Sprintf("[worker %d] error reading from server: %s", index, err.Error()))
						return
					}
					post_time := time.Now()
					delta := post_time.Sub(prev_time)
					lresults.AddDataPoint(delta)
				}
			}
		}(&wg, i, cfg, notify, &results)
	}

	// time.C only sends one signal, without closing
	// Therefore it could not be shared between goroutines
	// So just use notify channel to represent the timer
	<-timer.C
	close(notify)
	wg.Wait()

	results.Report(cfg)

	return nil
}

func generateRandPayload(plen uint32) []byte {
	var alp = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, plen)
	for i := range b {
		b[i] = alp[rand.Intn(len(alp))]
	}
	return b
}
