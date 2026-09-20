package player

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

type scaledExplosionSource struct {
	position  mgl64.Vec3
	damage    float64
	knockback float64
}

func (s scaledExplosionSource) Position() mgl64.Vec3                  { return s.position }
func (scaledExplosionSource) Size() float64                           { return 4 }
func (s scaledExplosionSource) ExplosionDamageMultiplier() float64    { return s.damage }
func (s scaledExplosionSource) ExplosionKnockbackMultiplier() float64 { return s.knockback }

func TestExplosionSourceCanScaleDamageAndKnockbackIndependently(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	var baseHealth, scaledHealth, baseVelocity, scaledVelocity float64
	err := w.Do(func(tx *world.Tx) {
		origin := mgl64.Vec3{0.5, 64.5, 0.5}
		basePos := mgl64.Vec3{1.5, 64.5, 0.5}
		scaledPos := mgl64.Vec3{2.5, 64.5, 0.5}
		base := tx.AddEntity(world.EntitySpawnOpts{Position: basePos}.New(Type, Config{
			UUID: uuid.New(), Name: "Base", Position: basePos,
		})).(*Player)
		scaled := tx.AddEntity(world.EntitySpawnOpts{Position: scaledPos}.New(Type, Config{
			UUID: uuid.New(), Name: "Scaled", Position: scaledPos,
		})).(*Player)
		base.Explode(world.BlockExplosionSource{Pos: cube.Pos{0, 64, 0}, ExplosionSize: 4}, 0.1)
		scaled.Explode(scaledExplosionSource{position: origin, damage: 4, knockback: 2}, 0.1)
		baseHealth, scaledHealth = base.Health(), scaled.Health()
		baseVelocity, scaledVelocity = base.Velocity()[0], scaled.Velocity()[0]
	}).Wait(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseHealth != 16 || scaledHealth != 4 {
		t.Fatalf("health after blast = %v/%v, want 16/4", baseHealth, scaledHealth)
	}
	if math.Abs(scaledVelocity-baseVelocity*2) > 1e-9 {
		t.Fatalf("horizontal knockback = %v/%v, want scaled twice base", baseVelocity, scaledVelocity)
	}
}
