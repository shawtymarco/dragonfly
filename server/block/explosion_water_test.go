package block

import (
	"math/rand/v2"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
)

func TestExplosionWaterResistanceOptIn(t *testing.T) {
	for _, test := range []struct {
		name        string
		ignoreWater bool
		wantBroken  bool
	}{
		{name: "default water shield", wantBroken: false},
		{name: "water transparent to blast", ignoreWater: true, wantBroken: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := world.Config{Synchronous: true}.New()
			t.Cleanup(func() { _ = w.Close() })
			var broken bool
			err := w.Do(func(tx *world.Tx) {
				origin := cube.Pos{0, 64, 0}
				target := origin.Side(cube.FaceEast)
				tx.SetBlock(target, Wool{Colour: item.ColourWhite()}, nil)
				tx.SetLiquid(origin, Water{Still: true, Depth: 8})
				ExplosionConfig{IgnoreWaterResistance: test.ignoreWater, ItemDropChance: -1, RandSource: rand.NewPCG(1, 2)}.
					Explode(tx, world.BlockExplosionSource{Pos: origin, ExplosionSize: 4})
				_, broken = tx.Block(target).(Air)
			}).Wait(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if broken != test.wantBroken {
				t.Fatalf("neighbour broken = %t, want %t", broken, test.wantBroken)
			}
		})
	}
}
