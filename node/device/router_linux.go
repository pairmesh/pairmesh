package device

import (
	"github.com/pairmesh/pairmesh/node/device/runner"

	"inet.af/netaddr"
)

func (r *router) add(devName string, localAddress netaddr.IP, target netaddr.IPPrefix) error {
	args := []string{
		"ip",
		"route",
		"add",
		target.Masked().String(),
		"via",
		localAddress.String(),
	}
	return runner.Run(args)
}

func (r *router) del(devName string, localAddress netaddr.IP, target netaddr.IPPrefix) error {
	args := []string{
		"ip",
		"route",
		"del",
		target.Masked().String(),
		"via",
		localAddress.String(),
	}
	return runner.Run(args)
}
