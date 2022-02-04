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

type EventType byte

const (
	EventTypeClientClosed EventType = iota
	EventTypeClientConnected
	EventTypeSessionClosed
	EventTypeSessionConnected
)

type (
	Event struct {
		Type EventType
		Data interface{}
	}

	EventClientClosed struct {
		RelayServer protocol.RelayServer
		Client      *Client
	}

	EventClientConnected struct {
		RelayServer protocol.RelayServer
		Client      *Client
	}

	EventSessionClosed struct {
		Session *Session
	}

	EventSessionConnected struct {
		Session *Session
	}
)
