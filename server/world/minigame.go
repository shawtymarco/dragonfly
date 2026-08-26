package world

import "sync"

// MinigameConfig contains opt-in performance switches intended for fixed-map
// minigame servers. All fields default to false so the normal Dragonfly
// behaviour is preserved unless a switch is explicitly enabled.
type MinigameConfig struct {
	// DisablePlayerSurvivalTicks skips survival-only work in the specialised
	// minigame player tick: hunger, air/drowning, suffocation, fire ticking,
	// turtle-shell water breathing and elytra durability. Void damage, item use,
	// breaking, HUD/debug flushing and server-controlled movement are retained.
	DisablePlayerSurvivalTicks bool
	// DisablePlayerEffectTicks skips EffectManager ticking for players. Enable
	// only when the minigame does not depend on timed vanilla effects.
	DisablePlayerEffectTicks bool
	// DisablePortalTicks skips per-tick portal contact checks for players and
	// generic entities. Portal interaction should not be used while enabled.
	DisablePortalTicks bool
	// DeduplicatePlayerCollisionTicks avoids repeating collision and on-ground
	// scans in Player.Tick for network-controlled players. PlayerAuthInput.Move
	// remains responsible for those checks.
	DeduplicatePlayerCollisionTicks bool
	// FastSetBlock makes Tx.SetBlock calls with nil SetOpts skip neighbour,
	// liquid-displacement and redstone updates. Explicit SetOpts always win.
	FastSetBlock bool
	// FastBreakBlock routes network block-break completion through the minigame
	// fast breaker, which keeps validation/events/visuals but skips vanilla
	// drops, XP, break handlers, exhaustion and tool durability.
	FastBreakBlock bool
	// DisableBlockTicks skips random block ticks and all TickerBlock/block-entity
	// ticks, avoiding the loaded-chunk/block-entity scan entirely.
	DisableBlockTicks bool
	// DisableScheduledBlockTicks skips and discards scheduled block updates and
	// prevents new scheduled block updates from being queued through Tx.
	DisableScheduledBlockTicks bool
	// ActiveEntityTicking ticks entities from chunks that are actually active
	// (have viewers), instead of scanning the entire world entity map each tick.
	ActiveEntityTicking bool
	// MovementDirtyChunkTracking updates entity/chunk membership from movement
	// dirty marks instead of recomputing every entity's ChunkPos every tick. A
	// periodic full audit remains as a correctness fallback.
	MovementDirtyChunkTracking bool
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

func (c MinigameConfig) specialisedPlayerTick() bool {
	return c.DisablePlayerSurvivalTicks || c.DisablePlayerEffectTicks || c.DisablePortalTicks || c.DeduplicatePlayerCollisionTicks
}

var minigameFastSetOpts = &SetOpts{
	DisableBlockUpdates:       true,
	DisableLiquidDisplacement: true,
	DisableRedstoneUpdates:    true,
}

// movementDirtyEntities is intentionally transient: entries are inserted only
// when an entity changes position and are consumed by the world tick. A sync.Map
// keeps marking safe for movement paths that may run outside the world ticker.
var movementDirtyEntities sync.Map // *EntityHandle -> struct{}

// MarkEntityMovementDirty marks an Entity for chunk-membership reconciliation.
// It is a no-op unless MovementDirtyChunkTracking is enabled for its World.
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
