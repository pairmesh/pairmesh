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

package peer

import (
	"math"
	"sync"
	"time"

	"github.com/pairmesh/pairmesh/node/mesh/tunnel"
	"github.com/pairmesh/pairmesh/protocol"
)

const (
	probeInterval   = 600 * time.Second
	catchupInterval = 30 * time.Second
)

type (
	// ProbeInfo represents the probe information about the peer.
	ProbeInfo struct {
		LastProbeRequestAt  time.Time
		LastProbeResponseAt time.Time
		IsOnline            bool
	}

	// CatchupInfo represents the catchup information about the peer.
	CatchupInfo struct {
		LastSendCatchupAt time.Time
		CatchupAt         time.Time
		IsCatchup         bool
		NoAckCounter      int
	}

	// Peer represents the peer node of the mesh network.
	Peer struct {
		// Readonly fields and no lock protection.
		info   protocol.Peer
		tunnel *tunnel.Tunnel

		mu      sync.RWMutex
		probe   ProbeInfo
		catchup CatchupInfo
	}
)

func New(peerInfo protocol.Peer) *Peer {
	return &Peer{info: peerInfo}
}

func (p *Peer) IPv4() string {
	return p.info.IPv4
}

func (p *Peer) ID() protocol.PeerID {
	return p.info.ID
}

func (p *Peer) PrimaryServerID() protocol.ServerID {
	return p.info.ServerID
}

func (p *Peer) SetProbeStatus(isOnline bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.probe.IsOnline = isOnline
	p.probe.LastProbeResponseAt = time.Now()
}

func (p *Peer) IsCatchup() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.catchup.IsCatchup
}

func (p *Peer) IsNeedCatchup() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.catchup.IsCatchup {
		return false
	}

	// Exponential catchup retry duration.
	expo := math.Pow(float64(p.catchup.NoAckCounter)+1, 2)
	retryCatchupInterval := time.Duration(expo) * catchupInterval
	return time.Since(p.catchup.LastSendCatchupAt) > retryCatchupInterval
}

func (p *Peer) IsNeedProbe() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return time.Since(p.probe.LastProbeRequestAt) > probeInterval
}

func (p *Peer) SetLastSendCatchupAt(at time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.catchup.NoAckCounter++
	p.catchup.LastSendCatchupAt = at
}

func (p *Peer) SetCatchupAt(now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.catchup.IsCatchup = true
	p.catchup.NoAckCounter = 0
	p.catchup.CatchupAt = now
}

func (p *Peer) SetLastProbeRequestAt(at time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.probe.LastProbeRequestAt = at
}

func (p *Peer) SetTunnel(t *tunnel.Tunnel) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tunnel != nil {
		p.tunnel.Close()
	}

	p.tunnel = t
}

func (p *Peer) Tunnel() *tunnel.Tunnel {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.tunnel
}

func (p *Peer) PeerInfo() protocol.Peer {
	return p.info
}

// Close destroy the remote peer resources.
func (p *Peer) Close() {
	p.tunnel.Close()
}
