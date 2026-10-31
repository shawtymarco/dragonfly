package entity

import (
	"testing"
	"time"

	"github.com/go-gl/mathgl/mgl64"
)

func TestItemBehaviourDefaultPickupDelay(t *testing.T) {
	t.Parallel()

	behaviour := (ItemBehaviourConfig{}).New()
	if got, want := behaviour.pickupDelay, time.Second/2; got != want {
		t.Fatalf("unexpected default pickup delay: got %v, want %v", got, want)
	}
}

func TestInPickupRange(t *testing.T) {
	t.Parallel()

	item := mgl64.Vec3{10, 5, 10}
	for _, c := range []struct {
		pos  mgl64.Vec3
		want bool
	}{
		{mgl64.Vec3{10, 5, 10}, true},
		{mgl64.Vec3{11.25, 5, 10}, true},
		{mgl64.Vec3{11.3, 5, 10}, false},
		{mgl64.Vec3{11, 5, 11}, false},
		{mgl64.Vec3{10, 3.5, 10}, false},
	} {
		if got := inPickupRange(c.pos, item); got != c.want {
			t.Fatalf("inPickupRange(%v) = %v", c.pos, got)
		}
	}
}

func TestItemBehaviourExplicitZeroPickupDelay(t *testing.T) {
	t.Parallel()

	behaviour := (ItemBehaviourConfig{pickupDelaySet: true}).New()
	if got := behaviour.pickupDelay; got != 0 {
		t.Fatalf("explicit zero pickup delay was replaced: got %v", got)
	}
}
