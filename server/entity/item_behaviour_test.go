package entity

import (
	"testing"
	"time"
)

func TestItemBehaviourDefaultPickupDelay(t *testing.T) {
	t.Parallel()

	behaviour := (ItemBehaviourConfig{}).New()
	if got, want := behaviour.pickupDelay, time.Second/2; got != want {
		t.Fatalf("unexpected default pickup delay: got %v, want %v", got, want)
	}
}

func TestItemBehaviourExplicitZeroPickupDelay(t *testing.T) {
	t.Parallel()

	behaviour := (ItemBehaviourConfig{pickupDelaySet: true}).New()
	if got := behaviour.pickupDelay; got != 0 {
		t.Fatalf("explicit zero pickup delay was replaced: got %v", got)
	}
}
