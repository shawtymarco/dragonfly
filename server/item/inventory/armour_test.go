package inventory_test

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
)

func TestArmourDamageReductionUsesFixedDefencePoints(t *testing.T) {
	armour := inventory.NewArmour(nil)
	armour.Set(
		item.NewStack(item.Helmet{Tier: item.ArmourTierLeather{}}, 1),
		item.NewStack(item.Chestplate{Tier: item.ArmourTierLeather{}}, 1),
		item.NewStack(item.Leggings{Tier: item.ArmourTierLeather{}}, 1),
		item.NewStack(item.Boots{Tier: item.ArmourTierLeather{}}, 1),
	)
	if got, want := armour.DamageReduction(10, entity.AttackDamageSource{}), 2.8; math.Abs(got-want) > 1e-9 {
		t.Fatalf("DamageReduction() = %v, want %v", got, want)
	}
}
