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

// SPDX-License-Identifier: MIT
//
// Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
//

package tun

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	// import unsafe
	_ "unsafe"

	"github.com/pairmesh/pairmesh/node/device/tun/wintun"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

const (
	rateMeasurementGranularity = uint64((time.Second / 2) / time.Nanosecond)
	spinloopRateThreshold      = 800000000 / 8                                   // 800mbps
	spinloopDuration           = uint64(time.Millisecond / 80 / time.Nanosecond) // ~1gbit/s
)

type rateJuggler struct {
	current       uint64
	nextByteCount uint64
	nextStartTime int64
	changing      int32
}

type windowsDevice struct {
	wt        *wintun.Adapter
	name      string
	handle    windows.Handle
	rate      rateJuggler
	session   wintun.Session
	readWait  windows.Handle
	running   sync.WaitGroup
	closeOnce sync.Once
	close     int32
	forcedMTU int
}

var (
	WintunTunnelName          = "PairMesh"
	WintunTunnelType          = "PairMesh"
	WintunStaticRequestedGUID *windows.GUID
)

func init() {
	guid, err := windows.GUIDFromString("{418099dd-0ee3-4624-96ae-fef1070f8777}")
	if err != nil {
		panic(err)
	}
	WintunStaticRequestedGUID = &guid
}

//go:linkname procyield runtime.procyield
func procyield(cycles uint32)

//go:linkname nanotime runtime.nanotime
func nanotime() int64

// NewTUN creates a new TUN device and set the address to the specified address
// It will creates a Wintun interface with the given name. Should a Wintun
// interface with the same name exist, it is reused.
func NewTUN() (Device, error) {
	// Remove the previous adapter
	if err := wintun.Uninstall(); err != nil {
		return nil, err
	}

	wt, err := wintun.CreateAdapter(WintunTunnelName, WintunTunnelType, WintunStaticRequestedGUID)
	if err != nil {
		return nil, fmt.Errorf("Error creating interface: %w", err)
	}

	dev := &windowsDevice{
		name:      WintunTunnelName,
		wt:        wt,
		handle:    windows.InvalidHandle,
		forcedMTU: DefaultMTU,
	}

	dev.session, err = wt.StartSession(0x800000) // Ring capacity, 8 MiB
	if err != nil {
		wt.Close()
		return nil, fmt.Errorf("Error starting session: %w", err)
	}
	dev.readWait = dev.session.ReadWaitEvent()
	return dev, err
}

func (d *windowsDevice) Name() string {
	return d.name
}

func (d *windowsDevice) Close() error {
	var err error
	d.closeOnce.Do(func() {
		atomic.StoreInt32(&d.close, 1)
		windows.SetEvent(d.readWait)
		d.running.Wait()
		d.session.End()
		if d.wt != nil {
			d.wt.Close()
		}
	})
	return err
}

// Note: Read() and Write() assume the caller comes only from a single thread; there's no locking.

func (d *windowsDevice) Read(buff []byte) (int, error) {
	d.running.Add(1)
	defer d.running.Done()
retry:
	if atomic.LoadInt32(&d.close) == 1 {
		return 0, os.ErrClosed
	}
	start := nanotime()
	shouldSpin := atomic.LoadUint64(&d.rate.current) >= spinloopRateThreshold && uint64(start-atomic.LoadInt64(&d.rate.nextStartTime)) <= rateMeasurementGranularity*2
	for {
		if atomic.LoadInt32(&d.close) == 1 {
			return 0, os.ErrClosed
		}
		packet, err := d.session.ReceivePacket()
		switch err {
		case nil:
			packetSize := len(packet)
			copy(buff, packet)
			d.session.ReleaseReceivePacket(packet)
			d.rate.update(uint64(packetSize))
			return packetSize, nil
		case windows.ERROR_NO_MORE_ITEMS:
			if !shouldSpin || uint64(nanotime()-start) >= spinloopDuration {
				windows.WaitForSingleObject(d.readWait, windows.INFINITE)
				goto retry
			}
			procyield(1)
			continue
		case windows.ERROR_HANDLE_EOF:
			return 0, os.ErrClosed
		case windows.ERROR_INVALID_DATA:
			return 0, errors.New("Send ring corrupt")
		}
		return 0, fmt.Errorf("Read failed: %w", err)
	}
}
func (d *windowsDevice) Flush() error {
	return nil
}

func (d *windowsDevice) Write(buff []byte) (int, error) {
	d.running.Add(1)
	defer d.running.Done()
	if atomic.LoadInt32(&d.close) == 1 {
		return 0, os.ErrClosed
	}

	packetSize := len(buff)
	d.rate.update(uint64(packetSize))

	packet, err := d.session.AllocateSendPacket(packetSize)
	if err == nil {
		copy(packet, buff)
		d.session.SendPacket(packet)
		return packetSize, nil
	}
	switch err {
	case windows.ERROR_HANDLE_EOF:
		return 0, os.ErrClosed
	case windows.ERROR_BUFFER_OVERFLOW:
		return 0, nil // Dropping when ring is full.
	}
	return 0, fmt.Errorf("Write failed: %w", err)
}

// RunningVersion returns the running version of the Wintun driver.
func (d *windowsDevice) RunningVersion() (version uint32, err error) {
	return wintun.RunningVersion()
}

func (rate *rateJuggler) update(packetLen uint64) {
	now := nanotime()
	total := atomic.AddUint64(&rate.nextByteCount, packetLen)
	period := uint64(now - atomic.LoadInt64(&rate.nextStartTime))
	if period >= rateMeasurementGranularity {
		if !atomic.CompareAndSwapInt32(&rate.changing, 0, 1) {
			return
		}
		atomic.StoreInt64(&rate.nextStartTime, now)
		atomic.StoreUint64(&rate.current, total*uint64(time.Second/time.Nanosecond)/period)
		atomic.StoreUint64(&rate.nextByteCount, 0)
		atomic.StoreInt32(&rate.changing, 0)
	}
}
