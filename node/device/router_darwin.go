package device

import "inet.af/netaddr"

func (r *router) add(devName string, localAddress netaddr.IP, target netaddr.IPPrefix) error {
	return bsdAdd(devName, localAddress, target)
}

func (r *router) del(devName string, localAddress netaddr.IP, target netaddr.IPPrefix) error {
	return bsdDel(devName, localAddress, target)
}
