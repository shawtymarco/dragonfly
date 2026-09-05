package player

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

func TestNaturalRegenerationUsesUniformEightyTickInterval(t *testing.T) {
	for _, test := range []struct {
		name       string
		difficulty world.Difficulty
	}{
		{name: "normal with saturation", difficulty: world.DifficultyNormal},
		{name: "peaceful with saturation", difficulty: world.DifficultyPeaceful},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := world.Config{Synchronous: true}.New()
			t.Cleanup(func() { _ = w.Close() })
			w.SetDifficulty(test.difficulty)

			if err := w.Do(func(tx *world.Tx) {
				pos := mgl64.Vec3{0.5, 64, 0.5}
				pl := tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{
					UUID: uuid.New(), Name: "Regeneration", Position: pos, GameMode: world.GameModeSurvival,
				})).(*Player)
				pl.health.AddHealth(-10)
				pl.hunger.foodTick = 2

				for range 79 {
					pl.tickFood()
				}
				if got := pl.Health(); got != 10 {
					t.Fatalf("health before first interval = %v, want 10", got)
				}
				pl.tickFood()
				if got := pl.Health(); got != 11 {
					t.Fatalf("health after first interval = %v, want 11", got)
				}

				for range 79 {
					pl.tickFood()
				}
				if got := pl.Health(); got != 11 {
					t.Fatalf("health before second interval = %v, want 11", got)
				}
				pl.tickFood()
				if got := pl.Health(); got != 12 {
					t.Fatalf("health after second interval = %v, want 12", got)
				}
			}).Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
