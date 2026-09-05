package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
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

func TestActorParticleEncoding(t *testing.T) {
	s := &Session{packets: make(chan outboundMessage, 1), closeBackground: make(chan struct{}), currentEntityRuntimeID: 10}
	pos := mgl64.Vec3{1, 2, 3}
	s.ViewParticle(pos, particle.Actor{Identifier: "minecraft:lightning_bolt"})
	message := <-s.packets
	pk, ok := message.packet.(*packet.AddActor)
	if !ok || pk.EntityRuntimeID != 11 || pk.EntityUniqueID != 11 || pk.EntityType != "minecraft:lightning_bolt" || pk.Position != vec64To32(pos) {
		t.Fatalf("actor particle = %#v", message.packet)
	}
}

func TestRuntimeBackedParticleEncoding(t *testing.T) {
	for _, test := range []struct {
		name     string
		particle world.Particle
		event    int32
		data     func(*Session) int32
	}{
		{name: "item break", particle: particle.ItemBreak{Item: item.Slimeball{}}, event: packet.LevelEventParticleLegacyEvent | 14, data: func(*Session) int32 {
			runtimeID, meta, _ := world.ItemRuntimeID(item.Slimeball{})
			return int32(uint32(runtimeID)<<16 | uint32(uint16(meta)))
		}},
		{name: "terrain", particle: particle.Terrain{Block: block.Stone{}}, event: packet.LevelEventParticleLegacyEvent | 21, data: func(s *Session) int32 {
			return int32(s.br.BlockRuntimeID(block.Stone{}))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := &Session{packets: make(chan outboundMessage, 1), closeBackground: make(chan struct{}), br: world.DefaultBlockRegistry}
			s.ViewParticle(mgl64.Vec3{}, test.particle)
			pk, ok := (<-s.packets).packet.(*packet.LevelEvent)
			if !ok || pk.EventType != test.event || pk.EventData != test.data(s) {
				t.Fatalf("particle packet = %#v", pk)
			}
		})
	}
}
