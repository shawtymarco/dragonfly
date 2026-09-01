package player

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

func TestRepeatedReleasableUsePreservesStart(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })

	if err := w.Do(func(tx *world.Tx) {
		pos := mgl64.Vec3{0.5, 64, 0.5}
		pl := tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{
			UUID: uuid.New(), Name: "Archer", Position: pos,
		})).(*Player)
		_ = pl.Inventory().SetItem(0, item.NewStack(item.Bow{}, 1))
		_ = pl.Inventory().SetItem(1, item.NewStack(item.Arrow{}, 1))

		pl.UseItem()
		started := pl.usingSince
		if !pl.usingItem {
			t.Fatal("bow use did not start")
		}
		pl.UseItem()
		if !pl.usingItem || !pl.usingSince.Equal(started) {
			t.Fatal("repeated bow use restarted or stopped the release state")
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestEarlyChargeRepeatKeepsCharging(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })

	if err := w.Do(func(tx *world.Tx) {
		pos := mgl64.Vec3{0.5, 64, 0.5}
		pl := tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{
			UUID: uuid.New(), Name: "Crossbowman", Position: pos,
		})).(*Player)
		_ = pl.Inventory().SetItem(0, item.NewStack(item.Crossbow{}, 1))
		_ = pl.Inventory().SetItem(1, item.NewStack(item.Arrow{}, 1))

		pl.UseItem()
		if !pl.usingItem {
			t.Fatal("crossbow charge did not start")
		}
		pl.UseItem()
		if !pl.usingItem {
			t.Fatal("early crossbow repeat stopped charging")
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}
