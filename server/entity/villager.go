package entity

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// NewVillager creates a Villager that stands still at the position passed. It has no behaviour of its own:
// interacting with it is left to the player handler, which may open a trade window for it.
func NewVillager(name string, pos mgl64.Vec3, rot cube.Rotation) *world.EntityHandle {
	return world.EntitySpawnOpts{Position: pos, Rotation: rot, NameTag: name}.New(VillagerType, villagerConf{})
}

type villagerConf struct{}

// Apply ...
func (villagerConf) Apply(data *world.EntityData) {
	data.Data = villagerBehaviour{StationaryBehaviourConfig{}.New()}
}

// villagerBehaviour is a StationaryBehaviour that stops the client from moving the villager by itself.
type villagerBehaviour struct {
	*StationaryBehaviour
}

// Immobile always returns true.
func (villagerBehaviour) Immobile() bool { return true }

// InteractText returns the button touch clients show when aiming at the villager.
func (villagerBehaviour) InteractText() string { return "거래" }

// VillagerType is a world.EntityType implementation for a Villager.
var VillagerType villagerType

type villagerType struct{}

func (villagerType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	return Open(tx, handle, data)
}
func (villagerType) EncodeEntity() string        { return "dragonfly:villager" }
func (villagerType) NetworkEncodeEntity() string { return "minecraft:villager" }
func (villagerType) BBox(world.Entity) cube.BBox {
	return cube.Box(-0.3, 0, -0.3, 0.3, 1.9, 0.3)
}

func (villagerType) DecodeNBT(_ map[string]any, data *world.EntityData) {
	villagerConf{}.Apply(data)
}
func (villagerType) EncodeNBT(*world.EntityData) map[string]any { return nil }
