package world

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// An old match may retain an original participant handle after that player
// moves to Hub. Its lookup must not race Hub's remove/add binding updates.
func TestEntityLookupWhileAnotherOwnerRebindsHandle(t *testing.T) {
	previous, current := Config{}.New(), Config{}.New()
	t.Cleanup(func() { _ = current.Close(); _ = previous.Close() })
	handle := NewEntity(taskTestEntityType{}, taskTestEntityConfig{})
	if err := current.Do(func(tx *Tx) { tx.AddEntity(handle) }).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	reader := previous.Do(func(tx *Tx) {
		<-start
		for range 10000 {
			if _, ok := handle.Entity(tx); ok {
				t.Error("old owner resolved a foreign entity")
				return
			}
			runtime.Gosched()
		}
	})
	writer := current.Do(func(tx *Tx) {
		<-start
		for range 10000 {
			entity, ok := handle.Entity(tx)
			if !ok {
				t.Error("current owner lost entity")
				return
			}
			tx.RemoveEntity(entity)
			runtime.Gosched()
			tx.AddEntity(handle)
		}
	})
	close(start)
	if err := reader.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

type conditionCheckingEntityType struct{ taskTestEntityType }

func (conditionCheckingEntityType) Open(tx *Tx, handle *EntityHandle, data *EntityData) Entity {
	handle.cond.L.Lock()
	handle.cond.L.Unlock()
	return taskTestEntityType{}.Open(tx, handle, data)
}

func TestEntityLookupReleasesConditionBeforeOpen(t *testing.T) {
	w := Config{}.New()
	t.Cleanup(func() { _ = w.Close() })
	handle := NewEntity(conditionCheckingEntityType{}, taskTestEntityConfig{})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if err := w.Do(func(tx *Tx) { tx.AddEntity(handle) }).Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if err := handle.Do(func(tx *Tx, entity Entity) {
		tx.RemoveEntity(entity)
		tx.AddEntity(handle)
	}).Wait(ctx); err != nil {
		t.Fatalf("entity callback could not rebind itself: %v", err)
	}
}

func TestWorldlessEntityLookupDoesNotWaitForConditionOrDestination(t *testing.T) {
	w := Config{}.New()
	t.Cleanup(func() { _ = w.Close() })
	handle := NewEntity(taskTestEntityType{}, taskTestEntityConfig{})
	if err := w.Do(func(tx *Tx) {
		tx.RemoveEntity(tx.AddEntity(handle))
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	handle.cond.L.Lock()
	task := w.Do(func(tx *Tx) {
		if _, ok := handle.Entity(tx); ok {
			t.Error("worldless handle was resolved")
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	err := task.Wait(ctx)
	// Release even on failure so a locking regression cannot deadlock cleanup.
	handle.cond.L.Unlock()
	if err != nil {
		t.Fatalf("worldless lookup waited for the condition lock: %v", err)
	}
	if err := w.Do(func(tx *Tx) {
		entity := tx.AddEntity(handle)
		if _, ok := handle.Entity(tx); !ok {
			t.Error("rebound entity did not resolve")
		}
		tx.RemoveEntity(entity)
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	_ = handle.Close()
}
