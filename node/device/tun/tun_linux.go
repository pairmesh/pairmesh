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

package tun

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

const (
	cloneDevicePath = "/dev/net/tun"
	ifReqSize       = unix.IFNAMSIZ + 64
)

// NewTUN creates a new TUN device and set the address to the specified address
func NewTUN(name string) (Device, error) {
	nfd, err := syscall.Open(cloneDevicePath, os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("CreateTUN(%q) failed; %s does not exist", name, cloneDevicePath)
		}
		return nil, err

	}

	errno := setDeviceName(nfd, name)
	if errno != nil {
		return nil, errno
	}

	// Set MTU
	setmtu := func() error {
		// open datagram socket
		fd, err := unix.Socket(
			unix.AF_INET,
			unix.SOCK_DGRAM,
			0,
		)

		if err != nil {
			return err
		}

		defer unix.Close(fd)

		var setmtu [ifReqSize]byte
		copy(setmtu[:], name)
		*(*uint32)(unsafe.Pointer(&setmtu[unix.IFNAMSIZ])) = uint32(DefaultMTU)

		_, _, errno := unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(fd),
			uintptr(unix.SIOCSIFMTU),
			uintptr(unsafe.Pointer(&setmtu[0])),
		)

		if errno != 0 {
			return errors.New("failed to set MTU of TUN device")
		}
		return nil
	}

	err = setmtu()
	if err != nil {
		return nil, err
	}

	err = unix.SetNonblock(nfd, true)
	if err != nil {
		return nil, err
	}

	fname := fmt.Sprintf("pairmeshDeviceFile/%d", nfd)
	dev := &generalDevice{
		name:            name,
		ReadWriteCloser: os.NewFile(uintptr(nfd), fname),
	}

	return dev, nil
}

// setDeviceName set device name with the file descriptor
func setDeviceName(nfd int, name string) error {

	var ifr [ifReqSize]byte
	var flags uint16 = unix.IFF_TUN | unix.IFF_NO_PI //(disabled for TUN status hack)
	nameBytes := []byte(name)
	if len(nameBytes) >= unix.IFNAMSIZ {
		return fmt.Errorf("interface name too long: %w", unix.ENAMETOOLONG)
	}
	copy(ifr[:], nameBytes)
	*(*uint16)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = flags

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(nfd),
		uintptr(unix.TUNSETIFF),
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return errno
	}
	return nil
}
