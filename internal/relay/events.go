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

package relay

import "github.com/pairmesh/pairmesh/protocol"

// EventType represents the type of an event
type EventType byte

// EventType represents the type of an event
const (
	EventTypeClientClosed EventType = iota
	EventTypeClientConnected
)

type (
	// Event is a generic event item
	Event struct {
		Type EventType
		Data interface{}
	}

	// EventClientClosed is the event when a client is closed
	EventClientClosed struct {
		RelayServer protocol.RelayServer
		Client      *Client
	}

	// EventClientConnected is the event when a client is connected
	EventClientConnected struct {
		RelayServer protocol.RelayServer
		Client      *Client
	}
)
