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
	"sort"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/jsonapi"
	"github.com/pairmesh/pairmesh/protocol"
	"go.uber.org/zap"
)

// Client is used to access with the remote gateway
type Client struct {
	restful *jsonapi.Client
}

// New returns a new Client instance which can be used to interact
// with the gateway.
func New(server, token, machineid string) *Client {
	return &Client{
		restful: jsonapi.NewClient(server, token, machineid),
	}
}

func (c *Client) SetToken(key string) {
	c.restful.SetToken(key)
}

// Logout logout the current  node
func (c *Client) Logout() {
	err := c.restful.Get(constant.URILogout, nil)
	if err != nil {
		zap.L().Error("Error sending logout message")
	}
}

// Preflight request the prerequisite for bootup the current node
func (c *Client) Preflight(os, hostname string) (*protocol.PreflightResponse, error) {
	req := &protocol.PreflightRequest{
		OS:   os,
		Host: hostname,
	}
	resp := &protocol.PreflightResponse{}

	if err := c.restful.Post(constant.URIDevicePreflight, req, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// PeerGraph requests the server to open device and retrieve the peers
// of the current node
func (c *Client) PeerGraph(uniqueHash string) (*protocol.PeerGraphResponse, error) {
	resp := &protocol.PeerGraphResponse{}

	if err := c.restful.Get(constant.URIDevicePeerGraph+"?hash="+uniqueHash, resp); err != nil {
		return nil, err
	}

	sort.Slice(resp.Peers, func(i, j int) bool {
		return resp.Peers[i].ID < resp.Peers[j].ID
	})
	return resp, nil
}

// KeyExchange use the auth key to exchange back a jwt token
func (c *Client) KeyExchange() (*protocol.KeyExchangeResponse, error) {
	var resp protocol.KeyExchangeResponse
	err := c.restful.Post(constant.URLKeyExchange, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RenewCredential request the server to renew the credential.
func (c *Client) RenewCredential(credential string) (*protocol.RenewCredentialResponse, error) {
	req := &protocol.RenewCredentialRequest{
		Credential: credential,
	}
	res := &protocol.RenewCredentialResponse{}
	err := c.restful.Post(constant.URIRenewCredential, req, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
