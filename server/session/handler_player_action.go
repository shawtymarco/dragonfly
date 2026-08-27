package session

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// PlayerActionHandler handles the PlayerAction packet.
type PlayerActionHandler struct{}

// Handle ...
func (*PlayerActionHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, c Controllable) error {
	pk := p.(*packet.PlayerAction)

	return handlePlayerAction(pk.ActionType, pk.BlockFace, pk.BlockPosition, pk.EntityRuntimeID, s, c)
}

// handlePlayerAction handles an action performed by a player, found in packet.PlayerAction and packet.PlayerAuthInput.
func handlePlayerAction(action int32, face int32, pos protocol.BlockPos, entityRuntimeID uint64, s *Session, c Controllable) error {
	if entityRuntimeID != selfEntityRuntimeID {
		return errSelfRuntimeID
	}
	switch action {
	case protocol.PlayerActionStartSleeping, protocol.PlayerActionRespawn:
		// Don't do anything for these actions.
	case protocol.PlayerActionDimensionChangeDone:
		// Completing a dimension transition resets the client's local flight
		// state. This is observable after proxy fast transfers in faux/native
		// spectator: collision is already disabled, but the client falls until
		// it toggles flight itself. Reapply the authoritative mode at the client
		// acknowledgement boundary so forced flight and noclip are restored
		// after the reset rather than racing it with a timer.
		resyncForcedFlightAfterDimension(c)
	case protocol.PlayerActionStopSleeping:
		c.Wake()
	case protocol.PlayerActionStartBreak, protocol.PlayerActionContinueDestroyBlock:
		s.swingingArm.Store(true)
		defer s.swingingArm.Store(false)

		s.breakingPos = cube.Pos{int(pos[0]), int(pos[1]), int(pos[2])}
		c.StartBreaking(s.breakingPos, cube.Face(face))
	case protocol.PlayerActionAbortBreak:
		c.AbortBreaking()
	case protocol.PlayerActionPredictDestroyBlock, protocol.PlayerActionStopBreak:
		s.swingingArm.Store(true)
		defer s.swingingArm.Store(false)
		c.FinishBreaking()
	case protocol.PlayerActionCrackBreak:
		// Don't do anything for this action. It is no longer used. Block
		// cracking is done fully server-side.
	case protocol.PlayerActionStartItemUseOn:
		// TODO: Properly utilize these actions.
	case protocol.PlayerActionStopItemUseOn:
		c.ReleaseItem()
	case protocol.PlayerActionStartBuildingBlock:
		// Don't do anything for this action.
	case protocol.PlayerActionCreativePlayerDestroyBlock:
		// Don't do anything for this action.
	case protocol.PlayerActionMissedSwing:
		s.swingingArm.Store(true)
		defer s.swingingArm.Store(false)
		c.PunchAir()
	default:
		return fmt.Errorf("unhandled ActionType %v", action)
	}
	return nil
}

// resyncForcedFlightAfterDimension republishes modes that require flight and
// noclip to remain continuously enabled. Ordinary creative flight is excluded
// because the player owns that toggle.
func resyncForcedFlightAfterDimension(c Controllable) {
	mode := c.GameMode()
	if requiresPostDimensionFlightResync(mode) {
		c.SetGameMode(mode)
	}
}

func requiresPostDimensionFlightResync(mode world.GameMode) bool {
	return mode.AllowsFlying() && !mode.HasCollision()
}
