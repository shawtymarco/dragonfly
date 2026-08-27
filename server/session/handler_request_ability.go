package session

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// RequestAbilityHandler handles the RequestAbility packet.
type RequestAbilityHandler struct{}

// Handle ...
func (a RequestAbilityHandler) Handle(p packet.Packet, s *Session, _ *world.Tx, c Controllable) error {
	pk := p.(*packet.RequestAbility)
	if pk.Ability == packet.AbilityFlying {
		flying, ok := pk.Value.(bool)
		if !ok {
			return fmt.Errorf("RequestAbility: flying value has type %T, expected bool", pk.Value)
		}
		handleFlightToggle(s, c, flying)
	}
	return nil
}

// handleFlightToggle applies a client-requested flight state without allowing
// the client to disable a server-forced spectator flight state.
func handleFlightToggle(s *Session, c Controllable, flying bool) {
	if flying == c.Flying() {
		return
	}
	if !world.AllowsFlightToggle(c.GameMode()) {
		s.SendAbilities(c)
		return
	}
	if flying {
		c.StartFlying()
	} else {
		c.StopFlying()
	}
}
