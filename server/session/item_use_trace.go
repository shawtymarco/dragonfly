package session

import (
	"sync"
	"time"
)

const itemUseTraceEventsPerSecond = 200

type itemUseTraceLimiter struct {
	mu          sync.Mutex
	windowStart time.Time
	events      int
	dropped     int
}

func (l *itemUseTraceLimiter) allow(now time.Time) (allowed bool, dropped int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.windowStart.IsZero() || now.Sub(l.windowStart) >= time.Second {
		dropped = l.dropped
		l.windowStart = now
		l.events = 0
		l.dropped = 0
	}
	if l.events >= itemUseTraceEventsPerSecond {
		l.dropped++
		return false, dropped
	}
	l.events++
	return true, dropped
}

func (s *Session) traceItemUse(c Controllable, event string, attributes ...any) {
	if !s.conf.ItemUseTrace || s == Nop {
		return
	}
	sequence := s.itemUseTraceSequence.Add(1)
	allowed, dropped := s.itemUseTraceLimiter.allow(time.Now())
	if !allowed {
		return
	}
	if dropped != 0 {
		s.conf.Log.Info("item use trace",
			"event", "rate_limit_summary",
			"dropped", dropped,
			"ping_ms", s.Latency().Milliseconds(),
			"sequence", sequence)
	}

	held, _ := c.HeldItems()
	heldName := "minecraft:air"
	if !held.Empty() {
		heldName, _ = held.Item().EncodeItem()
	}
	base := []any{
		"event", event,
		"sequence", sequence,
		"ping_ms", s.Latency().Milliseconds(),
		"using", c.UsingItem(),
		"held_item", heldName,
		"held_count", held.Count(),
		"held_durability", held.Durability(),
	}
	s.conf.Log.Info("item use trace", append(base, attributes...)...)
}
