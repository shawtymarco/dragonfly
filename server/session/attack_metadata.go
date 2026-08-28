package session

import (
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl64"
)

// AttackMetadata holds the client-reported semantic fields for the entity
// attack currently handled by a Session. Position fields are diagnostic input
// and must not be trusted as authoritative player state.
type AttackMetadata struct {
	TargetRuntimeID uint64
	HotBarSlot      int32
	Position        mgl64.Vec3
	ClickedPosition mgl64.Vec3
	Latency         time.Duration
}

type attackMetadataTracker struct {
	mu      sync.RWMutex
	current AttackMetadata
	active  bool
}

func (s *Session) beginAttackMetadata(metadata AttackMetadata) {
	s.attackMeta.mu.Lock()
	s.attackMeta.current = metadata
	s.attackMeta.active = true
	s.attackMeta.mu.Unlock()
}

func (s *Session) endAttackMetadata() {
	s.attackMeta.mu.Lock()
	s.attackMeta.current = AttackMetadata{}
	s.attackMeta.active = false
	s.attackMeta.mu.Unlock()
}

// CurrentAttackMetadata returns metadata for the entity attack currently
// handled on the world owner.
func (s *Session) CurrentAttackMetadata() (AttackMetadata, bool) {
	s.attackMeta.mu.RLock()
	defer s.attackMeta.mu.RUnlock()
	return s.attackMeta.current, s.attackMeta.active
}
