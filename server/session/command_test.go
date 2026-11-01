package session

import (
	"reflect"
	"testing"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type commandTestSource struct{}

func (commandTestSource) SendCommandOutput(*cmd.Output) {}
func (commandTestSource) Position() mgl64.Vec3          { return mgl64.Vec3{} }

type commandTestTarget string

func (commandTestTarget) Type() string { return "target" }
func (commandTestTarget) Parse(*cmd.Line, reflect.Value) error {
	return nil
}

type commandTestEnum string

func (commandTestEnum) Type() string                { return "action" }
func (commandTestEnum) Options(cmd.Source) []string { return []string{"one", "two"} }

type commandTestRunnable struct {
	Player commandTestTarget `cmd:"player"`
	Action commandTestEnum   `cmd:"action"`
}

func (commandTestRunnable) Run(cmd.Source, *cmd.Output, *world.Tx) {}

// TestBuildAvailableCommandsIsStable verifies that the packet does not change from one build to the next, and that a
// target parameter reaches the client as one.
func TestBuildAvailableCommandsIsStable(t *testing.T) {
	commands := map[string]cmd.Command{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		commands[name] = cmd.New(name, "", []string{name + "x"}, commandTestRunnable{})
	}
	first, _ := BuildAvailableCommands(commands, commandTestSource{}, nil)
	if len(first.Commands) != len(commands) {
		t.Fatalf("expected %v commands, got %v", len(commands), len(first.Commands))
	}
	for range 50 {
		next, _ := BuildAvailableCommands(commands, commandTestSource{}, nil)
		if !reflect.DeepEqual(first, next) {
			t.Fatal("expected every build to be the same")
		}
	}
	param := first.Commands[0].Overloads[0].Parameters[0]
	if param.Type != protocol.CommandArgValid|protocol.CommandArgTypeTarget {
		t.Fatalf("expected a target parameter, got %#x", param.Type)
	}
}
