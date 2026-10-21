package world_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

func TestNativeSnapshotProvenance(t *testing.T) {
	data, err := os.ReadFile("native_snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Protocol int `json:"protocol"`
		Files    []struct {
			Destination, SHA256 string
			Bytes               int
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Protocol != 2225 || len(manifest.Files) != 9 {
		t.Fatal("unexpected native source manifest")
	}
	for _, file := range manifest.Files {
		t.Run(file.Destination, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(file.Destination)))
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(data)
			if len(data) != file.Bytes || hex.EncodeToString(hash[:]) != file.SHA256 {
				t.Fatal("native asset differs from the exact-build provenance manifest")
			}
		})
	}
}

func TestNativeDerivedStateDefaultsPreserveStoredGameplay(t *testing.T) {
	registry := world.DefaultBlockRegistry
	registry.Finalize()
	for _, test := range []struct {
		name, key string
		value     int32
	}{
		{"minecraft:fire", "age", 14}, {"minecraft:redstone_wire", "redstone_signal", 13},
	} {
		t.Run(test.name, func(t *testing.T) {
			properties := map[string]any{test.key: test.value}
			id, ok := registry.StateToRuntimeID(test.name, properties)
			if !ok {
				t.Fatal("stored block no longer resolves")
			}
			name, resolved, ok := registry.RuntimeIDToState(id)
			if !ok || name != test.name || resolved[test.key] != test.value {
				t.Fatalf("stored gameplay property reset: %s %#v", name, resolved)
			}
			if len(properties) != 1 {
				t.Fatal("source properties mutated")
			}
		})
	}
}
