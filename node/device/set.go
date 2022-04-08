package device

import "inet.af/netaddr"

// Set is the set of netaddr.IPPrefix with a map under the hood
type Set map[netaddr.IPPrefix]struct{}

// NewSet returns new Set struct with given ips
func NewSet(ips ...netaddr.IPPrefix) Set {
	s := Set{}
	for _, ip := range ips {
		s[ip] = struct{}{}
	}
	return s
}

// Add adds the input ip into the Set
func (s Set) Add(ip netaddr.IPPrefix) {
	s[ip] = struct{}{}
}

// Has returns whether the Set has given ip
func (s Set) Has(ip netaddr.IPPrefix) bool {
	_, found := s[ip]
	return found
}
