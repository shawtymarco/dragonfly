package chunk

import "testing"

func TestBlockPaletteEncodingAcceptsInt32LegacyMeta(t *testing.T) {
	registry := legacyMetaTestRegistry{}
	for attempt := 0; attempt < 2; attempt++ {
		runtimeID, err := (BlockPaletteEncoding{Blocks: registry}).DecodeBlockState(map[string]any{
			"name": "minecraft:concrete",
			"val":  int32(4),
		})
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if runtimeID != 42 {
			t.Fatalf("attempt %d runtime ID = %d, want 42", attempt, runtimeID)
		}
	}
}

func TestLegacyBlockMetaRejectsOverflow(t *testing.T) {
	if _, ok := legacyBlockMeta(int32(1 << 16)); ok {
		t.Fatal("overflowing metadata was accepted")
	}
}

type legacyMetaTestRegistry struct{ mappingTestBlockRegistry }

func (legacyMetaTestRegistry) StateToRuntimeID(name string, properties map[string]any) (uint32, bool) {
	if name == "minecraft:yellow_concrete" && len(properties) == 0 {
		return 42, true
	}
	return 0, false
}
