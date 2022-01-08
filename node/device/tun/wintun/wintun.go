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
	"go.uber.org/zap"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type loggerLevel int

const (
	logInfo loggerLevel = iota
	logWarn
	logErr
)

const AdapterNameMax = 128

type Adapter struct {
	handle uintptr
}

var (
	modwintun                         = newLazyDLL("wintun.dll", setupLogger)
	procWintunCreateAdapter           = modwintun.NewProc("WintunCreateAdapter")
	procWintunOpenAdapter             = modwintun.NewProc("WintunOpenAdapter")
	procWintunCloseAdapter            = modwintun.NewProc("WintunCloseAdapter")
	procWintunDeleteDriver            = modwintun.NewProc("WintunDeleteDriver")
	procWintunGetAdapterLUID          = modwintun.NewProc("WintunGetAdapterLUID")
	procWintunGetRunningDriverVersion = modwintun.NewProc("WintunGetRunningDriverVersion")
)

type TimestampedWriter interface {
	WriteWithTimestamp(p []byte, ts int64) (n int, err error)
}

func logMessage(level loggerLevel, timestamp uint64, msg *uint16) int {
	zap.L().Info(windows.UTF16PtrToString(msg), zap.Int64("ts", (int64(timestamp)-116444736000000000)*100))
	return 0
}

func setupLogger(dll *lazyDLL) {
	var callback uintptr
	if runtime.GOARCH == "386" {
		callback = windows.NewCallback(func(level loggerLevel, timestampLow, timestampHigh uint32, msg *uint16) int {
			return logMessage(level, uint64(timestampHigh)<<32|uint64(timestampLow), msg)
		})
	} else if runtime.GOARCH == "arm" {
		callback = windows.NewCallback(func(level loggerLevel, _, timestampLow, timestampHigh uint32, msg *uint16) int {
			return logMessage(level, uint64(timestampHigh)<<32|uint64(timestampLow), msg)
		})
	} else if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		callback = windows.NewCallback(logMessage)
	}
	syscall.Syscall(dll.NewProc("WintunSetLogger").Addr(), 1, callback, 0, 0)
}

func closeAdapter(wintun *Adapter) {
	syscall.Syscall(procWintunCloseAdapter.Addr(), 1, wintun.handle, 0, 0)
}

// CreateAdapter creates a Wintun adapter. name is the cosmetic name of the adapter.
// tunnelType represents the type of adapter and should be "Wintun". requestedGUID is
// the GUID of the created network adapter, which then influences NLA generation
// deterministically. If it is set to nil, the GUID is chosen by the system at random,
// and hence a new NLA entry is created for each new adapter.
func CreateAdapter(name string, tunnelType string, requestedGUID *windows.GUID) (wintun *Adapter, err error) {
	var name16 *uint16
	name16, err = windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	var tunnelType16 *uint16
	tunnelType16, err = windows.UTF16PtrFromString(tunnelType)
	if err != nil {
		return
	}
	r0, _, e1 := syscall.Syscall(procWintunCreateAdapter.Addr(), 3, uintptr(unsafe.Pointer(name16)), uintptr(unsafe.Pointer(tunnelType16)), uintptr(unsafe.Pointer(requestedGUID)))
	if r0 == 0 {
		err = e1
		return
	}
	wintun = &Adapter{handle: r0}
	runtime.SetFinalizer(wintun, closeAdapter)
	return
}

// OpenAdapter opens an existing Wintun adapter by name.
func OpenAdapter(name string) (wintun *Adapter, err error) {
	var name16 *uint16
	name16, err = windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	r0, _, e1 := syscall.Syscall(procWintunOpenAdapter.Addr(), 1, uintptr(unsafe.Pointer(name16)), 0, 0)
	if r0 == 0 {
		err = e1
		return
	}
	wintun = &Adapter{handle: r0}
	runtime.SetFinalizer(wintun, closeAdapter)
	return
}

// Close closes a Wintun adapter.
func (wintun *Adapter) Close() (err error) {
	runtime.SetFinalizer(wintun, nil)
	r1, _, e1 := syscall.Syscall(procWintunCloseAdapter.Addr(), 1, wintun.handle, 0, 0)
	if r1 == 0 {
		err = e1
	}
	return
}

// Uninstall removes the driver from the system if no drivers are currently in use.
func Uninstall() (err error) {
	r1, _, e1 := syscall.Syscall(procWintunDeleteDriver.Addr(), 0, 0, 0, 0)
	if r1 == 0 {
		err = e1
	}
	return
}

// RunningVersion returns the version of the loaded driver.
func RunningVersion() (version uint32, err error) {
	r0, _, e1 := syscall.Syscall(procWintunGetRunningDriverVersion.Addr(), 0, 0, 0, 0)
	version = uint32(r0)
	if version == 0 {
		err = e1
	}
	return
}

// LUID returns the LUID of the adapter.
func (wintun *Adapter) LUID() (luid uint64) {
	syscall.Syscall(procWintunGetAdapterLUID.Addr(), 2, uintptr(wintun.handle), uintptr(unsafe.Pointer(&luid)), 0)
	return
}
