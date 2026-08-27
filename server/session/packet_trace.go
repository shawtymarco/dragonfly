package session

import (
	"sync"
	"time"
)

const (
	// PacketTraceRoleAttacker identifies feedback delivered to the player that
	// originated an action.
	PacketTraceRoleAttacker uint8 = iota + 1
	// PacketTraceRoleVictim identifies feedback delivered to the player affected
	// by an action.
	PacketTraceRoleVictim
)

const (
	PacketTraceReasonAccepted      = "accepted"
	PacketTraceReasonCooldown      = "cooldown"
	PacketTraceReasonInvulnerable  = "dead_or_invulnerable"
	PacketTraceReasonHandler       = "handler_cancelled"
	PacketTraceReasonUnreachable   = "unreachable_current_state"
	PacketTraceReasonNonPlayer     = "non_player_target"
	PacketTraceReasonInternal      = "internal_error"
	PacketTraceReasonSelfTarget    = "self_target"
	PacketTraceReasonTargetMissing = "target_missing"
	PacketTraceReasonTargetWorld   = "target_world_mismatch"
	PacketTraceReasonHeldItem      = "held_item_mismatch"
)

// PacketTrace carries transport-owned metadata for one incoming packet. The
// timestamp is created by the connection implementation in this process so
// durations may use Go's monotonic clock without synchronising hosts.
type PacketTrace struct {
	ID         uint64
	ReceivedAt time.Time
}

// PacketTraceResult is queued on the normal ordered session writer after all
// feedback relevant to the result has been queued. Connection implementations
// may carry it over an internal transport; it is never a Bedrock packet.
type PacketTraceResult struct {
	ID               uint64
	Accepted         bool
	Terminal         bool
	FeedbackComplete bool
	Role             uint8
	Reason           string
	QueueDuration    time.Duration
	HandlerDuration  time.Duration
}

// PacketTraceConn is an optional extension implemented by internal transports
// that attach metadata to incoming packets and consume ordered result markers.
// Normal Minecraft connections do not need to implement it.
type PacketTraceConn interface {
	ConsumePacketTrace() (PacketTrace, bool)
	WritePacketTraceResult(PacketTraceResult) error
}

type packetTraceState struct {
	trace         PacketTrace
	queueDuration time.Duration
	startedAt     time.Time
	terminal      bool
	acceptedHint  bool
}

type packetTraceTracker struct {
	mu      sync.Mutex
	current packetTraceState
	active  bool
}

func (s *Session) beginPacketTrace(trace PacketTrace, startedAt time.Time) {
	s.packetTrace.mu.Lock()
	s.packetTrace.current = packetTraceState{
		trace:         trace,
		queueDuration: startedAt.Sub(trace.ReceivedAt),
		startedAt:     startedAt,
	}
	s.packetTrace.active = true
	s.packetTrace.mu.Unlock()
}

func (s *Session) endPacketTrace() {
	s.packetTrace.mu.Lock()
	if !s.packetTrace.active {
		s.packetTrace.mu.Unlock()
		return
	}
	state := s.packetTrace.current
	s.packetTrace.active = false
	s.packetTrace.current = packetTraceState{}
	s.packetTrace.mu.Unlock()
	if !state.terminal {
		s.queuePacketTraceResult(packetTraceResult(state, false, true, PacketTraceRoleAttacker, PacketTraceReasonInternal))
	}
}

// CurrentPacketTrace returns the trace attached to the packet currently being
// handled on the session's world owner.
func (s *Session) CurrentPacketTrace() (PacketTrace, bool) {
	s.packetTrace.mu.Lock()
	defer s.packetTrace.mu.Unlock()
	if !s.packetTrace.active {
		return PacketTrace{}, false
	}
	return s.packetTrace.current.trace, true
}

// RejectPacketTrace records a terminal rejected result for the current packet.
// Repeated calls for the same trace are ignored.
func (s *Session) RejectPacketTrace(id uint64, reason string) {
	s.finishPacketTrace(id, false, reason)
}

// MarkPacketTraceAccepted records that gameplay intentionally handled a hit
// while cancelling Dragonfly's default damage path. FinishPacketTraceAccepted
// still publishes the ordered terminal result after remaining feedback.
func (s *Session) MarkPacketTraceAccepted(id uint64) {
	s.packetTrace.mu.Lock()
	defer s.packetTrace.mu.Unlock()
	if s.packetTrace.active && s.packetTrace.current.trace.ID == id && !s.packetTrace.current.terminal {
		s.packetTrace.current.acceptedHint = true
	}
}

// PacketTraceAccepted reports whether a cancelled default damage path was
// explicitly accepted by gameplay code.
func (s *Session) PacketTraceAccepted(id uint64) bool {
	s.packetTrace.mu.Lock()
	defer s.packetTrace.mu.Unlock()
	return s.packetTrace.active && s.packetTrace.current.trace.ID == id && s.packetTrace.current.acceptedHint
}

// PacketTraceFinished reports whether the current trace already has a terminal
// result.
func (s *Session) PacketTraceFinished(id uint64) bool {
	s.packetTrace.mu.Lock()
	defer s.packetTrace.mu.Unlock()
	return s.packetTrace.active && s.packetTrace.current.trace.ID == id && s.packetTrace.current.terminal
}

// FinishPacketTraceAccepted queues the attacker feedback barrier and terminal
// accepted result on the normal writer.
func (s *Session) FinishPacketTraceAccepted(id uint64) {
	s.finishPacketTrace(id, true, PacketTraceReasonAccepted)
}

func (s *Session) finishPacketTrace(id uint64, accepted bool, reason string) {
	s.packetTrace.mu.Lock()
	if !s.packetTrace.active || s.packetTrace.current.trace.ID != id || s.packetTrace.current.terminal {
		s.packetTrace.mu.Unlock()
		return
	}
	s.packetTrace.current.terminal = true
	state := s.packetTrace.current
	s.packetTrace.mu.Unlock()
	s.queuePacketTraceResult(packetTraceResult(state, accepted, true, PacketTraceRoleAttacker, reason))
}

// QueuePacketTraceFeedback queues a non-terminal feedback barrier for another
// participant, typically after deferred victim motion has been queued.
func (s *Session) QueuePacketTraceFeedback(id uint64, role uint8) {
	if id == 0 {
		return
	}
	s.queuePacketTraceResult(PacketTraceResult{
		ID:               id,
		Accepted:         true,
		FeedbackComplete: true,
		Role:             role,
		Reason:           PacketTraceReasonAccepted,
	})
}

func packetTraceResult(state packetTraceState, accepted, terminal bool, role uint8, reason string) PacketTraceResult {
	return PacketTraceResult{
		ID:               state.trace.ID,
		Accepted:         accepted,
		Terminal:         terminal,
		FeedbackComplete: accepted,
		Role:             role,
		Reason:           reason,
		QueueDuration:    state.queueDuration,
		HandlerDuration:  time.Since(state.startedAt),
	}
}
