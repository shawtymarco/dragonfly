package session

import (
	"bytes"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
)

func TestBorderBlockData(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	registry := world.NewBlockRegistry()
	registry.Finalize()
	c := chunk.New(registry, cube.Range{-64, 319})
	rid := registry.BlockRuntimeID(block.Border{})
	c.SetBlock(2, -20, 3, 0, rid)
	c.SetBlock(2, 100, 3, 0, rid)
	c.SetBlock(15, 200, 15, 0, rid)

	got := borderBlockData(c, registry)
	want := []byte{2, 0x32, 0xff}
	if !bytes.Equal(got, want) {
		t.Fatalf("borderBlockData() = %v, want %v", got, want)
	}
}

func TestBorderBlockDataWithoutBorders(t *testing.T) {
	world.DefaultBlockRegistry.Finalize()
	registry := world.NewBlockRegistry()
	registry.Finalize()
	c := chunk.New(registry, cube.Range{-64, 319})
	if got := borderBlockData(c, registry); !bytes.Equal(got, []byte{0}) {
		t.Fatalf("borderBlockData() = %v, want [0]", got)
	}
}
