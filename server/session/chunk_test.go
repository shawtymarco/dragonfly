package session

import (
	"bytes"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft"
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

type blockActorNBTTestProtocol struct{ minecraft.Protocol }

func (blockActorNBTTestProtocol) ConvertBlockActorNBT(data map[string]any) map[string]any {
	converted := make(map[string]any, len(data)+1)
	for key, value := range data {
		converted[key] = value
	}
	converted["converted"] = byte(1)
	return converted
}

func TestConvertBlockActorNBTUsesProtocolHook(t *testing.T) {
	input := map[string]any{"id": "FlowerPot"}
	converted := convertBlockActorNBT(blockActorNBTTestProtocol{}, input)
	if converted["converted"] != byte(1) || input["converted"] != nil {
		t.Fatalf("block actor conversion = %#v, input = %#v", converted, input)
	}
	if untouched := convertBlockActorNBT(nil, input); len(untouched) != 1 || untouched["id"] != "FlowerPot" {
		t.Fatalf("unmapped block actor changed: %#v", untouched)
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
