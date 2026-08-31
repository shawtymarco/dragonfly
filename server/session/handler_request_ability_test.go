package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

func TestCollisionlessStopFlyingRequiresAuthoritativeResync(t *testing.T) {
	for _, mode := range []world.GameMode{world.GameModeSpectator, world.GameModeNativeSpectator} {
		if got := flightTogglePolicy(mode, true, false); got != flightToggleResync {
			t.Fatalf("%T stop-flying policy = %d, want resync", mode, got)
		}
	}
}

func TestCreativeFlightToggleRemainsClientControlled(t *testing.T) {
	if got := flightTogglePolicy(world.GameModeCreative, false, true); got != flightToggleStart {
		t.Fatalf("creative start-flying policy = %d, want start", got)
	}
	if got := flightTogglePolicy(world.GameModeCreative, true, false); got != flightToggleStop {
		t.Fatalf("creative stop-flying policy = %d, want stop", got)
	}
}
