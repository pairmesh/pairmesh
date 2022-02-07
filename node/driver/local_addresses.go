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

package driver

import (
	"net"
	"runtime"
	"sort"
	"strings"

	"go.uber.org/zap"
	"inet.af/netaddr"
)

func isUp(iface *net.Interface) bool       { return iface.Flags&net.FlagUp != 0 }
func isLoopback(iface *net.Interface) bool { return iface.Flags&net.FlagLoopback != 0 }
func isProblematicInterface(iface *net.Interface) bool {
	name := iface.Name

	// Don't try to send disco/etc packets over zerotier; they effectively
	// DoS each other by doing traffic amplification, both of them
	// preferring/trying to use each other for transport. See:
	// https://github.com/tailscale/tailscale/issues/1208
	if strings.HasPrefix(name, "zt") || (runtime.GOOS == "windows" && strings.Contains(name, "ZeroTier")) {
		return true
	}

	if strings.HasPrefix(name, "ts") || (runtime.GOOS == "windows" && strings.Contains(name, "TailScale")) {
		return true
	}

	return false
}

// localAddresses detects the local interfaces and returns all the IP addresses
// belongs to current node.
func (d *NodeDriver) localAddresses() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		zap.L().Error("Retrieve the interfaces failed", zap.Error(err))
		return nil
	}

	var localAddresses []netaddr.IP

	for i := range ifaces {
		iface := &ifaces[i]
		if !isUp(iface) || isProblematicInterface(iface) {
			// Skip down interfaces and ones that are
			// problematic that we don't want to try to
			// send Tailscale traffic over.
			continue
		}

		ifcIsLoopback := isLoopback(iface)
		if ifcIsLoopback {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			zap.L().Error("Retrieve interface addresses failed",
				zap.String("interface", iface.Name), zap.Error(err))
			continue
		}

		for _, a := range addrs {
			v, ok := a.(*net.IPNet)
			if !ok {
				continue
			}

			ip, ok := netaddr.FromStdIP(v.IP)
			if !ok {
				zap.L().Info("====>", zap.Reflect("ip", v.IP))
				continue
			}

			if ip.Is6() {
				continue
			}

			if ip.String() == d.credential.address {
				continue
			}

			localAddresses = append(localAddresses, ip)
		}
	}

	sort.Slice(localAddresses, func(i, j int) bool {
		return localAddresses[i].Less(localAddresses[j])
	})

	// Deduplicate
	var addresses []string
	for _, a := range localAddresses {
		addr := a.String()
		if len(addresses) == 0 || (len(addresses) > 0 && addresses[len(addresses)-1] != addr) {
			addresses = append(addresses, addr)
		}
	}

	return addresses
}
