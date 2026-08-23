package block

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

func TestBorderStatesRegistered(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	registry := world.NewBlockRegistry()
	registry.Finalize()
	for _, state := range allBorders() {
		rid := registry.BlockRuntimeID(state)
		got, ok := registry.BlockByRuntimeID(rid)
		if !ok {
			t.Fatalf("border state with runtime ID %d was not registered", rid)
		}
		if _, ok := got.(Border); !ok {
			t.Fatalf("runtime ID %d resolved to %T, want Border", rid, got)
		}
	}
}

func TestBorderModelExtendsVertically(t *testing.T) {
	b := Border{Post: true}
	boxes := b.Model().BBox(cube.Pos{}, nil)
	if len(boxes) != 1 {
		t.Fatalf("Border.Model().BBox() returned %d boxes, want 1", len(boxes))
	}
	if min, max := boxes[0].Min(), boxes[0].Max(); min.Y() != 0 || max.Y() != math.MaxFloat64 {
		t.Fatalf("Border.Model().BBox() Y bounds = [%v, %v], want [0, %v]", min.Y(), max.Y(), math.MaxFloat64)
	}
}
