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

// Package serde is used to serialize/deserialize network packets.
package serde

import (
	"fmt"
	"reflect"

	"github.com/pairmesh/pairmesh/message"

	"google.golang.org/protobuf/proto"
)

// sharedProtos is the type repositories for en/decode
var sharedProtos = []reflect.Type{
	message.PacketType_Handshake:     reflect.TypeOf(&message.PacketHandshake{}),
	message.PacketType_HandshakeAck:  reflect.TypeOf(&message.PacketHandshakeAck{}),
	message.PacketType_Heartbeat:     reflect.TypeOf(&message.PacketHeartbeat{}),
	message.PacketType_ProbeRequest:  reflect.TypeOf(&message.PacketProbeRequest{}),
	message.PacketType_ProbeResponse: reflect.TypeOf(&message.PacketProbeResponse{}),
	message.PacketType_SyncPeer:      reflect.TypeOf(&message.PacketSyncPeer{}),
	message.PacketType_Forward:       reflect.TypeOf(&message.PacketForward{}),
	message.PacketType_Discovery:     reflect.TypeOf(&message.PacketDiscovery{}),

	// Unit test
	message.PacketType__UnitTestRequest:  reflect.TypeOf(&message.P_UnitTestRequest{}),
	message.PacketType__UnitTestResponse: reflect.TypeOf(&message.P_UnitTestResponse{}),
}

// Deserialize function deserializes input message from bytes to formatted proto.Message
func Deserialize(t message.PacketType, buf []byte) (proto.Message, error) {
	if int(t) >= len(sharedProtos) || sharedProtos[int(t)] == nil {
		return nil, fmt.Errorf("unrecognized protobuf type: %v", t)
	}

	protoType := reflect.New(sharedProtos[t].Elem()).Interface().(proto.Message)
	err := proto.Unmarshal(buf[:], protoType)
	if err != nil {
		return nil, fmt.Errorf("unmarshal for type: %v is failed with error: %w", t, err)
	}
	return protoType, nil
}
