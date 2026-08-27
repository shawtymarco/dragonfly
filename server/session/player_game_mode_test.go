package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestPocketMineSpectatorPresentation(t *testing.T) {
	if got := gameTypeFromMode(world.GameModeSpectator); got != packet.GameTypeCreative {
		t.Fatalf("spectator game type = %d, want creative", got)
	}
	abilities := gameModeAbilities(world.GameModeSpectator)
	if abilities&protocol.AbilityInstantBuild == 0 || abilities&protocol.AbilityInvulnerable == 0 {
		t.Fatalf("spectator abilities = %#x, missing PMMP creative/invulnerable bits", abilities)
	}
	if abilities&protocol.AbilityMayFly != 0 {
		t.Fatalf("spectator abilities = %#x, PMMP locks flying instead of allowing a toggle", abilities)
	}
	forbidden := uint32(protocol.AbilityDoorsAndSwitches | protocol.AbilityOpenContainers | protocol.AbilityAttackPlayers | protocol.AbilityAttackMobs)
	if abilities&forbidden != 0 {
		t.Fatalf("spectator abilities = %#x, interaction bits must remain disabled", abilities)
	}
}

func TestNativeSpectatorPresentationUnchanged(t *testing.T) {
	if got := gameTypeFromMode(world.GameModeNativeSpectator); got != packet.GameTypeSpectator {
		t.Fatalf("native spectator game type = %d, want native spectator", got)
	}
	abilities := gameModeAbilities(world.GameModeNativeSpectator)
	if abilities&protocol.AbilityInstantBuild != 0 {
		t.Fatalf("native spectator abilities = %#x, instant build unexpectedly enabled", abilities)
	}
}
