package session

import (
	"bytes"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestViewEntityIncludesInitialHealth(t *testing.T) {
	base := newSpawnAttributeActor(t, "minecraft:ender_dragon")
	actor := &spawnHealthActor{spawnAttributeActor: base, health: 20, maximum: 20}
	for _, value := range []float64{20, 7.5, 0} {
		actor.health = value
		pk := spawnAttributePacket(t, actor)
		if pk.EntityType != "minecraft:ender_dragon" {
			t.Fatalf("spawn actor = %q", pk.EntityType)
		}
		if len(pk.Attributes) != 1 {
			t.Fatalf("initial health missing from AddActor: %+v", pk.Attributes)
		}
		health := pk.Attributes[0]
		if health.Name != "minecraft:health" || health.Min != 0 || health.Max != 20 || health.Value != float32(value) {
			t.Fatalf("spawn health = %+v, want current %v/max 20", health, value)
		}
		if actor.health != value || actor.maximum != 20 {
			t.Fatal("viewing the actor changed its server-side health")
		}
	}
}

func TestViewEntityWithoutHealthHasNoHealthAttribute(t *testing.T) {
	actor := newSpawnAttributeActor(t, "minecraft:ender_crystal")
	if pk := spawnAttributePacket(t, actor); len(pk.Attributes) != 0 {
		t.Fatalf("non-living actor received health: %+v", pk.Attributes)
	}
}

func spawnAttributePacket(t *testing.T, actor world.Entity) *packet.AddActor {
	t.Helper()
	s := &Session{
		entityRuntimeIDs: map[*world.EntityHandle]uint64{},
		entities:         map[uint64]*world.EntityHandle{},
		packets:          make(chan outboundMessage, 2),
		closeBackground:  make(chan struct{}),
	}
	s.ViewEntity(actor)
	select {
	case message := <-s.packets:
		pk, ok := message.packet.(*packet.AddActor)
		if !ok {
			t.Fatalf("spawn packet = %T, want AddActor", message.packet)
		}
		// Exercise the actual wire schema as well as the packet construction.
		buf := bytes.NewBuffer(nil)
		pk.Marshal(protocol.NewWriter(buf, 0))
		decoded := &packet.AddActor{}
		decoded.Marshal(protocol.NewReader(buf, 0, true))
		if buf.Len() != 0 {
			t.Fatalf("spawn decode left %d bytes", buf.Len())
		}
		return decoded
	default:
		t.Fatal("no spawn packet queued")
		return nil
	}
}

type spawnAttributeActor struct{ handle *world.EntityHandle }

func (a *spawnAttributeActor) H() *world.EntityHandle { return a.handle }
func (*spawnAttributeActor) Position() mgl64.Vec3     { return mgl64.Vec3{} }
func (*spawnAttributeActor) Rotation() cube.Rotation  { return cube.Rotation{} }
func (a *spawnAttributeActor) Close() error           { return a.handle.Close() }

type spawnHealthActor struct {
	*spawnAttributeActor
	health, maximum float64
}

func (a *spawnHealthActor) Health() float64    { return a.health }
func (a *spawnHealthActor) MaxHealth() float64 { return a.maximum }

type spawnAttributeType struct{ id string }

func (t spawnAttributeType) EncodeEntity() string { return t.id }
func (spawnAttributeType) BBox(world.Entity) cube.BBox {
	return cube.Box(-.5, 0, -.5, .5, 1, .5)
}
func (spawnAttributeType) Open(_ *world.Tx, h *world.EntityHandle, _ *world.EntityData) world.Entity {
	return &spawnAttributeActor{handle: h}
}
func (spawnAttributeType) DecodeNBT(map[string]any, *world.EntityData) {}
func (spawnAttributeType) EncodeNBT(*world.EntityData) map[string]any  { return nil }
func (spawnAttributeType) Apply(*world.EntityData)                     {}

func newSpawnAttributeActor(t *testing.T, id string) *spawnAttributeActor {
	t.Helper()
	typ := spawnAttributeType{id: id}
	handle := (world.EntitySpawnOpts{}).New(typ, typ)
	t.Cleanup(func() { _ = handle.Close() })
	return &spawnAttributeActor{handle: handle}
}
