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

package jsonapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pairmesh/pairmesh/errcode"

	"github.com/pairmesh/pairmesh/constant"
	"github.com/pairmesh/pairmesh/pkg/logutil"
	"github.com/pairmesh/pairmesh/version"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// Client is used to access with the remote gateway
type Client struct {
	server    string
	token     *atomic.String
	machineid *atomic.String
}

// NewClient returns a new Client instance which can be used to interact
// with the gateway.
func NewClient(server, token, machineid string) *Client {
	return &Client{
		server:    server,
		token:     atomic.NewString(token),
		machineid: atomic.NewString(machineid),
	}
}

// SetToken sets token to the client
func (c *Client) SetToken(token string) {
	c.token.Store(token)
}

func (c *Client) do(method, api string, reader io.Reader, res interface{}) error {
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(c.server, "/"), strings.TrimPrefix(api, "/"))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return errors.WithStack(err)
	}

	// Set the req headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set(constant.HeaderXClientVersion, version.NewVersion().SemVer())
	req.Header.Set(constant.HeaderAuthentication, c.token.Load())
	req.Header.Set(constant.HeaderXMachineID, c.machineid.Load())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	defer resp.Body.Close()

	if logutil.IsEnablePortal() {
		zap.L().Debug("HTTP response", zap.String("method", method), zap.String("url", url))
	}

	if resp.StatusCode == http.StatusOK {
		return json.NewDecoder(resp.Body).Decode(res)
	}

	type response struct {
		Code  errcode.ErrCode `json:"code"`
		Error string          `json:"error"`
	}
	result := &response{}
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return err
	}

	return fmt.Errorf("%d: %s", result.Code, result.Error)
}

// Get is used to send the GET request
func (c *Client) Get(api string, res interface{}) error {
	if logutil.IsEnablePortal() {
		zap.L().Debug("HTTP Request", zap.String("method", "GET"), zap.String("url", api))
	}
	return c.do(http.MethodGet, api, nil, res)
}

// Post is used to send the POST request
func (c *Client) Post(api string, req, res interface{}) error {
	buffer := &bytes.Buffer{}
	err := json.NewEncoder(buffer).Encode(req)
	if err != nil {
		return errors.WithStack(err)
	}
	if logutil.IsEnablePortal() {
		zap.L().Debug("HTTP Request", zap.String("method", "POST"), zap.String("url", api), zap.String("data", buffer.String()))
	}
	return c.do(http.MethodPost, api, buffer, res)
}

// Put is used to send the PUT request
func (c *Client) Put(api string, req, res interface{}) error {
	buffer := &bytes.Buffer{}
	err := json.NewEncoder(buffer).Encode(req)
	if err != nil {
		return errors.WithStack(err)
	}
	if logutil.IsEnablePortal() {
		zap.L().Debug("HTTP Request", zap.String("method", "PUT"), zap.String("url", api), zap.String("data", buffer.String()))
	}
	return c.do(http.MethodPut, api, buffer, res)
}
