package chunk

import "testing"

type derivedStateTestRegistry struct{ mappingTestBlockRegistry }

func (derivedStateTestRegistry) StateToRuntimeID(name string, props map[string]any) (uint32, bool) {
	return 42, name == "minecraft:oak_stairs" && props["minecraft:corner"] == "none"
}

func TestAddedDerivedPropertiesMarkExistingVersionForRepair(t *testing.T) {
	legacy := false
	properties := map[string]any{"weirdo_direction": int32(0), "upside_down_bit": byte(0)}
	_, err := (BlockPaletteEncoding{Blocks: derivedStateTestRegistry{}, legacy: &legacy}).DecodeBlockState(map[string]any{
		"name": "minecraft:oak_stairs", "version": CurrentBlockVersion, "states": properties,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !legacy {
		t.Fatal("new corner defaults must be derived from neighbours even when the stored block version is unchanged")
	}
	if _, mutated := properties["minecraft:corner"]; mutated {
		t.Fatal("state upgrade mutated input properties")
	}
}
