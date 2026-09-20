package chunk

import (
	"bytes"
	"github.com/cespare/xxhash/v2"
	"github.com/df-mc/dragonfly/server/block/cube"
	"testing"
)

type testBiomeMapper struct{ protocol419TestRuntimeIDMapper }

func (testBiomeMapper) MapBiomeRuntimeID(id uint32) (uint32, bool) {
	if id == 195 {
		return 4, true
	}
	return id, true
}

func TestBiomeMappingPrecedesHashAndPreservesWorld(t *testing.T) {
	c := New(mappingTestBlockRegistry{}, cube.Range{0, 255})
	c.SetBiome(0, 0, 0, 195)
	c.SetBiome(4, 0, 0, 4)
	mapper := testBiomeMapper{protocol419TestRuntimeIDMapper{testRuntimeIDMapper{0: 0}}}
	e := NetworkEncodingWithBlockMapper(mapper)
	native, mapped := EncodeBiomes(c, NetworkEncoding), EncodeBiomes(c, e)
	if bytes.Equal(native, mapped) || xxhash.Sum64(native) == xxhash.Sum64(mapped) {
		t.Fatal("biome bytes and cache hashes were not translated")
	}
	reader := bytes.NewBuffer(mapped)
	first, err := decodePalettedStorage(reader, NetworkEncoding, BiomePaletteEncoding)
	if err != nil {
		t.Fatal(err)
	}
	if first.At(0, 0, 0) != 4 || first.At(4, 0, 0) != 4 {
		t.Fatal("mapped palette has wrong biome semantics")
	}
	if c.Biome(0, 0, 0) != 195 {
		t.Fatal("encoding modified the live biome")
	}
	flat := encodeBiomes2D(c, e, 0, 255)
	if flat[0] != 4 {
		t.Fatalf("2D biome = %d, want forest 4", flat[0])
	}
	if c.Biome(0, 0, 0) != 195 {
		t.Fatal("2D encoding modified the live biome")
	}
}
