package world

import (
	"sync"
	"testing"
)

type cleanupViewer struct {
	NopViewer
	hidden []*EntityHandle
}

func (v *cleanupViewer) HideEntity(e Entity) { v.hidden = append(v.hidden, e.H()) }

// A player can leave the destination before the source owner handles its
// queued ChangeWorld cleanup. Close clears Loader.viewer in that interval.
func TestLoaderSourceCleanupAfterDestinationClose(t *testing.T) {
	source, destination := Config{}.New(), Config{}.New()
	defer source.Close()
	defer destination.Close()
	viewer := &cleanupViewer{}
	handle := NewEntity(taskTestEntityType{}, taskTestEntityConfig{})
	var loader *Loader
	if err := source.Do(func(tx *Tx) {
		tx.AddEntity(handle)
		loader = NewLoader(1, source, viewer)
		col := tx.chunk(ChunkPos{})
		source.addViewer(tx, col, loader)
		loader.loaded[ChunkPos{}] = col
	}).Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
	blocked, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	source.Do(func(*Tx) { close(blocked); <-release })
	<-blocked
	if err := destination.Do(func(tx *Tx) {
		loader.ChangeWorld(tx, destination)
		loader.Close(tx)
	}).Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
	releaseOnce.Do(func() { close(release) })
	if err := source.Do(func(tx *Tx) {
		col := tx.chunk(ChunkPos{})
		if len(col.loaders) != 0 || len(col.viewers) != 0 {
			t.Error("source retained the departing viewer")
		}
		if len(viewer.hidden) != 1 || viewer.hidden[0] != handle {
			t.Error("source entities were not hidden exactly once")
		}
	}).Wait(testContext(t)); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveViewerUsesChunkRegistrationAndIsIdempotent(t *testing.T) {
	w := Config{Synchronous: true}.New()
	defer w.Close()
	registered, replacement, neighbour := &cleanupViewer{}, &cleanupViewer{}, &cleanupViewer{}
	loader := &Loader{viewer: registered}
	other := &Loader{viewer: neighbour}
	w.Do(func(tx *Tx) {
		tx.AddEntity(NewEntity(taskTestEntityType{}, taskTestEntityConfig{}))
		col := tx.chunk(ChunkPos{})
		w.addViewer(tx, col, loader)
		w.addViewer(tx, col, other)
		loader.viewer = replacement
		w.removeViewer(tx, ChunkPos{}, loader)
		w.removeViewer(tx, ChunkPos{}, loader)
		if len(registered.hidden) != 1 || len(replacement.hidden) != 0 || len(neighbour.hidden) != 0 {
			t.Error("cleanup used a changed viewer or hid an unregistered viewer")
		}
		if len(col.loaders) != 1 || col.loaders[0] != other || len(col.viewers) != 1 || col.viewers[0] != neighbour {
			t.Error("cleanup removed another loader's registration")
		}
		w.removeViewer(tx, ChunkPos{}, other)
	}).Wait(testContext(t))
}
