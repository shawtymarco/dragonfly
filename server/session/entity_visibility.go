package session

import (
	"math"
	"slices"

	"github.com/df-mc/dragonfly/server/world"
)

// A player-list runtime ID is not proof that AddPlayer has been sent. Keep
// actual actor presence separate so overlapping visibility callbacks cannot
// duplicate spawns, remove an already removed player, or update a missing actor.
func (s *Session) entityShown(e world.Entity) bool {
	if e.H() == s.ent {
		return true
	}
	s.entityMutex.RLock()
	_, shown := s.shownEntities[e.H()]
	s.entityMutex.RUnlock()
	return shown
}

// playerChunkReady must not query the Loader: ViewEntity can run while Load
// holds its mutex. World viewer membership is owned by tx and is installed only
// after ViewChunk has queued the chunk. Cache misses must also be resolved.
func (s *Session) playerChunkReady(e world.Entity, tx *world.Tx) bool {
	if tx == nil || s.ent == nil {
		return false
	}
	if _, present := s.ent.Entity(tx); !present {
		return false
	}
	if !slices.Contains(tx.Viewers(e.Position()), world.Viewer(s)) {
		return false
	}
	pos := e.Position()
	chunkPos := world.ChunkPos{int32(math.Floor(pos.X())) >> 4, int32(math.Floor(pos.Z())) >> 4}
	s.blobMu.Lock()
	_, pending := s.chunkTransactions[chunkPos]
	s.blobMu.Unlock()
	return !pending
}

// deferPlayerSpawn records only a handle, never a transaction-bound Player.
// Normal non-player entities and standalone viewers retain immediate spawning.
func (s *Session) deferPlayerSpawn(e world.Entity) bool {
	if s.chunkLoader == nil {
		return false
	}
	if _, player := e.(Controllable); !player {
		return false
	}
	owned, ok := e.(interface{ Tx() *world.Tx })
	if !ok || s.playerChunkReady(e, owned.Tx()) {
		return false
	}
	s.entityMutex.Lock()
	if s.pendingPlayers == nil {
		s.pendingPlayers = make(map[*world.EntityHandle]struct{})
	}
	s.pendingPlayers[e.H()] = struct{}{}
	s.entityMutex.Unlock()
	return true
}

// flushPendingPlayers runs on the controlling player's current world owner.
// The existing chunk sender drives retries; hiding, disconnecting and world
// changes discard pending entries, and no extra timer or world scan is needed.
func (s *Session) flushPendingPlayers(tx *world.Tx) {
	s.entityMutex.RLock()
	var pending []*world.EntityHandle
	for handle := range s.pendingPlayers {
		pending = append(pending, handle)
	}
	s.entityMutex.RUnlock()
	for _, handle := range pending {
		e, present := handle.Entity(tx)
		if !present || s.entityHidden(e) {
			s.entityMutex.Lock()
			delete(s.pendingPlayers, handle)
			s.entityMutex.Unlock()
			continue
		}
		if s.playerChunkReady(e, tx) {
			s.ViewEntity(e)
			s.ViewEntityItems(e)
			s.ViewEntityArmour(e)
		}
	}
}
