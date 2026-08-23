package chunk

import (
	"bytes"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/df-mc/dragonfly/server/block/cube"
)

type testRuntimeIDMapper map[uint32]uint32

func (m testRuntimeIDMapper) MapBlockRuntimeID(runtimeID uint32) (uint32, bool) {
	mapped, ok := m[runtimeID]
	return mapped, ok
}

func TestRemapPalettedStorageDeduplicatesAndRepackages(t *testing.T) {
	original := emptyStorage(10)
	original.Set(0, 0, 0, 11)
	original.Set(1, 0, 0, 12)
	originalPalette := append([]uint32(nil), original.palette.values...)
	originalBits := original.bitsPerIndex

	mapped := remapPalettedStorage(original, testRuntimeIDMapper{10: 1, 11: 1, 12: 2})
	if got, want := mapped.palette.values, []uint32{1, 2}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("mapped palette: got %v, want %v", got, want)
	}
	if got, want := mapped.bitsPerIndex, uint16(1); got != want {
		t.Fatalf("mapped bits per index: got %d, want %d", got, want)
	}
	if got := mapped.At(0, 0, 0); got != 1 {
		t.Fatalf("deduplicated value: got %d, want 1", got)
	}
	if got := mapped.At(1, 0, 0); got != 2 {
		t.Fatalf("distinct value: got %d, want 2", got)
	}
	if got := original.palette.values; len(got) != len(originalPalette) {
		t.Fatalf("original palette length changed: got %v, want %v", got, originalPalette)
	} else {
		for index := range got {
			if got[index] != originalPalette[index] {
				t.Fatalf("original palette changed: got %v, want %v", got, originalPalette)
			}
		}
	}
	if original.bitsPerIndex != originalBits {
		t.Fatalf("original bits per index changed: got %d, want %d", original.bitsPerIndex, originalBits)
	}
}

func TestMappedEncodingRunsBeforeCacheHash(t *testing.T) {
	registry := mappingTestBlockRegistry{}
	column := New(registry, cube.Range{0, 15})
	column.SetBlock(0, 0, 0, 0, 10)
	native := EncodeSubChunk(column, NetworkEncoding, 0)
	mapped := EncodeSubChunk(column, NetworkEncodingWithBlockMapper(testRuntimeIDMapper{0: 0, 10: 1}), 0)
	if bytes.Equal(native, mapped) {
		t.Fatal("mapped sub-chunk bytes equal native bytes")
	}
	if xxhash.Sum64(native) == xxhash.Sum64(mapped) {
		t.Fatal("mapped sub-chunk retained native cache hash")
	}
	if got := column.Block(0, 0, 0, 0); got != 10 {
		t.Fatalf("mapped encoding mutated live chunk: got %d, want 10", got)
	}
}

type mappingTestBlockRegistry struct{}

func (mappingTestBlockRegistry) BlockCount() int                       { return 11 }
func (mappingTestBlockRegistry) AirRuntimeID() uint32                  { return 0 }
func (mappingTestBlockRegistry) FilteringBlock(uint32) uint8           { return 0 }
func (mappingTestBlockRegistry) LightBlock(uint32) uint8               { return 0 }
func (mappingTestBlockRegistry) RandomTickBlock(uint32) bool           { return false }
func (mappingTestBlockRegistry) NBTBlock(uint32) bool                  { return false }
func (mappingTestBlockRegistry) LiquidDisplacingBlock(uint32) bool     { return false }
func (mappingTestBlockRegistry) LiquidBlock(uint32) bool               { return false }
func (mappingTestBlockRegistry) HashToRuntimeID(uint32) (uint32, bool) { return 0, false }
func (mappingTestBlockRegistry) RuntimeIDToHash(uint32) (uint32, bool) { return 0, false }
func (mappingTestBlockRegistry) RuntimeIDToState(runtimeID uint32) (string, map[string]any, bool) {
	return "minecraft:test", map[string]any{"id": int32(runtimeID)}, true
}
func (mappingTestBlockRegistry) StateToRuntimeID(string, map[string]any) (uint32, bool) {
	return 0, false
}
