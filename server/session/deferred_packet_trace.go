package session

import (
	"sync"
	"time"
)

// DeferredPacketTrace owns the single terminal result of an action deliberately
// postponed by gameplay. Run must execute on the original session's world owner;
// Cancel may also be used when its scheduled task or entity is closed.
// A DeferredPacketTrace must not be copied. Every deferred trace must be run or
// cancelled by the caller, including failed scheduling and disconnect paths.
type DeferredPacketTrace struct {
	once  sync.Once
	s     *Session
	state packetTraceState
}

// DeferPacketTrace transfers ownership of the current result to the caller.
// The original handler may then return/cancel without producing a second result.
// It returns nil when no unfinished incoming trace exists.
func (s *Session) DeferPacketTrace() *DeferredPacketTrace {
	s.packetTrace.mu.Lock()
	defer s.packetTrace.mu.Unlock()
	if !s.packetTrace.active || s.packetTrace.current.terminal {
		return nil
	}
	d := &DeferredPacketTrace{s: s, state: s.packetTrace.current}
	s.packetTrace.current.terminal = true
	return d
}

// Run resumes tracing around f exactly once. Deliberate deferral is included in
// the edge's end-to-end time, but not reported as world-owner queue contention.
// With a nil receiver, f is simply run without tracing.
func (d *DeferredPacketTrace) Run(f func()) {
	if d == nil {
		f()
		return
	}
	d.once.Do(func() {
		s := d.s
		s.packetTrace.mu.Lock()
		previous, active := s.packetTrace.current, s.packetTrace.active
		state := d.state
		state.startedAt = time.Now()
		state.deferred = true
		s.packetTrace.current, s.packetTrace.active = state, true
		s.packetTrace.mu.Unlock()
		defer func() {
			s.endPacketTrace()
			s.packetTrace.mu.Lock()
			s.packetTrace.current, s.packetTrace.active = previous, active
			s.packetTrace.mu.Unlock()
		}()
		f()
	})
}

// Cancel publishes the original action's rejected terminal result exactly once.
// It never installs the old action as the session's current packet.
func (d *DeferredPacketTrace) Cancel(reason string) {
	if d == nil {
		return
	}
	d.once.Do(func() {
		state := d.state
		state.startedAt = time.Now()
		d.s.queuePacketTraceResult(packetTraceResult(state, false, true, PacketTraceRoleAttacker, reason))
	})
}
