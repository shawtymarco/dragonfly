package server

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/text"
)

// operators is a PMMP-style ops.txt list of operator names.
type operators struct {
	path string
	mu   sync.RWMutex
	names map[string]struct{}
}

func loadOperators(path string) *operators {
	o := &operators{path: path, names: make(map[string]struct{})}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.WriteFile(path, []byte{}, 0644)
		}
		return o
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name := strings.TrimSpace(sc.Text())
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		o.names[strings.ToLower(name)] = struct{}{}
	}
	return o
}

func (o *operators) has(name string) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	_, ok := o.names[strings.ToLower(name)]
	return ok
}

func (o *operators) set(name string, op bool) error {
	key := strings.ToLower(name)
	o.mu.Lock()
	if op {
		o.names[key] = struct{}{}
	} else {
		delete(o.names, key)
	}
	names := make([]string, 0, len(o.names))
	for n := range o.names {
		names = append(names, n)
	}
	path := o.path
	o.mu.Unlock()

	var b strings.Builder
	for _, n := range names {
		b.WriteString(n)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

// IsOp reports whether name is in ops.txt, matching PMMP Server::isOp (case-insensitive).
func (srv *Server) IsOp(name string) bool {
	if srv.ops == nil {
		return false
	}
	return srv.ops.has(name)
}

// AddOp adds name to ops.txt and, if they are online, grants operator abilities.
func (srv *Server) AddOp(name string) error {
	if err := srv.ops.set(name, true); err != nil {
		return err
	}
	srv.applyOperator(name, true)
	return nil
}

// RemoveOp removes name from ops.txt and, if they are online, revokes operator abilities.
func (srv *Server) RemoveOp(name string) error {
	if err := srv.ops.set(name, false); err != nil {
		return err
	}
	srv.applyOperator(name, false)
	return nil
}

func (srv *Server) applyOperator(name string, op bool) {
	if h, ok := srv.playerHandleFold(name); ok {
		player.Do(h, func(_ *world.Tx, p *player.Player) {
			p.SetOperator(op)
		})
	}
}

func (srv *Server) playerHandleFold(name string) (*world.EntityHandle, bool) {
	srv.pmu.RLock()
	defer srv.pmu.RUnlock()
	for _, p := range srv.p {
		if strings.EqualFold(p.name, name) {
			return p.handle, true
		}
	}
	return nil, false
}

// BroadcastAdmin sends a grey italic admin copy of msg to every operator except
// src, matching PMMP Command::broadcastCommandMessage(..., false). Used so
// operators can see private tells.
func (srv *Server) BroadcastAdmin(src cmd.Source, tx *world.Tx, msg string) {
	srcName := "unknown"
	if n, ok := src.(cmd.NamedTarget); ok {
		srcName = n.Name()
	}
	line := text.Colourf("<grey><italic>[%s: %s]</italic></grey>", srcName, msg)
	var srcUUID string
	if p, ok := src.(*player.Player); ok {
		srcUUID = p.UUID().String()
	}
	send := func(p *player.Player) {
		if !p.Operator() {
			return
		}
		if srcUUID != "" && p.UUID().String() == srcUUID {
			return
		}
		p.Message(line)
	}
	if tx != nil {
		for p := range srv.Players(tx) {
			send(p)
		}
	} else {
		for p := range srv.Players(nil) {
			send(p)
		}
	}
	srv.conf.Log.Info("admin", "src", srcName, "msg", msg)
}
