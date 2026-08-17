package item

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// FishingRod is a durable item that casts a fishing hook. Using it again
// retrieves the hook, pulling any hooked player or item towards the user.
type FishingRod struct{}

// MaxCount ...
func (FishingRod) MaxCount() int {
	return 1
}

// DurabilityInfo ...
func (FishingRod) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability: 384,
		BrokenItem:    simpleItem(Stack{}),
	}
}

// FuelInfo ...
func (FishingRod) FuelInfo() FuelInfo {
	return newFuelInfo(time.Second * 10)
}

// EnchantmentValue ...
func (FishingRod) EnchantmentValue() int {
	return 1
}

// HandEquipped ...
func (FishingRod) HandEquipped() bool {
	return true
}

// Use casts a fishing hook or retrieves the user's existing hook.
func (FishingRod) Use(tx *world.Tx, user User, ctx *UseContext) bool {
	if cd, ok := user.(interface {
		HasCooldown(world.Item) bool
		SetCooldown(world.Item, time.Duration)
	}); ok {
		if cd.HasCooldown(FishingRod{}) {
			return false
		}
		cd.SetCooldown(FishingRod{}, time.Millisecond*200)
	}
	use := tx.World().EntityRegistry().Config().UseFishingRod
	if use == nil {
		return false
	}
	if use(tx, user) {
		ctx.DamageItem(1)
	}
	return true
}

// UseOnBlock casts or retrieves the hook when the rod is used on a block.
// Player.UseItemOnBlock prefers this over activating the clicked block, except
// for ender chests.
func (f FishingRod) UseOnBlock(_ cube.Pos, _ cube.Face, _ mgl64.Vec3, tx *world.Tx, user User, ctx *UseContext) bool {
	return f.Use(tx, user, ctx)
}

// EncodeItem ...
func (FishingRod) EncodeItem() (name string, meta int16) {
	return "minecraft:fishing_rod", 0
}
