//go:build windows

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

/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2021 WireGuard LLC. All Rights Reserved.
 */

package wintun

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Session struct {
	handle uintptr
}

const (
	PacketSizeMax   = 0xffff    // Maximum packet size
	RingCapacityMin = 0x20000   // Minimum ring capacity (128 kiB)
	RingCapacityMax = 0x4000000 // Maximum ring capacity (64 MiB)
)

// Packet with data
type Packet struct {
	Next *Packet              // Pointer to next packet in queue
	Size uint32               // Size of packet (max WINTUN_MAX_IP_PACKET_SIZE)
	Data *[PacketSizeMax]byte // Pointer to layer 3 IPv4 or IPv6 packet
}

var (
	procWintunAllocateSendPacket   = modwintun.NewProc("WintunAllocateSendPacket")
	procWintunEndSession           = modwintun.NewProc("WintunEndSession")
	procWintunGetReadWaitEvent     = modwintun.NewProc("WintunGetReadWaitEvent")
	procWintunReceivePacket        = modwintun.NewProc("WintunReceivePacket")
	procWintunReleaseReceivePacket = modwintun.NewProc("WintunReleaseReceivePacket")
	procWintunSendPacket           = modwintun.NewProc("WintunSendPacket")
	procWintunStartSession         = modwintun.NewProc("WintunStartSession")
)

func (wintun *Adapter) StartSession(capacity uint32) (session Session, err error) {
	r0, _, e1 := syscall.Syscall(procWintunStartSession.Addr(), 2, uintptr(wintun.handle), uintptr(capacity), 0)
	if r0 == 0 {
		err = e1
	} else {
		session = Session{r0}
	}
	return
}

func (session Session) End() {
	syscall.Syscall(procWintunEndSession.Addr(), 1, session.handle, 0, 0)
	session.handle = 0
}

func (session Session) ReadWaitEvent() (handle windows.Handle) {
	r0, _, _ := syscall.Syscall(procWintunGetReadWaitEvent.Addr(), 1, session.handle, 0, 0)
	handle = windows.Handle(r0)
	return
}

func (session Session) ReceivePacket() (packet []byte, err error) {
	var packetSize uint32
	r0, _, e1 := syscall.Syscall(procWintunReceivePacket.Addr(), 2, session.handle, uintptr(unsafe.Pointer(&packetSize)), 0)
	if r0 == 0 {
		err = e1
		return
	}
	packet = unsafe.Slice((*byte)(unsafe.Pointer(r0)), packetSize)
	return
}

func (session Session) ReleaseReceivePacket(packet []byte) {
	syscall.Syscall(procWintunReleaseReceivePacket.Addr(), 2, session.handle, uintptr(unsafe.Pointer(&packet[0])), 0)
}

func (session Session) AllocateSendPacket(packetSize int) (packet []byte, err error) {
	r0, _, e1 := syscall.Syscall(procWintunAllocateSendPacket.Addr(), 2, session.handle, uintptr(packetSize), 0)
	if r0 == 0 {
		err = e1
		return
	}
	packet = unsafe.Slice((*byte)(unsafe.Pointer(r0)), packetSize)
	return
}

func (session Session) SendPacket(packet []byte) {
	syscall.Syscall(procWintunSendPacket.Addr(), 2, session.handle, uintptr(unsafe.Pointer(&packet[0])), 0)
}
