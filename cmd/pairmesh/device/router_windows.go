package device

import (
	"github.com/pairmesh/pairmesh/cmd/pairmesh/device/runner"

	"inet.af/netaddr"
)

func (r *router) add(_ string, localAddress netaddr.IP, target netaddr.IPPrefix) error {
	args := []string{
		"route",
		"add",
		target.IP().String(),
		"mask",
		target.Masked().String(),
		localAddress.String(),
	}
	return runner.Run(args)
}

func (r *router) del(_ string, _ netaddr.IP, target netaddr.IPPrefix) error {
	args := []string{
		"route",
		"delete ",
		target.IP().String(),
		"mask",
		target.Masked().String(),
	}
	return runner.Run(args)
}
