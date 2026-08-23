package server

import (
	"errors"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

var errResourceReloadUnsupported = errors.New("no listener supports resource pack reloading")

type resourceReloadingListener interface {
	reloadResources([]*resource.Pack)
}

// ReloadResources replaces the resource packs offered to new connections by
// every listener that supports runtime resource pack replacement. Existing
// connections are not affected and must reconnect to negotiate the new packs.
func (srv *Server) ReloadResources(resources []*resource.Pack) error {
	reloaded := false
	for _, l := range srv.listeners {
		if resourceListener, ok := l.(resourceReloadingListener); ok {
			resourceListener.reloadResources(resources)
			reloaded = true
		}
	}
	if !reloaded {
		return errResourceReloadUnsupported
	}
	return nil
}
