package chunk

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
)

func TestFillLightUsesUniformCopyOnWriteArrays(t *testing.T) {
	c := New(mappingTestBlockRegistry{}, cube.Range{-64, 319})
	FillLight(c, 15)
	for _, y := range []int16{-64, 0, 64, 319} {
		if sky, block := c.SkyLight(3, y, 7), c.SubChunk(y).BlockLight(3, uint8(y&15), 7); sky != 15 || block != 15 {
			t.Fatalf("light at y=%d = sky %d, block %d; want 15, 15", y, sky, block)
		}
	}

	first, second := c.SubChunk(0), c.SubChunk(16)
	first.SetBlockLight(3, 0, 7, 2)
	if got := first.BlockLight(3, 0, 7); got != 2 {
		t.Fatalf("changed block light = %d, want 2", got)
	}
	if got := second.BlockLight(3, 0, 7); got != 15 {
		t.Fatalf("copy-on-write leaked into another sub-chunk: got %d, want 15", got)
	}
}

func TestFillLightRejectsInvalidLevel(t *testing.T) {
	c := New(mappingTestBlockRegistry{}, cube.Range{-64, 319})
	defer func() {
		if recover() == nil {
			t.Fatal("FillLight accepted a level above 15")
		}
	}()
	FillLight(c, 16)
}

func BenchmarkLightInitialization(b *testing.B) {
	template := New(mappingTestBlockRegistry{}, cube.Range{-64, 319})
	for x := byte(0); x < 16; x++ {
		for z := byte(0); z < 16; z++ {
			template.SetBlock(x, 64, z, 0, 1)
		}
	}
	b.Run("terrain", func(b *testing.B) {
		for range b.N {
			c := template.Clone()
			LightArea([]*Chunk{c}, 0, 0).Fill()
		}
	})
	b.Run("fixed", func(b *testing.B) {
		for range b.N {
			c := template.Clone()
			FillLight(c, 15)
		}
	})
}
