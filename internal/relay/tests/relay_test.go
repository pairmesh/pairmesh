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

package tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/flynn/noise"
	"github.com/pairmesh/pairmesh/internal/netutil"
	"github.com/pairmesh/pairmesh/internal/relay"
	"github.com/pairmesh/pairmesh/message"
	"github.com/pairmesh/pairmesh/protocol"
	"github.com/pairmesh/pairmesh/security"
	"github.com/pairmesh/pairmesh/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
)

func TestRelay(t *testing.T) {
	// Preflight the logger
	config := zap.NewDevelopmentEncoderConfig()
	encoder := zapcore.NewConsoleEncoder(config)
	logger := zap.New(zapcore.NewCore(encoder, os.Stdout, zap.DebugLevel))
	zap.ReplaceGlobals(logger)

	port, err := netutil.PickFreePort(netutil.TCP)

	assert.Nil(t, err)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// Generate test keys
	serverDHKey, err := noise.DH25519.GenerateKeypair(rand.Reader)
	assert.Nil(t, err)

	clientDHKey, err := noise.DH25519.GenerateKeypair(rand.Reader)
	assert.Nil(t, err)

	priv, err := rsa.GenerateKey(rand.Reader, 512)
	assert.Nil(t, err)

	server := relay.NewServer(addr, 5*time.Second, serverDHKey, &priv.PublicKey)

	// Register customize callback
	server.Handler().On(message.PacketType__UnitTestRequest, func(s *relay.Session, typ message.PacketType, msg proto.Message) error {
		res := &message.P_UnitTestResponse{Field: msg.(*message.P_UnitTestRequest).Field}
		return s.Send(message.PacketType__UnitTestResponse, res)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zap.L().Info("Starting server")

	go func() {
		// Ignore Server errors here
		_ = server.Serve(ctx)
	}()

	// Wait for the server to get up running and ready to accept connections
	assert.True(t, utils.WaitForServerUp(addr))

	// Generate mock credentials
	credentials, err := security.Credential(priv, protocol.UserID(1), protocol.PeerID(11000), net.ParseIP("1.2.3.4"), time.Hour)
	assert.Nil(t, err)

	relayServer := protocol.RelayServer{
		Host: "127.0.0.1",
		Port: port,
	}
	trs := relay.NewClientTransporter(relayServer, credentials, clientDHKey, security.NewDHPublic(serverDHKey.Public))
	client := relay.NewClient(trs)
	go client.Serve(ctx)

	const iter = 5
	var counter = 0
	chWait := make(chan string, iter)
	client.Handler().On(message.PacketType__UnitTestResponse, func(s *relay.Client, typ message.PacketType, msg proto.Message) error {
		res := msg.(*message.P_UnitTestResponse)
		assert.Equal(t, fmt.Sprintf("magic-%d", counter), res.Field)
		counter++
		chWait <- res.Field
		return nil
	})

	err = client.Connect(ctx)
	assert.Nil(t, err)

	for i := 0; i < iter; i++ {
		// Use client to send a message to server
		err = client.Send(message.PacketType__UnitTestRequest, &message.P_UnitTestRequest{Field: fmt.Sprintf("magic-%d", i)})
		assert.Nil(t, err)
	}

	// Wait server response the request message.
	for i := 0; i < iter; i++ {
		m := <-chWait
		assert.Equal(t, m, fmt.Sprintf("magic-%d", i))
	}
}

// Test in the scenario that when a session sees network connection failure somehow,
// It closes properly and the server would evict it from the session map.
func TestRelayNetworkFailure(t *testing.T) {
	// Preflight the logger
	config := zap.NewDevelopmentEncoderConfig()
	encoder := zapcore.NewConsoleEncoder(config)
	logger := zap.New(zapcore.NewCore(encoder, os.Stdout, zap.DebugLevel))
	zap.ReplaceGlobals(logger)

	port, err := netutil.PickFreePort(netutil.TCP)

	assert.Nil(t, err)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// Generate test keys
	serverDHKey, err := noise.DH25519.GenerateKeypair(rand.Reader)
	assert.Nil(t, err)

	clientDHKey, err := noise.DH25519.GenerateKeypair(rand.Reader)
	assert.Nil(t, err)

	priv, err := rsa.GenerateKey(rand.Reader, 512)
	assert.Nil(t, err)

	server := relay.NewServer(addr, 5*time.Second, serverDHKey, &priv.PublicKey)

	// Register customize callback
	server.Handler().On(message.PacketType__UnitTestRequest, func(s *relay.Session, typ message.PacketType, msg proto.Message) error {
		res := &message.P_UnitTestResponse{Field: msg.(*message.P_UnitTestRequest).Field}
		return s.Send(message.PacketType__UnitTestResponse, res)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zap.L().Info("Starting server")

	go func() {
		// Ignore Server errors here
		_ = server.Serve(ctx)
	}()

	// Wait for the server to get up running and ready to accept connections
	assert.True(t, utils.WaitForServerUp(addr))

	peerID := protocol.PeerID(11000)

	// Generate mock credentials
	credentials, err := security.Credential(priv, protocol.UserID(1), peerID, net.ParseIP("1.2.3.4"), time.Hour)
	assert.Nil(t, err)

	relayServer := protocol.RelayServer{
		Host: "127.0.0.1",
		Port: port,
	}
	trs := relay.NewClientTransporter(relayServer, credentials, clientDHKey, security.NewDHPublic(serverDHKey.Public))
	client := relay.NewClient(trs)
	go client.Serve(ctx)

	const iter = 5
	var counter = 0
	chWait := make(chan string, iter)
	client.Handler().On(message.PacketType__UnitTestResponse, func(s *relay.Client, typ message.PacketType, msg proto.Message) error {
		res := msg.(*message.P_UnitTestResponse)
		assert.Equal(t, fmt.Sprintf("magic-%d", counter), res.Field)
		counter++
		chWait <- res.Field
		return nil
	})

	err = client.Connect(ctx)
	assert.Nil(t, err)

	session := server.Session(peerID)
	assert.True(t, session != nil)

	// Assume that the session conn somehow disconnected after some time.
	time.Sleep(time.Duration(1 * time.Second))
	session.SessionTransporter.Close()

	time.Sleep(time.Duration(1 * time.Second))
	// This should have triggered the onSessionClosed() function. So the session should be evicted from server.
	assert.True(t, server.Session(peerID) == nil)
}
