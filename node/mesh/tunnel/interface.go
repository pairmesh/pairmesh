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

import "net"

// FragmentCallback represents the callback of receiving fragment data.
type FragmentCallback interface {
	// OnFragment will be called if there are fragments received from the
	// low-level mesh network.
	OnFragment(data []byte)
}

// UDPPacketCallback represents the callback of UDP packets.
// There are two ways to read UDP packets:
// 1. UDP packets from relay server (PacketConnect/PacketFragment/Other system packets).
// 2. UDP packets from peers (PacketKeepalive/PacketFragment).
type UDPPacketCallback interface {
	// OnUDPPacket will be called if there are UDP packets received from
	// the low-level UDPConn connections.
	OnUDPPacket(udpConn *net.UDPConn, data []byte)
}
