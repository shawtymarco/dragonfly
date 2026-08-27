package player

import (
	"testing"

	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
)

func TestCollisionlessGameModesNeverReportOnGround(t *testing.T) {
	tests := []struct {
		name string
		mode world.GameMode
		want bool
	}{
		{name: "survival", mode: world.GameModeSurvival, want: true},
		{name: "creative", mode: world.GameModeCreative, want: true},
		{name: "adventure", mode: world.GameModeAdventure, want: true},
		{name: "faux spectator", mode: world.GameModeSpectator},
		{name: "native spectator", mode: world.GameModeNativeSpectator},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Player{playerData: &playerData{s: &session.Session{}, gameMode: test.mode, onGround: true}}
			if got := p.OnGround(); got != test.want {
				t.Fatalf("OnGround() = %t, want %t", got, test.want)
			}
		})
	}
}
