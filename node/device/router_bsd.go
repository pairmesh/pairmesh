//go:build darwin || freebsd
// +build darwin freebsd

package device

import (
	"fmt"
	"runtime"

	"github.com/pairmesh/pairmesh/node/device/runner"

	"inet.af/netaddr"
)

func inet(p netaddr.IPPrefix) string {
	if p.IP().Is6() {
		return "inet6"
	}
	return "inet"
}

func bsdAdd(devName string, _ netaddr.IP, target netaddr.IPPrefix) error {
	net := target.IPNet()
	nip := net.IP.Mask(net.Mask)
	nstr := fmt.Sprintf("%v/%d", nip, target.Bits())
	args := []string{
		"route",
		"-q",
		"-n",
		"add",
		"-" + inet(target),
		nstr,
		"-iface", devName,
	}
	return runner.Run(args)
}

func bsdDel(devName string, _ netaddr.IP, target netaddr.IPPrefix) error {
	net := target.IPNet()
	nip := net.IP.Mask(net.Mask)
	nstr := fmt.Sprintf("%v/%d", nip, target.Bits)
	del := "del"
	if runtime.GOOS == "darwin" {
		del = "delete"
	}

	args := []string{
		"route",
		"-q",
		"-n",
		del,
		"-" + inet(target),
		nstr,
		"-iface",
		devName,
	}

	return runner.Run(args)
}
