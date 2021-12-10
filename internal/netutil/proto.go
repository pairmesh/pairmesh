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

// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package netutil contains IP Protocol constants.
package netutil

import (
	"fmt"
	"net"
)

// Proto is an IP subprotocol as defined by the IANA protocol
// numbers list
// (https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml),
// or the special values Unknown or Fragment.
type Proto uint8

const (
	// Unknown represents an unknown or unsupported protocol; it's
	// deliberately the zero value. Strictly speaking the zero
	// value is IPv6 hop-by-hop extensions, but we don't support
	// those, so this is still technically correct.
	Unknown Proto = 0x00

	// Values from the IANA registry.
	ICMPv4 Proto = 0x01
	IGMP   Proto = 0x02
	ICMPv6 Proto = 0x3a
	TCP    Proto = 0x06
	UDP    Proto = 0x11
	SCTP   Proto = 0x84

	// MSMP is the Meshstep Message Protocol (our ICMP-ish
	// thing), an IP protocol used only between Tailscale nodes
	// (still encrypted by WireGuard) that communicates why things
	// failed, etc.
	//
	// Proto number 99 is reserved for "any private encryption
	// scheme". We never accept these from the host OS stack nor
	// send them to the host network stack. It's only used between
	// nodes.
	MSMP Proto = 99

	// Fragment represents any non-first IP fragment, for which we
	// don't have the sub-protocol header (and therefore can't
	// figure out what the sub-protocol is).
	//
	// 0xFF is reserved in the IANA registry, so we steal it for
	// internal use.
	Fragment Proto = 0xFF
)

func (p Proto) String() string {
	switch p {
	case Unknown:
		return "Unknown"
	case Fragment:
		return "Frag"
	case ICMPv4:
		return "ICMPv4"
	case IGMP:
		return "IGMP"
	case ICMPv6:
		return "ICMPv6"
	case UDP:
		return "UDP"
	case TCP:
		return "TCP"
	case SCTP:
		return "SCTP"
	case MSMP:
		return "MSMP"
	default:
		return fmt.Sprintf("IPProto-%d", int(p))
	}
}

// PickFreePort Random select a unused free tcp/udp port
func PickFreePort(p Proto) (int, error) {
	var (
		port        int
		err         error
		udpConn     *net.UDPConn
		tcpListener *net.TCPListener
	)
	switch p {
	case UDP:
		udpConn, err = net.ListenUDP("udp", &net.UDPAddr{})
		if err != nil {
			return -1, err
		}
		port = udpConn.LocalAddr().(*net.UDPAddr).Port
		udpConn.Close()

	case TCP:
		tcpListener, err = net.ListenTCP("tcp", &net.TCPAddr{})
		if err != nil {
			return -1, err
		}
		port = tcpListener.Addr().(*net.TCPAddr).Port
		tcpListener.Close()
	}
	return port, err

}
