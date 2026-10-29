package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

func TestEffectiveChunkRadius(t *testing.T) {
	tests := []struct {
		name               string
		requested, maximum int32
		want               int32
	}{
		{name: "request below policy", requested: 4, maximum: 10, want: 4},
		{name: "request at policy", requested: 4, maximum: 4, want: 4},
		{name: "request above policy", requested: 10, maximum: 4, want: 4},
		{name: "protocol maximum", requested: 10, maximum: 9, want: 9},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := effectiveChunkRadius(test.requested, test.maximum); got != test.want {
				t.Fatalf("effective chunk radius = %d, want %d", got, test.want)
			}
		})
	}
}

func TestResizeChunkLoaderRequiresOwningWorldTransaction(t *testing.T) {
	oldWorld := world.Config{Synchronous: true}.New()
	newWorld := world.Config{Synchronous: true}.New()
	t.Cleanup(func() {
		_ = newWorld.Close()
		_ = oldWorld.Close()
	})

	loader := world.NewLoader(8, oldWorld, world.NopViewer{})
	s := &Session{chunkLoader: loader, chunkRadius: 4}
	if err := newWorld.Do(func(tx *world.Tx) {
		if s.resizeChunkLoader(tx) {
			t.Fatal("loader radius changed through a transaction from another world")
		}
		loader.ChangeWorld(tx, newWorld)
		if !s.resizeChunkLoader(tx) {
			t.Fatal("loader radius was not applied after its world switch")
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}
