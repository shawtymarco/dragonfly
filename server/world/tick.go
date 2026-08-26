package world

import (
	"maps"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/internal/sliceutil"
)

// ticker implements World ticking methods.
type ticker struct {
	interval time.Duration
}

// tickLoop starts ticking the World 20 times every second, updating all
// entities, blocks and other features such as the time and weather of the
// world, as required.
func (t ticker) tickLoop(w *World) {
	tc := time.NewTicker(t.interval)
	defer tc.Stop()
	for {
		select {
		case <-tc.C:
			start := time.Now()
			<-w.exec(t.tick)
			w.recordTickDuration(time.Since(start))
		case <-w.closing:
			w.running.Done()
			return
		}
	}
}

// AdvanceTick advances the World by a single tick. It is generally only useful
// for Worlds created with Config.Synchronous set: other Worlds tick
// automatically 20 times per second. Synchronous Worlds tick loaded chunks
// even when no viewers are present.
func (w *World) AdvanceTick() {
	<-w.exec(ticker{}.tick)
}

// tick performs a tick on the World and updates the time, weather, blocks and
// entities that require updates.
func (t ticker) tick(tx *Tx) {
	viewers, loaders := tx.World().allViewers()
	w := tx.World()
	w.loadedChunks.Store(int64(len(w.chunks)))
	w.loadedEntities.Store(int64(len(w.entities)))

	w.set.Lock()
	if s := w.set.Spawn; s[1] > tx.Range()[1] && w.Dimension() == Overworld {
		w.set.Spawn[1] = tx.highestObstructingBlock(s[0], s[2]) + 1
	}
	if len(viewers) == 0 && w.set.CurrentTick != 0 && !w.conf.Synchronous {
		w.set.Unlock()
		return
	}
	if w.advance {
		w.set.CurrentTick++
		if w.set.TimeCycle {
			w.set.Time++
		}
		if w.set.WeatherCycle {
			w.advanceWeather()
		}
	}

	rain, thunder, tick, tim, cycle := w.set.Raining, w.set.Thundering && w.set.Raining, w.set.CurrentTick, int(w.set.Time), w.set.TimeCycle

	tryAdvanceDay := false
	if tx.w.set.RequiredSleepTicks > 0 {
		tx.w.set.RequiredSleepTicks--
		tryAdvanceDay = tx.w.set.RequiredSleepTicks <= 0
	}
	w.set.Unlock()

	if tryAdvanceDay {
		t.tryAdvanceDay(tx, cycle)
	}
	if tick%20 == 0 {
		for _, viewer := range viewers {
			if w.Dimension().TimeCycle() && cycle {
				viewer.ViewTime(tim)
			}
			if w.Dimension().WeatherCycle() {
				viewer.ViewWeather(rain, thunder)
			}
		}
	}
	if thunder {
		w.tickLightning(tx)
	}

	cfg := w.conf.Minigame
	t.tickEntities(tx, tick)
	if cfg.DisableScheduledBlockTicks {
		w.scheduledUpdates.currentTick = tick
		w.scheduledUpdates.ticks = w.scheduledUpdates.ticks[:0]
		clear(w.scheduledUpdates.furthestTicks)
	} else {
		w.scheduledUpdates.tick(tx, tick)
	}
	if !cfg.DisableBlockTicks {
		t.tickBlocksRandomly(tx, loaders, tick)
	}
	t.performNeighbourUpdates(tx)
	w.redstone.tick(tx, tick)
}

// performNeighbourUpdates performs all block updates that came as a result of a neighbouring block being changed.
func (t ticker) performNeighbourUpdates(tx *Tx) {
	updates := slices.Clone(tx.World().neighbourUpdates)
	clear(tx.World().neighbourUpdates)
	tx.World().neighbourUpdates = tx.World().neighbourUpdates[:0]

	for _, update := range updates {
		pos, changedNeighbour := update.pos, update.neighbour
		if ticker, ok := tx.Block(pos).(NeighbourUpdateTicker); ok {
			ticker.NeighbourUpdateTick(pos, changedNeighbour, tx)
		}
		if liquid, ok := tx.additionalLiquid(pos); ok {
			if ticker, ok := liquid.(NeighbourUpdateTicker); ok {
				ticker.NeighbourUpdateTick(pos, changedNeighbour, tx)
			}
		}
	}
}

// tickBlocksRandomly executes random block ticks in loaded chunks within range of loaders.
func (t ticker) tickBlocksRandomly(tx *Tx, loaders []*Loader, tick int64) {
	var (
		r             = int32(tx.World().tickRange())
		g             randUint4
		blockEntities []cube.Pos
		randomBlocks  []cube.Pos
	)
	if r == 0 {
		return
	}

	loaded := make([]ChunkPos, 0, len(loaders))
	if tx.World().conf.Synchronous {
		loaded = slices.Collect(maps.Keys(tx.World().chunks))
	} else {
		for _, loader := range loaders {
			loader.mu.RLock()
			pos := loader.pos
			loader.mu.RUnlock()
			loaded = append(loaded, pos)
		}
	}

	for pos, c := range tx.World().chunks {
		if !t.anyWithinDistance(pos, loaded, r) {
			continue
		}
		blockEntities = append(blockEntities, slices.Collect(maps.Keys(c.BlockEntities))...)

		cx, cz := int(pos[0]<<4), int(pos[1]<<4)
		for j := 0; j < tx.World().conf.RandomTickSpeed; j++ {
			x, y, z := g.uint4(tx.World().r), g.uint4(tx.World().r), g.uint4(tx.World().r)
			for i, sub := range c.Sub() {
				if sub.Empty() {
					continue
				}
				if rid := sub.Layers()[0].At(x, y, z); tx.World().conf.Blocks.RandomTickBlock(rid) {
					subY := (i + (tx.Range().Min() >> 4)) << 4
					randomBlocks = append(randomBlocks, cube.Pos{cx + int(x), subY + int(y), cz + int(z)})
					x, y, z = g.uint4(tx.World().r), g.uint4(tx.World().r), g.uint4(tx.World().r)
				}
			}
		}
	}

	for _, pos := range randomBlocks {
		b := tx.Block(pos)
		rb, ok := b.(RandomTicker)
		if !ok {
			continue
		}
		if f := tx.World().conf.RandomTickFilter; f != nil && !f(b) {
			continue
		}
		rb.RandomTick(pos, tx, tx.World().r)
	}
	for _, pos := range blockEntities {
		if tb, ok := tx.Block(pos).(TickerBlock); ok {
			tb.Tick(tick, pos, tx)
		}
	}
}

// anyWithinDistance checks if any of the ChunkPos loaded are within the distance r of the ChunkPos pos.
func (t ticker) anyWithinDistance(pos ChunkPos, loaded []ChunkPos, r int32) bool {
	for _, chunkPos := range loaded {
		xDiff, zDiff := chunkPos[0]-pos[0], chunkPos[1]-pos[1]
		if (xDiff*xDiff)+(zDiff*zDiff) <= r*r {
			return true
		}
	}
	return false
}

// tickEntities ticks entities using either the normal full-map scan or the
// opt-in minigame active/dirty paths.
func (t ticker) tickEntities(tx *Tx, tick int64) {
	w := tx.World()
	cfg := w.conf.Minigame
	if !cfg.ActiveEntityTicking && !cfg.MovementDirtyChunkTracking {
		for handle, lastPos := range w.entities {
			t.reconcileEntityChunk(tx, handle, lastPos)
			pos, ok := w.entities[handle]
			if !ok {
				continue
			}
			c, ok := w.chunks[pos]
			if !ok {
				continue
			}
			if w.conf.Synchronous || len(c.viewers) > 0 {
				t.tickEntity(tx, handle, tick, cfg)
			}
		}
		return
	}

	if cfg.MovementDirtyChunkTracking {
		for _, handle := range w.takeMovementDirty() {
			if lastPos, ok := w.entities[handle]; ok {
				t.reconcileEntityChunk(tx, handle, lastPos)
			}
		}
		// A low-frequency audit makes dirty tracking fail-safe for custom entity
		// code that mutates position without calling MarkEntityMovementDirty.
		if tick%20 == 0 {
			for handle, lastPos := range w.entities {
				t.reconcileEntityChunk(tx, handle, lastPos)
			}
		}
	} else {
		// ActiveEntityTicking alone still needs exact chunk membership each tick.
		for handle, lastPos := range w.entities {
			t.reconcileEntityChunk(tx, handle, lastPos)
		}
	}

	if cfg.ActiveEntityTicking && !w.conf.Synchronous {
		for _, c := range w.chunks {
			if len(c.viewers) == 0 {
				continue
			}
			for _, handle := range slices.Clone(c.Entities) {
				if handle == nil || handle.Closed() {
					continue
				}
				t.tickEntity(tx, handle, tick, cfg)
			}
		}
		return
	}

	// Dirty tracking without active-chunk ticking still scans handles to tick
	// them, but avoids recomputing their chunk position every tick.
	for handle, pos := range w.entities {
		c, ok := w.chunks[pos]
		if !ok {
			continue
		}
		if w.conf.Synchronous || len(c.viewers) > 0 {
			t.tickEntity(tx, handle, tick, cfg)
		}
	}
}

func (t ticker) tickEntity(tx *Tx, handle *EntityHandle, tick int64, cfg MinigameConfig) {
	if handle == nil || handle.Closed() {
		return
	}
	e := handle.mustEntity(tx)
	before := handle.data.Pos
	if cfg.specialisedEntityTick() {
		if mt, ok := e.(minigameTickerEntity); ok {
			mt.MinigameTick(tx, tick, cfg)
		} else if te, ok := e.(TickerEntity); ok {
			te.Tick(tx, tick)
		}
	} else if te, ok := e.(TickerEntity); ok {
		te.Tick(tx, tick)
	}
	if cfg.MovementDirtyChunkTracking && !handle.Closed() && !before.ApproxEqual(handle.data.Pos) {
		movementDirtyEntities.Store(handle, struct{}{})
	}
}

func (t ticker) reconcileEntityChunk(tx *Tx, handle *EntityHandle, lastPos ChunkPos) {
	if handle == nil || handle.Closed() {
		return
	}
	w := tx.World()
	chunkPos := chunkPosFromVec3(handle.data.Pos)
	if lastPos == chunkPos {
		return
	}
	c, ok := w.chunks[chunkPos]
	if !ok {
		return
	}

	e := handle.mustEntity(tx)
	w.entities[handle] = chunkPos
	if slices.Index(c.Entities, handle) == -1 {
		c.Entities = append(c.Entities, handle)
	}

	var viewers []Viewer
	if old, ok := w.chunks[lastPos]; ok {
		old.Entities = sliceutil.DeleteVal(old.Entities, handle)
		viewers = old.viewers
	}
	for _, viewer := range viewers {
		if slices.Index(c.viewers, viewer) == -1 {
			viewer.HideEntity(e)
		}
	}
	for _, viewer := range c.viewers {
		if slices.Index(viewers, viewer) == -1 {
			showEntity(e, viewer)
		}
	}
}

// randUint4 is a structure used to generate random uint4s.
type randUint4 struct {
	x uint64
	n uint8
}

// uint4 returns a random uint4.
func (g *randUint4) uint4(r *rand.Rand) uint8 {
	if g.n == 0 {
		g.x = r.Uint64()
		g.n = 16
	}
	val := g.x & 0b1111
	g.x >>= 4
	g.n--
	return uint8(val)
}

// scheduledTickQueue implements a queue for scheduled block updates. Scheduled
// block updates are both position and block type specific.
type scheduledTickQueue struct {
	ticks         []scheduledTick
	furthestTicks map[scheduledTickIndex]int64
	currentTick   int64
}

type scheduledTick struct {
	pos   cube.Pos
	b     Block
	bhash uint64
	t     int64
}

type scheduledTickIndex struct {
	pos  cube.Pos
	hash uint64
}

// newScheduledTickQueue creates a queue for scheduled block ticks.
func newScheduledTickQueue(tick int64) *scheduledTickQueue {
	return &scheduledTickQueue{furthestTicks: make(map[scheduledTickIndex]int64), currentTick: tick}
}

// tick processes scheduled ticks, calling ScheduledTicker.ScheduledTick for any
// block update that is scheduled for the tick passed, and removing it from the queue.
func (queue *scheduledTickQueue) tick(tx *Tx, tick int64) {
	queue.currentTick = tick
	w := tx.World()
	for _, t := range queue.ticks {
		if t.t > tick {
			continue
		}
		b := tx.Block(t.pos)
		if ticker, ok := b.(ScheduledTicker); ok && w.conf.Blocks.BlockHash(b) == t.bhash {
			ticker.ScheduledTick(t.pos, tx, w.r)
		} else if liquid, ok := tx.additionalLiquid(t.pos); ok && w.conf.Blocks.BlockHash(liquid) == t.bhash {
			if ticker, ok := liquid.(ScheduledTicker); ok {
				ticker.ScheduledTick(t.pos, tx, w.r)
			}
		}
	}
	queue.ticks = slices.DeleteFunc(queue.ticks, func(t scheduledTick) bool { return t.t <= tick })
	maps.DeleteFunc(queue.furthestTicks, func(_ scheduledTickIndex, t int64) bool { return t <= tick })
}

// schedule schedules a block update at the position passed for the block type passed after a specific delay.
func (queue *scheduledTickQueue) schedule(br BlockRegistry, pos cube.Pos, b Block, delay time.Duration) {
	resTick := queue.currentTick + int64(max(delay/(time.Second/20), 1))
	index := scheduledTickIndex{pos: pos, hash: br.BlockHash(b)}
	if t, ok := queue.furthestTicks[index]; ok && t >= resTick && t > queue.currentTick {
		return
	}
	queue.furthestTicks[index] = resTick
	queue.ticks = append(queue.ticks, scheduledTick{pos: pos, t: resTick, b: b, bhash: index.hash})
}

// fromChunk returns all scheduled ticks positioned within a ChunkPos.
func (queue *scheduledTickQueue) fromChunk(pos ChunkPos) []scheduledTick {
	m := make([]scheduledTick, 0, 8)
	for _, t := range queue.ticks {
		if pos == chunkPosFromBlockPos(t.pos) {
			m = append(m, t)
		}
	}
	return m
}

// removeChunk removes all scheduled ticks positioned within a ChunkPos.
func (queue *scheduledTickQueue) removeChunk(pos ChunkPos) {
	queue.ticks = slices.DeleteFunc(queue.ticks, func(tick scheduledTick) bool {
		return chunkPosFromBlockPos(tick.pos) == pos
	})
	maps.DeleteFunc(queue.furthestTicks, func(index scheduledTickIndex, _ int64) bool {
		return chunkPosFromBlockPos(index.pos) == pos
	})
}

// add adds a slice of scheduled ticks to the queue. It assumes no duplicate ticks are present in the slice.
func (queue *scheduledTickQueue) add(ticks []scheduledTick) {
	queue.ticks = append(queue.ticks, ticks...)
	for _, t := range ticks {
		index := scheduledTickIndex{pos: t.pos, hash: t.bhash}
		if existing, ok := queue.furthestTicks[index]; ok {
			queue.furthestTicks[index] = max(existing, t.t)
		} else {
			queue.furthestTicks[index] = t.t
		}
	}
}
