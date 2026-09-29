package player

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

func TestGroundRequiresSupportAtCurrentFeet(t *testing.T) {
	for _, test := range []struct {
		name     string
		position mgl64.Vec3
		delta    mgl64.Vec3
		blockPos cube.Pos
		block    world.Block
		grounded bool
	}{
		{"solid_floor", mgl64.Vec3{0.5, 64, 0.5}, mgl64.Vec3{0, -2}, cube.Pos{0, 63, 0}, block.Stone{}, true},
		{"edge_support", mgl64.Vec3{1.29, 64, 0.5}, mgl64.Vec3{}, cube.Pos{0, 63, 0}, block.Stone{}, true},
		{"touching_side_only", mgl64.Vec3{1.3, 64, 0.5}, mgl64.Vec3{}, cube.Pos{0, 63, 0}, block.Stone{}, false},
		{"above_floor", mgl64.Vec3{0.5, 64.02, 0.5}, mgl64.Vec3{0, -1}, cube.Pos{0, 63, 0}, block.Stone{}, false},
		{"encoded_just_above_floor", mgl64.Vec3{0.5, 64.00001, 0.5}, mgl64.Vec3{}, cube.Pos{0, 63, 0}, block.Stone{}, true},
		{"encoded_just_below_floor", mgl64.Vec3{0.5, 63.99999, 0.5}, mgl64.Vec3{}, cube.Pos{0, 63, 0}, block.Stone{}, true},
		{"wall_overlap", mgl64.Vec3{0.71, 64, 0.5}, mgl64.Vec3{0, -1}, cube.Pos{1, 64, 0}, block.Stone{}, false},
		{"ceiling_overlap", mgl64.Vec3{0.5, 64, 0.5}, mgl64.Vec3{0, -1}, cube.Pos{0, 65, 0}, block.Stone{}, false},
		{"departed_ledge", mgl64.Vec3{1.5, 63.5, 0.5}, mgl64.Vec3{1, -0.5}, cube.Pos{0, 63, 0}, block.Stone{}, false},
		{"diagonal_corner", mgl64.Vec3{1.5, 64, 1.5}, mgl64.Vec3{1, -2, 1}, cube.Pos{0, 64, 1}, block.Stone{}, false},
		{"bottom_slab", mgl64.Vec3{0.5, 63.5, 0.5}, mgl64.Vec3{0, -1}, cube.Pos{0, 63, 0}, block.Slab{Block: block.Stone{}}, true},
		{"above_bottom_slab", mgl64.Vec3{0.5, 64, 0.5}, mgl64.Vec3{0, -1}, cube.Pos{0, 63, 0}, block.Slab{Block: block.Stone{}}, false},
		{"top_slab", mgl64.Vec3{0.5, 64, 0.5}, mgl64.Vec3{0, -1}, cube.Pos{0, 63, 0}, block.Slab{Block: block.Stone{}, Top: true}, true},
		{"lower_stair", mgl64.Vec3{0.5, 63.5, 0.85}, mgl64.Vec3{0, -0.5}, cube.Pos{0, 63, 0}, block.Stairs{Block: block.Stone{}, Facing: cube.North}, true},
		{"upper_stair", mgl64.Vec3{0.5, 64, 0.15}, mgl64.Vec3{0, -0.5}, cube.Pos{0, 63, 0}, block.Stairs{Block: block.Stone{}, Facing: cube.North}, true},
		{"fence_top", mgl64.Vec3{0.5, 64.5, 0.5}, mgl64.Vec3{0, -1}, cube.Pos{0, 63, 0}, block.WoodFence{}, true},
		{"fence_side", mgl64.Vec3{0.5, 64, 0.5}, mgl64.Vec3{0, -1}, cube.Pos{0, 63, 0}, block.WoodFence{}, false},
		{"negative_coordinates", mgl64.Vec3{-0.5, -3, -0.5}, mgl64.Vec3{0, -1}, cube.Pos{-1, -4, -1}, block.Stone{}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := world.Config{Synchronous: true}.New()
			t.Cleanup(func() { _ = w.Close() })
			if err := w.Do(func(tx *world.Tx) {
				tx.SetBlock(test.blockPos, test.block, nil)
				p := tx.AddEntity(world.EntitySpawnOpts{ID: uuid.New()}.New(Type, Config{Position: test.position.Sub(test.delta)})).(*Player)
				p.Move(test.delta, 0, 0)
				if got := p.onGround; got != test.grounded {
					t.Fatalf("grounded = %v, want %v", got, test.grounded)
				}
			}).Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
