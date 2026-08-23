package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync/atomic"

	"github.com/df-mc/dragonfly/server/session"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// Listener is a source for connections that may be listened on by a Server using Server.listen. Proxies can use this to
// provide players from a different source.
type Listener interface {
	// Accept blocks until the next connection is established and returns it. An error is returned if the Listener was
	// closed using Close.
	Accept() (session.Conn, error)
	// Disconnect disconnects a connection from the Listener with a reason.
	Disconnect(conn session.Conn, reason string) error
	io.Closer
}

// listenerFunc may be used to return a *minecraft.Listener using a Config. It
// is the standard listener used when UserConfig.Config() is called.
func (uc UserConfig) listenerFunc(conf Config) (Listener, error) {
	l := &listener{}
	l.reloadResources(conf.Resources)
	cfg := minecraft.ListenConfig{
		MaximumPlayers:         conf.MaxPlayers,
		StatusProvider:         conf.StatusProvider,
		AuthenticationDisabled: conf.AuthDisabled,
		AcceptedProtocols:      conf.AcceptedProtocols,
		ResourcePacks:          conf.Resources,
		TexturePacksRequired:   conf.ResourcesRequired,
		Compression:            conf.Compression,
		Allow:                  conf.Allower.Allow,
	}
	cfg.FetchResourcePacks = func(login.IdentityData, login.ClientData, []*resource.Pack) []*resource.Pack {
		return l.resourcePacks()
	}
	if conf.Log.Enabled(context.Background(), slog.LevelDebug) {
		cfg.ErrorLog = conf.Log.With("net origin", "gophertunnel")
	}
	networkListener, err := cfg.Listen("raknet", uc.Network.Address)
	if err != nil {
		return nil, fmt.Errorf("create minecraft listener: %w", err)
	}
	l.Listener = networkListener
	conf.Log.Info("Listener running.", "addr", networkListener.Addr())
	return l, nil
}

// listener is a Listener implementation that wraps around a minecraft.Listener so that it can be listened on by
// Server.
type listener struct {
	*minecraft.Listener
	resources atomic.Pointer[[]*resource.Pack]
}

func (l *listener) reloadResources(packs []*resource.Pack) {
	cloned := slices.Clone(packs)
	l.resources.Store(&cloned)
}

func (l *listener) resourcePacks() []*resource.Pack {
	packs := l.resources.Load()
	if packs == nil {
		return nil
	}
	return slices.Clone(*packs)
}

// Accept blocks until the next connection is established and returns it. An error is returned if the Listener was
// closed using Close.
func (l *listener) Accept() (session.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return conn.(session.Conn), err
}

// Disconnect disconnects a connection from the Listener with a reason.
func (l *listener) Disconnect(conn session.Conn, reason string) error {
	return l.Listener.Disconnect(conn.(*minecraft.Conn), reason)
}
