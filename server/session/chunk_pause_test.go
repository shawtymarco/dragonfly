package session

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl64"
)

type chunkPauseViewer struct {
	world.NopViewer
	chunks chan struct{}
}

func (v chunkPauseViewer) ViewChunk(world.ChunkPos, world.Dimension, map[cube.Pos]world.Block, *chunk.Chunk) {
	select {
	case v.chunks <- struct{}{}:
	default:
	}
}

type chunkPausePlayer struct{ Controllable }

func (chunkPausePlayer) Position() mgl64.Vec3 { return mgl64.Vec3{0, 64, 0} }

func TestInitialChunksWaitForApplicationResume(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	defer w.Close()
	v := chunkPauseViewer{chunks: make(chan struct{}, 4)}
	s := &Session{chunkRadius: 2, packets: make(chan outboundMessage, 8)}
	defer func() { _ = w.Do(func(tx *world.Tx) { s.chunkLoader.Close(tx) }).Wait(t.Context()) }()
	if err := w.Do(func(tx *world.Tx) {
		s.chunkLoader = world.NewLoader(2, w, v)
		s.SetChunkLoadingPaused(true)
		s.sendChunks(tx, chunkPausePlayer{})
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := w.Do(func(tx *world.Tx) {
		select {
		case <-v.chunks:
			t.Fatal("provisional chunk escaped during authorization")
		default:
		}
		s.SetChunkLoadingPaused(false)
		s.sendChunks(tx, chunkPausePlayer{})
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-v.chunks:
	case <-time.After(3 * time.Second):
		t.Fatal("initial chunks did not resume")
	}
}
