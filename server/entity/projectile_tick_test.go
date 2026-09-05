package entity

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func TestProjectileAirborneTickHook(t *testing.T) {
	w := world.Config{}.New()
	t.Cleanup(func() { _ = w.Close() })

	calls := 0
	conf := ProjectileBehaviourConfig{Tick: func(e *Ent, tx *world.Tx) {
		calls++
		if e.Position() != (mgl64.Vec3{0.5, 64, 0.5}) || tx.World() != w {
			t.Fatalf("tick hook received position %v in world %p", e.Position(), tx.World())
		}
	}}
	handle := world.EntitySpawnOpts{Position: mgl64.Vec3{0.5, 64, 0.5}}.New(ArrowType, conf)
	mustDo(t, w, func(tx *world.Tx) {
		entity := tx.AddEntity(handle)
		entity.(world.TickerEntity).Tick(tx, 1)
	})
	if calls != 1 {
		t.Fatalf("tick hook calls = %d", calls)
	}
}

func TestThrowableConstructorsPreserveTickHook(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	calls := 0
	if err := w.Do(func(tx *world.Tx) {
		ownerHandle := world.EntitySpawnOpts{Position: mgl64.Vec3{0, 64, 0}}.New(testMovingEntType{}, testMoveConfig{})
		owner := tx.AddEntity(ownerHandle)
		for _, handle := range []*world.EntityHandle{
			NewEggWithTick(world.EntitySpawnOpts{Position: mgl64.Vec3{0, 65, 0}}, owner, func(*Ent, *world.Tx) { calls++ }),
			NewSnowballWithTick(world.EntitySpawnOpts{Position: mgl64.Vec3{0, 65, 0}}, owner, func(*Ent, *world.Tx) { calls++ }),
		} {
			projectile := tx.AddEntity(handle).(world.TickerEntity)
			projectile.Tick(tx, 1)
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("throwable tick hook calls = %d", calls)
	}
}
