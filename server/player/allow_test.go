package player

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestAboveAllow(t *testing.T) {
	column := map[int]string{-3: "minecraft:allow", -1: "minecraft:grass_block", 5: "minecraft:deny"}
	name := func(pos cube.Pos) string { return column[pos.Y()] }
	for y, want := range map[int]bool{-4: false, -3: false, -2: true, 0: true, 5: true, 6: false, 20: false} {
		if got := aboveAllow(cube.Pos{0, y, 0}, -64, name); got != want {
			t.Fatalf("aboveAllow at y=%d = %v, want %v", y, got, want)
		}
	}
}
