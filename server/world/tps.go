package world

import (
	"math"
	"time"
)

const tickBudget = time.Second / 20

func (w *World) recordTickDuration(d time.Duration) {
	if w == nil {
		return
	}
	n := w.tickCount.Add(1)
	w.tickDurations[(n-1)%20].Store(int64(d))
}

func (w *World) lastTickDuration() time.Duration {
	if w == nil {
		return 0
	}
	n := w.tickCount.Load()
	if n == 0 {
		return 0
	}
	return time.Duration(w.tickDurations[(n-1)%20].Load())
}

// TicksPerSecond returns the current world TPS, capped at 20.
func (w *World) TicksPerSecond() float64 {
	return tpsFromDuration(w.lastTickDuration())
}

// TickUsage returns the current tick time as a percentage of the 50ms budget.
func (w *World) TickUsage() float64 {
	return usageFromDuration(w.lastTickDuration())
}

// TicksPerSecondAverage returns the TPS averaged over the last 20 ticks.
func (w *World) TicksPerSecondAverage() float64 {
	return tpsFromDuration(w.averageTickDuration())
}

// TickUsageAverage returns the tick-budget usage averaged over the last 20 ticks.
func (w *World) TickUsageAverage() float64 {
	return usageFromDuration(w.averageTickDuration())
}

func (w *World) averageTickDuration() time.Duration {
	if w == nil {
		return 0
	}
	n := w.tickCount.Load()
	if n == 0 {
		return 0
	}
	limit := min(n, 20)
	var sum int64
	for i := n - limit; i < n; i++ {
		sum += w.tickDurations[i%20].Load()
	}
	return time.Duration(sum / int64(limit))
}

func tpsFromDuration(d time.Duration) float64 {
	if d <= 0 {
		return 20
	}
	tps := float64(time.Second) / float64(d)
	return math.Min(20, tps)
}

func usageFromDuration(d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(d) / float64(tickBudget) * 100
}

// LoadedChunkCount returns the number of loaded chunks as of the last world tick.
func (w *World) LoadedChunkCount() int {
	if w == nil {
		return 0
	}
	return int(w.loadedChunks.Load())
}

// EntityCount returns the number of loaded entities as of the last world tick.
func (w *World) EntityCount() int {
	if w == nil {
		return 0
	}
	return int(w.loadedEntities.Load())
}
