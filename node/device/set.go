package device

import "inet.af/netaddr"

type Set map[netaddr.IPPrefix]struct{}

func NewSet(ips ...netaddr.IPPrefix) Set {
	s := Set{}
	for _, ip := range ips {
		s[ip] = struct{}{}
	}
	return s
}

func (s Set) Add(ip netaddr.IPPrefix) {
	s[ip] = struct{}{}
}

func (s Set) Has(ip netaddr.IPPrefix) bool {
	_, found := s[ip]
	return found
}
