package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestLegacyParticleEncoding(t *testing.T) {
	s := &Session{packets: make(chan outboundMessage, 1), closeBackground: make(chan struct{})}
	pos := mgl64.Vec3{1.25, 64.5, -2.75}
	s.ViewParticle(pos, particle.Legacy{ID: 42, Data: 17})

	message := <-s.packets
	pk, ok := message.packet.(*packet.LevelEvent)
	if !ok {
		t.Fatalf("packet type = %T", message.packet)
	}
	if pk.EventType != packet.LevelEventParticleLegacyEvent|42 || pk.EventData != 17 {
		t.Fatalf("legacy particle = type %#x data %d", pk.EventType, pk.EventData)
	}
	if pk.Position != vec64To32(pos) {
		t.Fatalf("position = %v, want %v", pk.Position, vec64To32(pos))
	}
}

func TestLegacyParticleRejectsInvalidID(t *testing.T) {
	s := &Session{packets: make(chan outboundMessage, 1), closeBackground: make(chan struct{})}
	for _, id := range []int32{0, -1, packet.LevelEventParticleLegacyEvent} {
		s.ViewParticle(mgl64.Vec3{}, particle.Legacy{ID: id})
		select {
		case message := <-s.packets:
			t.Fatalf("ID %d emitted %T", id, message.packet)
		default:
		}
	}
}
