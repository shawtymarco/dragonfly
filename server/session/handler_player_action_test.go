package session

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type legacyPlayerActionControllable struct {
	Controllable
	action string
}

func (c *legacyPlayerActionControllable) Jump()           { c.action = "jump" }
func (c *legacyPlayerActionControllable) StartSprinting() { c.action = "start_sprint" }
func (c *legacyPlayerActionControllable) StopSprinting()  { c.action = "stop_sprint" }
func (c *legacyPlayerActionControllable) StartSneaking()  { c.action = "start_sneak" }
func (c *legacyPlayerActionControllable) StopSneaking()   { c.action = "stop_sneak" }
func (c *legacyPlayerActionControllable) StartSwimming()  { c.action = "start_swim" }
func (c *legacyPlayerActionControllable) StopSwimming()   { c.action = "stop_swim" }
func (c *legacyPlayerActionControllable) StartGliding()   { c.action = "start_glide" }
func (c *legacyPlayerActionControllable) StopGliding()    { c.action = "stop_glide" }

func TestHandleLegacyPlayerActions(t *testing.T) {
	tests := []struct {
		action int32
		want   string
	}{
		{action: protocol.PlayerActionJump, want: "jump"},
		{action: protocol.PlayerActionStartSprint, want: "start_sprint"},
		{action: protocol.PlayerActionStopSprint, want: "stop_sprint"},
		{action: protocol.PlayerActionStartSneak, want: "start_sneak"},
		{action: protocol.PlayerActionStopSneak, want: "stop_sneak"},
		{action: protocol.PlayerActionStartSwimming, want: "start_swim"},
		{action: protocol.PlayerActionStopSwimming, want: "stop_swim"},
		{action: protocol.PlayerActionStartGlide, want: "start_glide"},
		{action: protocol.PlayerActionStopGlide, want: "stop_glide"},
	}
	for _, test := range tests {
		controllable := &legacyPlayerActionControllable{}
		if err := handlePlayerAction(test.action, 0, protocol.BlockPos{}, selfEntityRuntimeID, &Session{}, controllable); err != nil {
			t.Fatalf("action %d returned error: %v", test.action, err)
		}
		if controllable.action != test.want {
			t.Fatalf("action %d called %q, want %q", test.action, controllable.action, test.want)
		}
	}
}
