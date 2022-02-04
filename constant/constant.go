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

package constant

import "time"

// PluginKeyType represents the unique type of plugin key
type PluginKeyType string

const KeyRawRequest = PluginKeyType("_plugin_key_request")

const MachineIDProtect = "pairmesh"

const EnvLogLevel = "PAIRMESH_LOG_VERBOSE"

// API path group
const (
	URIDevicePeerGraph = "/api/v1/device/peers"
	URIDevicePreflight = "/api/v1/device/preflight"
	URIRelay           = "/api/v1/relay"
	URIPeersOffline    = "/api/v1/relay/peers"
	URILogout          = "/api/v1/logout"
	URLKeyExchange     = "/api/v1/key/exchange"
	URIRenewCredential = "/api/v1/credential/renew"
)

// HTTP header constants
const (
	HeaderAuthentication = "Authorization"
	HeaderXClientVersion = "X-PairMesh-Version"
	HeaderXMachineID     = "X-PairMesh-Machine-ID"
)

// token prefix
const (
	PrefixAuthKey  = "AuthKey"
	PrefixFastKey  = "FastKey"
	PrefixJwtToken = "Bearer"
)

// DefaultAPIGateway represents the gateway's default address
const DefaultAPIGateway = "https://api.pairmesh.com"
const DefaultMyGateway = "https://my.pairmesh.com"

// Packet protocol constants

// Relay packet format:
// | nonce(4bytes) | type(2bytes) | payload size(4bytes) | payload |

// Fragment packet format:
// | nonce(4bytes) | type(2bytes) | peer_id(8bytes) | payload |

const (
	HeaderNonceSize      = 4
	HeaderPacketTypeSize = 2
	PacketHeaderSize     = 10
	FragmentHeaderSize   = 14
)

// MaxBufferSize represents the max buffer size of read UDP packet
const MaxBufferSize = 4096

// DiscoveryDuration represents the interval of retrying send
// initial Ping packet to the peer
const DiscoveryDuration = 30 * time.Second

const (
	MaxSegmentSize = 2048 - 32      // largest possible UDP datagram
	MaxMessageSize = MaxSegmentSize // maximum size of transport message
)
const HeartbeatInterval = 30 * time.Second
