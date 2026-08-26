package world

import (
	"sync"

	"github.com/df-mc/dragonfly/server/block/cube"
)

// MinigameConfig contains opt-in performance switches intended for fixed-map
// minigame servers. All fields default to false so the normal Dragonfly
// behaviour is preserved unless a switch is explicitly enabled.
type MinigameConfig struct {
	DisablePlayerSurvivalTicks bool
	DisablePlayerEffectTicks bool
	DisablePortalTicks bool
	DeduplicatePlayerCollisionTicks bool
	FastSetBlock bool
	FastBreakBlock bool
	DisableBlockTicks bool
	DisableScheduledBlockTicks bool
	ActiveEntityTicking bool
	MovementDirtyChunkTracking bool
}

// SetMinigameConfig replaces the world's minigame optimisation switches. It is
// intended to be called while constructing the server, before players enter the
// world.
func (w *World) SetMinigameConfig(conf MinigameConfig) {
	if w != nil {
		w.conf.Minigame = conf
	}
}

// MinigameConfig returns the minigame optimisation switches of the World.
func (w *World) MinigameConfig() MinigameConfig {
	if w == nil {
		return MinigameConfig{}
	}
	return w.conf.Minigame
}

// MinigameConfig returns the minigame optimisation switches of the transaction's World.
func (tx *Tx) MinigameConfig() MinigameConfig {
	if tx == nil {
		return MinigameConfig{}
	}
	return tx.World().MinigameConfig()
}

func (c MinigameConfig) specialisedEntityTick() bool {
	return c.DisablePlayerSurvivalTicks || c.DisablePlayerEffectTicks || c.DisablePortalTicks || c.DeduplicatePlayerCollisionTicks
}

var minigameFastSetOpts = &SetOpts{
	DisableBlockUpdates:       true,
	DisableLiquidDisplacement: true,
	DisableRedstoneUpdates:    true,
}

// FastSetBlock writes a block using the minigame fast mutation path when enabled.
// When FastSetBlock is false it has identical semantics to SetBlock(pos, b, nil).
func (tx *Tx) FastSetBlock(pos cube.Pos, b Block) {
	if tx.MinigameConfig().FastSetBlock {
		tx.setBlock(pos, b, minigameFastSetOpts)
		return
	}
	tx.setBlock(pos, b, nil)
}

type minigameTickerEntity interface {
	MinigameTick(tx *Tx, current int64, conf MinigameConfig)
}

var movementDirtyEntities sync.Map // *EntityHandle -> struct{}

// MarkEntityMovementDirty marks an Entity for chunk-membership reconciliation.
func MarkEntityMovementDirty(e Entity) {
	if e == nil || e.H() == nil {
		return
	}
	h := e.H()
	w := h.w
	if w == nil || w == closeWorld || !w.conf.Minigame.MovementDirtyChunkTracking {
		return
	}
	movementDirtyEntities.Store(h, struct{}{})
}

func (w *World) takeMovementDirty() []*EntityHandle {
	if w == nil || !w.conf.Minigame.MovementDirtyChunkTracking {
		return nil
	}
	var dirty []*EntityHandle
	movementDirtyEntities.Range(func(key, _ any) bool {
		h, ok := key.(*EntityHandle)
		if !ok || h == nil || h.Closed() || h.w == closeWorld {
			movementDirtyEntities.Delete(key)
			return true
		}
		if h.w == w {
			dirty = append(dirty, h)
			movementDirtyEntities.Delete(key)
		}
		return true
	})
	return dirty
}
