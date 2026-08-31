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

// handleFlightToggle applies a client-requested flight state. Collisionless
// modes force flight, so rejecting a stop request silently is insufficient: the
// client has already changed its local state and must receive authoritative
// abilities again or it keeps Creative collision until another manual toggle.
func handleFlightToggle(s *Session, c Controllable, flying bool) {
	switch flightTogglePolicy(c.GameMode(), c.Flying(), flying) {
	case flightToggleResync:
		s.conf.Log.Debug("process flight toggle: resynchronising rejected request", "flying", flying)
		s.SendAbilities(c)
	case flightToggleStart:
		c.StartFlying()
	case flightToggleStop:
		c.StopFlying()
	}
}

type flightToggleDecision uint8

const (
	flightToggleNoop flightToggleDecision = iota
	flightToggleStart
	flightToggleStop
	flightToggleResync
)

func flightTogglePolicy(mode world.GameMode, current, requested bool) flightToggleDecision {
	if !mode.AllowsFlying() {
		return flightToggleResync
	}
	if !mode.HasCollision() {
		if !requested || !current {
			return flightToggleResync
		}
		return flightToggleNoop
	}
	if requested == current {
		return flightToggleNoop
	}
	if requested {
		return flightToggleStart
	}
	return flightToggleStop
}
