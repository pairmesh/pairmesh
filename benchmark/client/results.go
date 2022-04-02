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

package client

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Results struct reprents the general testing results from client view
type Results struct {
	l          sync.Mutex
	latencies  []time.Duration
	throughput uint64
}

func NewResults() Results {
	return Results{
		latencies:  make([]time.Duration, 0),
		throughput: 0,
	}
}

func (r *Results) Submit(res *Results) {
	r.l.Lock()
	defer r.l.Unlock()

	r.latencies = append(r.latencies, res.latencies...)
	r.throughput += res.throughput
}

func (r *Results) AddDataPoint(lat time.Duration) {
	r.l.Lock()
	defer r.l.Unlock()

	r.latencies = append(r.latencies, lat)
	r.throughput += 1
}

func (r *Results) Report(cfg *Config) {
	r.l.Lock()
	defer r.l.Unlock()

	sort.Slice(r.latencies, func(i, j int) bool {
		return r.latencies[i] < r.latencies[j]
	})

	tps := r.throughput / uint64(cfg.duration)
	llen := len(r.latencies)
	p50 := r.latencies[int(float64(llen)*0.5)]
	p90 := r.latencies[int(float64(llen)*0.9)]
	p99 := r.latencies[int(float64(llen)*0.99)]

	zap.L().Info(fmt.Sprintf("The total throughput is %d\n", r.throughput))
	zap.L().Info(fmt.Sprintf("The TPS is %d\n", tps))
	zap.L().Info(fmt.Sprintf("The P50 latency is %s\n", p50))
	zap.L().Info(fmt.Sprintf("The P90 latency is %s\n", p90))
	zap.L().Info(fmt.Sprintf("The P99 latency is %s\n", p99))
}
