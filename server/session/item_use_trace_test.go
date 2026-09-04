package session

import (
	"testing"
	"time"
)

func TestItemUseTraceLimiterBoundsAndReportsDrops(t *testing.T) {
	var limiter itemUseTraceLimiter
	now := time.Unix(100, 0)
	for index := 0; index < itemUseTraceEventsPerSecond; index++ {
		allowed, dropped := limiter.allow(now)
		if !allowed || dropped != 0 {
			t.Fatalf("event %d allowed=%v dropped=%d", index, allowed, dropped)
		}
	}
	if allowed, _ := limiter.allow(now); allowed {
		t.Fatal("event beyond the trace limit was allowed")
	}
	if allowed, _ := limiter.allow(now); allowed {
		t.Fatal("second event beyond the trace limit was allowed")
	}
	allowed, dropped := limiter.allow(now.Add(time.Second))
	if !allowed || dropped != 2 {
		t.Fatalf("next window allowed=%v dropped=%d", allowed, dropped)
	}
}
