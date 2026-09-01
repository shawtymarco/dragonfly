package player

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

type itemUseCountingHandler struct {
	NopHandler
	uses, consumes int
}

func (h *itemUseCountingHandler) HandleItemUse(*Context) { h.uses++ }

func (h *itemUseCountingHandler) HandleItemConsume(*Context, item.Stack) { h.consumes++ }

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
		pl.ReleaseItem()
		if pl.usingItem {
			t.Fatal("bow release did not complete the use cycle")
		}
		pl.usingSince = time.Time{}
		pl.UseItem()
		if !pl.usingItem || pl.usingSince.IsZero() {
			t.Fatal("continued input did not start a new bow use cycle")
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
		handler := &itemUseCountingHandler{}
		pl.Handle(handler)
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
		if handler.uses != 1 {
			t.Fatalf("item-use handler calls = %d, want 1", handler.uses)
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestConsumableRepeatsValidateDurationAndRestart(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })

	if err := w.Do(func(tx *world.Tx) {
		pos := mgl64.Vec3{0.5, 64, 0.5}
		pl := tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{
			UUID: uuid.New(), Name: "Eater", Position: pos, Food: 10,
		})).(*Player)
		handler := &itemUseCountingHandler{}
		pl.Handle(handler)
		_ = pl.Inventory().SetItem(0, item.NewStack(item.Apple{}, 2))

		pl.UseItem()
		if !pl.usingItem || handler.uses != 1 {
			t.Fatalf("initial consumption state = %v, uses = %d", pl.usingItem, handler.uses)
		}
		pl.usingSince = time.Now()
		pl.UseItem()
		held, _ := pl.HeldItems()
		if held.Count() != 2 || handler.uses != 1 || handler.consumes != 0 {
			t.Fatalf("early repeat changed stack/events: count=%d uses=%d consumes=%d", held.Count(), handler.uses, handler.consumes)
		}

		pl.usingSince = time.Now().Add(-2 * time.Second)
		pl.UseItem()
		held, _ = pl.HeldItems()
		if pl.usingItem || held.Count() != 1 || handler.uses != 1 || handler.consumes != 1 {
			t.Fatalf("completed consume state=%v count=%d uses=%d consumes=%d", pl.usingItem, held.Count(), handler.uses, handler.consumes)
		}

		pl.UseItem()
		if !pl.usingItem || handler.uses != 2 {
			t.Fatalf("held input did not restart next item: state=%v uses=%d", pl.usingItem, handler.uses)
		}
		pl.usingSince = time.Now().Add(-2 * time.Second)
		pl.UseItem()
		held, _ = pl.HeldItems()
		if pl.usingItem || !held.Empty() || handler.consumes != 2 {
			t.Fatalf("final consume left stale use state: state=%v held=%+v consumes=%d", pl.usingItem, held, handler.consumes)
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}
