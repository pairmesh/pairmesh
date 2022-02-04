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

package protocol

// Internal protocols between PairMesh relay and portal
type (
	// ServerID represents the identifier of relay server.
	ServerID uint64

	// RelayServer describes a relay server information RelayRegion.
	RelayServer struct {
		ID ServerID `json:"id" yaml:"id"`

		// Name is a unique node name (across all regions).
		// It is not a host name.
		// It's typically of the form "1b", "2a", "3b", etc. (region
		// ID + suffix within that region)
		Name string `json:"name" yaml:"name"`

		// Region is the Region of the RelayRegion that this node
		// is running in.
		Region string `json:"region" yaml:"region"`

		// Host describes the host information about the relay server.
		Host string `json:"host,omitempty" yaml:"host,omitempty"`

		Port int `json:"port,omitempty" yaml:"port,omitempty"`

		// STUNPort optionally specifies a STUN port to use.
		// Zero means 3478.
		// To disable STUN on this node, use -1.
		// https://datatracker.ietf.org/doc/html/rfc5389#section-18.4
		STUNPort int `json:"stun_port" yaml:"stun_port"`

		// PublicKey represents the public key of DHKey pairs.
		PublicKey string `json:"public_key"`
	}

	RelayKeepaliveRequest struct {
		// Name is a unique node name (across all regions).
		// It is not a host name.
		// It's typically of the form "1b", "2a", "3b", etc. (region
		// ID + suffix within that region)
		Name string `json:"name"`

		// Region is the Region of the RelayRegion that this node
		// is running in.
		Region string `json:"region"`

		// Host describes the host information about the relay server.
		Host string `json:"host,omitempty" yaml:"host,omitempty"`

		Port int `json:"port,omitempty"`

		// Port optionally specifies a STUN port to use.
		// Zero means 3478.
		// To disable STUN on this node, use -1.
		// https://datatracker.ietf.org/doc/html/rfc5389#section-18.4
		STUNPort int `json:"stun_port,omitempty"`

		// PublicKey represents the public key of DHKey pairs.
		PublicKey string `json:"public_key"`

		Peers []PeerID `json:"peers"`

		// StartedAt represents the unix timestamp of relay server start time
		StartedAt int64 `json:"started_at,omitempty"`
	}

	RelayKeepaliveResponse struct {
		// PublicKey represents the public key which is used to validate the credential
		PublicKey  string `json:"public_key,omitempty"`
		SyncFailed bool   `json:"sync_failed"`
	}

	RelayPeerOfflineRequest struct {
		Peers []PeerID `json:"peers"`
	}

	RelayPeerOfflineResponse struct {
	}
)
