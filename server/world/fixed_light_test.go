package world

import "testing"

func TestWorldFixedLightAppliesOnChunkLoad(t *testing.T) {
	fixed := Config{Synchronous: true, FixedLightLevel: 15}.New()
	normal := Config{Synchronous: true}.New()
	t.Cleanup(func() {
		_ = normal.Close()
		_ = fixed.Close()
	})
	for name, test := range map[string]struct {
		world *World
		want  uint8
	}{
		"fixed":  {world: fixed, want: 15},
		"normal": {world: normal, want: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if err := test.world.Do(func(tx *Tx) {
				column := tx.chunk(ChunkPos{})
				if got := column.SubChunk(64).BlockLight(0, 0, 0); got != test.want {
					t.Fatalf("block light = %d, want %d", got, test.want)
				}
			}).Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWorldRejectsInvalidFixedLightLevel(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("world accepted a fixed light level above 15")
		}
	}()
	_ = Config{Synchronous: true, FixedLightLevel: 16}.New()
}
