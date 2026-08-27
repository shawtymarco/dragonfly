package world

import "testing"

func TestBuiltInFlightTogglePolicy(t *testing.T) {
	tests := []struct {
		name string
		mode GameMode
		want bool
	}{
		{name: "survival", mode: GameModeSurvival},
		{name: "creative", mode: GameModeCreative, want: true},
		{name: "adventure", mode: GameModeAdventure},
		{name: "spectator", mode: GameModeSpectator},
		{name: "native spectator", mode: GameModeNativeSpectator},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AllowsFlightToggle(test.mode); got != test.want {
				t.Fatalf("AllowsFlightToggle() = %t, want %t", got, test.want)
			}
		})
	}
}
