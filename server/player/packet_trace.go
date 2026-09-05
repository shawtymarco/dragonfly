package player

import "github.com/df-mc/dragonfly/server/session"

// DeferPacketTrace transfers the incoming action's trace to a scheduled action.
// Callers must run or cancel the returned trace on every termination path.
func (p *Player) DeferPacketTrace() *session.DeferredPacketTrace {
	return p.session().DeferPacketTrace()
}
