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

package logutil

import (
	"os"
	"strings"

	"github.com/pairmesh/pairmesh/constant"
	"go.uber.org/zap/zapcore"
)

// bits are used to check whether output verbose log.
var bits = 0

func init() {
	v, ok := os.LookupEnv(constant.EnvLogLevel)
	if ok {
		v = strings.ToLower(v)
		if v == "all" {
			EnableAll()
		} else {
			parts := strings.Split(v, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				switch p {
				case "portal":
					Enable(DebugPortalLevel)
				case "relay":
					Enable(DebugRelayPacket)
				case "peer":
					Enable(DebugPeerPacket)
				case "device":
					Enable(DebugDevicePacket)
				}
			}
		}
	}
}

type Type byte

const (
	// DebugPortalLevel indicates HTTP the request between node and gateway
	DebugPortalLevel Type = 0
	// DebugRelayPacket indicates UDP packet message between node and gateway
	DebugRelayPacket Type = 1
	// DebugPeerPacket indicates UDP packet message between peers
	DebugPeerPacket Type = 2
	// DebugDevicePacket indicates Packets read from/write into virtual device
	DebugDevicePacket Type = 3
)

// Enable enables the output of some types of verbose log.
func Enable(t Type) {
	bits |= 1 << t
}

func EnableAll() {
	for _, l := range []Type{DebugPortalLevel, DebugRelayPacket, DebugPeerPacket, DebugDevicePacket} {
		Enable(l)
	}
}

// Level returns the log level corresponding to the verbosity level
func Level() zapcore.Level {
	if bits > 0 {
		return zapcore.DebugLevel
	}
	return zapcore.InfoLevel
}

// IsEnablePortal checks if http request between portal and node debug logs enabled.
func IsEnablePortal() bool {
	return bits&(1<<DebugPortalLevel) > 0
}

// IsEnableRelay checks if packet between relay and node device debug logs enabled.
func IsEnableRelay() bool {
	return bits&(1<<DebugRelayPacket) > 0
}

// IsEnablePeer checks if packet between peers debug logs enabled.
func IsEnablePeer() bool {
	return bits&(1<<DebugPeerPacket) > 0
}

// IsEnableDevice checks if device debug logs enabled.
func IsEnableDevice() bool {
	return bits&(1<<DebugDevicePacket) > 0
}
