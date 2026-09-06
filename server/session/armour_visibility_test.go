package session

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type armourPolicyActor struct {
	*spawnAttributeActor
	armour  *inventory.Armour
	visible bool
	viewer  *world.EntityHandle
}

func (a *armourPolicyActor) Armour() *inventory.Armour { return a.armour }
func (a *armourPolicyActor) Invisible() bool           { return !a.visible }
func (a *armourPolicyActor) ArmourVisible(viewer *world.EntityHandle) bool {
	a.viewer = viewer
	return a.visible
}

func TestArmourPolicyCoversSpawnResendAndReentry(t *testing.T) {
	a := &armourPolicyActor{spawnAttributeActor: newSpawnAttributeActor(t, "minecraft:zombie"), armour: inventory.NewArmour(nil)}
	a.armour.Set(item.NewStack(item.Helmet{Tier: item.ArmourTierIron{}}, 1), item.NewStack(item.Chestplate{Tier: item.ArmourTierIron{}}, 1), item.NewStack(item.Leggings{Tier: item.ArmourTierIron{}}, 1), item.NewStack(item.Boots{Tier: item.ArmourTierIron{}}, 1))
	observer := newSpawnAttributeActor(t, "minecraft:player")
	s := &Session{conf: Config{Log: slog.Default()}, ent: observer.H(), br: world.DefaultBlockRegistry,
		currentEntityRuntimeID: selfEntityRuntimeID,
		entityRuntimeIDs:       map[*world.EntityHandle]uint64{}, entities: map[uint64]*world.EntityHandle{},
		hiddenEntities: map[uuid.UUID]struct{}{}, packets: make(chan outboundMessage, 32), closeBackground: make(chan struct{})}
	check := func(visible bool) {
		t.Helper()
		var armour *packet.MobArmourEquipment
		for len(s.packets) > 0 {
			pk := (<-s.packets).packet
			switch pk := pk.(type) {
			case *packet.MobArmourEquipment:
				armour = pk
			case *packet.AddActor:
				if pk.EntityMetadata.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagInvisible) == visible {
					t.Fatal("spawn lost invisibility metadata")
				}
			case *packet.SetActorData:
				if pk.EntityMetadata.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagInvisible) == visible {
					t.Fatal("state resend lost invisibility metadata")
				}
			}
		}
		if armour == nil {
			t.Fatal("no armour packet")
		}
		buf := bytes.NewBuffer(nil)
		armour.Marshal(protocol.NewWriter(buf, 0))
		decoded := &packet.MobArmourEquipment{}
		decoded.Marshal(protocol.NewReader(buf, 0, true))
		if buf.Len() != 0 || decoded.EntityRuntimeID == 0 || a.viewer != observer.H() {
			t.Fatal("armour packet identity or wire framing is invalid")
		}
		for _, slot := range []protocol.ItemInstance{decoded.Helmet, decoded.Chestplate, decoded.Leggings, decoded.Boots} {
			if (slot.Stack.NetworkID != 0) != visible {
				t.Fatalf("armour exposed=%t, want %t", slot.Stack.NetworkID != 0, visible)
			}
		}
	}
	// Unknown actors must not receive equipment with runtime ID zero.
	s.ViewEntityArmour(a)
	if len(s.packets) != 0 {
		t.Fatal("equipment sent before actor spawn")
	}
	for _, visible := range []bool{false, true, false} {
		a.visible = visible
		s.ViewEntity(a)
		s.ViewEntityArmour(a)
		check(visible)
		a.armour.SetHelmet(item.NewStack(item.Helmet{Tier: item.ArmourTierDiamond{}}, 1))
		s.ViewEntityState(a)
		s.ViewEntityArmour(a)
		check(visible)
		s.HideEntity(a)
		s.ViewEntity(a)
		s.ViewEntityArmour(a)
		check(visible)
		s.StopShowingEntity(a)
		for len(s.packets) > 0 {
			<-s.packets
		}
		s.ViewEntityArmour(a)
		if len(s.packets) != 0 {
			t.Fatal("hidden actor received armour")
		}
		s.StartShowingEntity(a)
		check(visible)
	}
	if a.armour.Helmet().Empty() || a.armour.Boots().Empty() {
		t.Fatal("network visibility changed equipped armour")
	}
	s.entityRuntimeIDs[a.H()] = selfEntityRuntimeID
	s.ViewEntityArmour(a)
	if len(s.packets) != 0 {
		t.Fatal("self armour echoed")
	}
}
