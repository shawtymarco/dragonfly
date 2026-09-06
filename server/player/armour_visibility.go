package player

import "github.com/df-mc/dragonfly/server/world"

// ArmourVisibilityHandler optionally controls the armour shown to other players.
// It is evaluated in the subject's world transaction on every armour update,
// including initial spawn and re-entry into a viewer's loaded chunks. It must
// only read state and must not retain either player beyond the call.
type ArmourVisibilityHandler interface {
	HandleArmourVisibility(subject, viewer *Player) bool
}

// ArmourVisible reports whether viewer should receive the player's real armour.
// A false result sends empty equipment without changing the equipped inventory.
// The player's own inventory and handlers without this policy are unaffected.
func (p *Player) ArmourVisible(viewer *world.EntityHandle) bool {
	h, ok := p.h.(ArmourVisibilityHandler)
	if !ok || viewer == p.H() {
		return true
	}
	if viewer == nil {
		return false
	}
	e, ok := viewer.Entity(p.tx)
	if !ok {
		return false
	}
	v, ok := e.(*Player)
	return ok && h.HandleArmourVisibility(p, v)
}

// refreshArmourVisibility publishes effect/handler transitions to current
// viewers. Future viewers evaluate ArmourVisible when their actor is spawned.
func (p *Player) refreshArmourVisibility() {
	for _, viewer := range p.viewers() {
		viewer.ViewEntityArmour(p)
	}
}
