package server

import (
	"image"
	"testing"

	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/df-mc/dragonfly/server/world"
)

type registryMenuItem struct{}

func (registryMenuItem) EncodeItem() (string, int16) { return "dragonfly_test:registry_menu", 0 }
func (registryMenuItem) Name() string                { return "Registry Menu" }
func (registryMenuItem) Texture() image.Image        { return image.NewRGBA(image.Rect(0, 0, 16, 16)) }
func (registryMenuItem) Category() category.Category { return category.Items() }
func (registryMenuItem) MaxCount() int               { return 1 }

func TestItemEntriesIncludeCustomAndIsolateComponents(t *testing.T) {
	if _, exists := world.ItemByName("dragonfly_test:registry_menu", 0); !exists {
		world.RegisterItem(registryMenuItem{})
	}
	entries := ItemEntries()
	seen := make(map[int16]string)
	found := false
	for _, entry := range entries {
		if other, ok := seen[entry.RuntimeID]; ok && other != entry.Name {
			t.Fatalf("runtime ID collision: %s and %s", other, entry.Name)
		}
		seen[entry.RuntimeID] = entry.Name
		if entry.Name != "dragonfly_test:registry_menu" {
			continue
		}
		found = true
		rid, _, _ := world.ItemRuntimeID(registryMenuItem{})
		if int32(entry.RuntimeID) != rid || !entry.ComponentBased {
			t.Fatal("custom identity missing")
		}
		properties := entry.Data["components"].(map[string]any)["item_properties"].(map[string]any)
		if properties["max_stack_size"] != int32(1) {
			t.Fatal("custom components missing")
		}
		properties["max_stack_size"] = int32(64)
	}
	if !found {
		t.Fatal("custom item not advertised")
	}
	for _, entry := range CustomItemEntries() {
		if entry.Name == "dragonfly_test:registry_menu" && entry.Data["components"].(map[string]any)["item_properties"].(map[string]any)["max_stack_size"] != int32(1) {
			t.Fatal("registry component mutation escaped its snapshot")
		}
	}
	for _, entry := range VanillaItemEntries() {
		if entry.Data == nil {
			continue
		}
		entry.Data["test_mutation"] = true
	}
	for _, entry := range VanillaItemEntries() {
		if _, changed := entry.Data["test_mutation"]; changed {
			t.Fatal("vanilla registry mutation escaped its snapshot")
		}
	}
}
