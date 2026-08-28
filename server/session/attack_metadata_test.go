package session

import (
	"testing"
	"time"

	"github.com/go-gl/mathgl/mgl64"
)

func TestAttackMetadataLifecycle(t *testing.T) {
	s := &Session{}
	want := AttackMetadata{
		TargetRuntimeID: 42,
		HotBarSlot:      3,
		Position:        mgl64.Vec3{1, 2, 3},
		ClickedPosition: mgl64.Vec3{0.1, 0.2, 0.3},
		Latency:         75 * time.Millisecond,
	}
	s.beginAttackMetadata(want)
	got, ok := s.CurrentAttackMetadata()
	if !ok || got != want {
		t.Fatalf("attack metadata = %#v, %v, want %#v, true", got, ok, want)
	}
	s.endAttackMetadata()
	if _, ok := s.CurrentAttackMetadata(); ok {
		t.Fatal("attack metadata remained active")
	}
}
