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

type (
	// PeerID is the id of a peer
	PeerID uint64
	// UserID is the id of a user
	UserID uint64
	// NetworkID is the id of a network
	NetworkID uint64

	// Network is the struct of a network instance
	Network struct {
		ID    NetworkID `json:"id"`
		Name  string    `json:"name"`
		Peers []PeerID  `json:"peers"`
	}

	// Peer is the struct of a peer node instance
	Peer struct {
		ID       PeerID   `json:"id"`
		UserID   UserID   `json:"user_id"`
		Name     string   `json:"name,omitempty"`
		IPv4     string   `json:"ipv4"`
		ServerID ServerID `json:"server_id"`
		Active   bool     `json:"active"`
	}

	// PeerGraphResponse represents the topology of peers.
	PeerGraphResponse struct {
		// NotModified indicates the change status of peer graph and set to false
		// if no change from last access.
		NotModified bool   `json:"not_modified"`
		UniqueHash  string `json:"unique_hash"`
		// UpdateInterval indicates the interval of update peers graph from
		// portal service. <= 0 means use default interval.
		UpdateInterval int           `json:"update_interval"`
		RelayServers   []RelayServer `json:"relay_servers,omitempty"`
		Peers          []Peer        `json:"peers,omitempty"`
		Networks       []Network     `json:"networks,omitempty"`
	}

	// KeyExchangeResponse is response to requests to exchange key
	KeyExchangeResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	// PreflightRequest is request to gather information to handle preflight and get ready
	PreflightRequest struct {
		OS   string `json:"os"`
		Host string `json:"host"`
	}

	// PreflightResponse is the response to preflight requests with data needed
	PreflightResponse struct {
		ID PeerID `json:"id"`
		// User is the user who created the node. If ACL tags are in
		// use for the node then it doesn't reflect the ACL identity
		// that the node is running as.
		UserID          UserID
		Name            string      `json:"name"` // DNS
		IPv4            string      `json:"ipv4"`
		PrimaryServer   RelayServer `json:"primary_server"`
		Credential      string      `json:"credential"`
		CredentialLease uint64      `json:"credential_lease"`
	}

	// RenewCredentialRequest is used to request renew the credential
	RenewCredentialRequest struct {
		// current credential in BASE64 representation
		Credential string `json:"credential,omitempty"`
	}

	// RenewCredentialResponse is response to requests to renew a credential
	RenewCredentialResponse struct {
		// the renewed credential in BASE64 representation
		Credential      string `json:"credential,omitempty"`
		CredentialLease uint64 `json:"credential_lease"`
	}
)
