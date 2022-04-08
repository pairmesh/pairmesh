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

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pairmesh/pairmesh/message"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// Client represents the relay server client which is used to interactive with relay server.
type Client struct {
	ClientTransporter

	handler  ClientHandler
	latency  time.Duration
	closed   *atomic.Bool
	onClosed func() // Callback function
}

// NewClient returns a new client instance.
func NewClient(transporter ClientTransporter) *Client {
	c := &Client{
		ClientTransporter: transporter,
		closed:            atomic.NewBool(false),
		handler:           NewClientHandler(),
	}
	return c
}

// SetLatency sets c.latency = lat
func (c *Client) SetLatency(lat time.Duration) {
	c.latency = lat
}

// Handler return c.handler
func (c *Client) Handler() ClientHandler {
	return c.handler
}

// Send sends a given packet by pushing it into write queue
func (c *Client) Send(typ message.PacketType, msg proto.Message) (err error) {
	if c.closed.Load() {
		return errors.New("cannot send message to closed client")
	}

	// Prevents write data to a closed channel
	defer func() {
		if e := recover(); e != nil {
			e1, ok := e.(error)
			if !ok {
				err = fmt.Errorf("send data panicked: %v", e)
			} else {
				err = e1
			}
		}
	}()

	select {
	case c.WriteQueue() <- Packet{Type: typ, Message: msg}:
		return nil
	default:
		err = fmt.Errorf("send buffer exceeded: %s:%d", c.RelayServer().Host, c.RelayServer().Port)
	}
	return
}

// Serve starts a job to detect from read queue, and handles the data according to c.handler registrations
func (c *Client) Serve(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case p, ok := <-c.ReadQueue():
			if !ok {
				return
			}
			err := c.handler.Handle(c, p)
			if err != nil {
				zap.L().Error("Handle relay message failed", zap.Stringer("type", p.Type), zap.Error(err))
				continue
			}

		}
	}
}

// OnClosed updates callback function on client close
func (c *Client) OnClosed(cb func()) {
	c.onClosed = cb
}

// Close actually closes the client
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return errors.New("close a closed client")
	}
	if err := c.ClientTransporter.Close(); err != nil {
		return err
	}
	if c.onClosed != nil {
		c.onClosed()
	}
	return nil
}
