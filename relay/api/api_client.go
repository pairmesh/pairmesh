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

package api

import (
	"time"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/internal/jsonapi"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/pairmesh/pairmesh/relay/config"
)

// Client is used to access with the remote gateway
type Client struct {
	restful *jsonapi.Client
}

// NewClient returns a new Client instance which can be used to interact
// with the gateway.
func NewClient(server string, authKey string) *Client {
	return &Client{
		restful: jsonapi.NewClient(server, authKey, ""),
	}
}

// Keepalive request the portal server to keepalive
func (c *Client) Keepalive(node *config.Config, peers []protocol.PeerID, startedAt time.Time) (*protocol.RelayKeepaliveResponse, error) {
	req := &protocol.RelayKeepaliveRequest{
		Name:      node.Name,
		Region:    node.Region,
		Host:      node.Host,
		Port:      node.Port,
		STUNPort:  node.STUNPort,
		PublicKey: node.DHKey.Public.String(),
		Peers:     peers,
		StartedAt: startedAt.UnixNano(),
	}

	res := &protocol.RelayKeepaliveResponse{}
	if err := c.restful.Post(constant.URIRelay, req, res); err != nil {
		return nil, err
	}

	return res, nil
}
