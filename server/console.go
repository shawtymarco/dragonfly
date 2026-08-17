package server

import (
	"bufio"
	"log/slog"
	"os"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Console is a cmd.Source that runs commands typed into the process stdin.
type Console struct {
	log *slog.Logger
}

func (Console) Name() string         { return "CONSOLE" }
func (Console) Position() mgl64.Vec3 { return mgl64.Vec3{} }
func (Console) Operator() bool       { return true }

func (c Console) SendCommandOutput(o *cmd.Output) {
	for _, m := range o.Messages() {
		c.log.Info(m.String())
	}
	for _, err := range o.Errors() {
		c.log.Error(err.Error())
	}
}

func (srv *Server) listenConsole() {
	sc := bufio.NewScanner(os.Stdin)
	src := Console{log: srv.conf.Log}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		name, args, _ := strings.Cut(line, " ")
		name = strings.TrimPrefix(strings.ToLower(name), "/")
		command, ok := cmd.ByAlias(name)
		if !ok {
			o := &cmd.Output{}
			o.Errort(cmd.MessageUnknown, name)
			src.SendCommandOutput(o)
			continue
		}
		command.Execute(args, src, (*world.Tx)(nil))
	}
}
