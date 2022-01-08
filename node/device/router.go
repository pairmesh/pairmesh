package device

import (
	"go.uber.org/zap"

	"go.uber.org/atomic"

	"inet.af/netaddr"
)

type (
	Router interface {
		Set(cfg *Config)
		Add(cfg *Config)
	}

	router struct {
		dev Device

		// Cached previous routes result.
		routes atomic.Value // []netaddr.IPPrefix
	}

	// Config represents the router configurations
	Config struct {
		LocalAddress netaddr.IP         `json:"local_address"`
		Routes       []netaddr.IPPrefix `json:"routes"`
	}
)

// newRouter return a router instance.
func newRouter(dev Device) Router {
	return &router{dev: dev}
}

func (r *router) Set(cfg *Config) {
	var old Set
	if o := r.routes.Load(); o != nil {
		old = NewSet(o.([]netaddr.IPPrefix)...)
	}
	cur := NewSet(cfg.Routes...)
	del := NewSet()
	for ip := range old {
		if !cur.Has(ip) {
			del.Add(ip)
		}
	}

	add := NewSet()
	for ip := range cur {
		if !old.Has(ip) {
			add.Add(ip)
		}
	}

	devName := r.dev.Name()
	for ip := range del {
		err := r.del(devName, cfg.LocalAddress, ip)
		if err != nil {
			zap.L().Error("Delete route failed", zap.Stringer("target", ip), zap.Error(err))
		}
	}

	for ip := range add {
		err := r.add(devName, cfg.LocalAddress, ip)
		if err != nil {
			zap.L().Error("Add route failed", zap.Stringer("target", ip), zap.Error(err))
		}
	}

	// Update the cache.
	r.routes.Store(cfg.Routes)
}

func (r *router) Add(cfg *Config) {
	var old Set
	if o := r.routes.Load(); o != nil {
		old = NewSet(o.([]netaddr.IPPrefix)...)
	}
	cur := NewSet(cfg.Routes...)

	add := NewSet()
	for ip := range cur {
		if !old.Has(ip) {
			add.Add(ip)
		}
	}

	devName := r.dev.Name()
	var addArray []netaddr.IPPrefix
	for ip := range add {
		addArray = append(addArray, ip)
		err := r.add(devName, cfg.LocalAddress, ip)
		if err != nil {
			zap.L().Error("Add route failed", zap.Stringer("target", ip), zap.Error(err))
		}
	}

	// Update the cache.
	if o := r.routes.Load(); o != nil {
		r.routes.Store(append(o.([]netaddr.IPPrefix), addArray...))
	} else {
		r.routes.Store(cfg.Routes)
	}
}
