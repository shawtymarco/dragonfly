package player

import (
	"testing"

	"github.com/df-mc/dragonfly/server/entity"
)

func TestUpdateFallStatePreservesAirborneApexAcrossVerticalKnockback(t *testing.T) {
	p := &Player{playerData: &playerData{fallDistance: 8, mc: &entity.MovementComputer{}}}

	p.updateFallState(2)
	if got, want := p.FallDistance(), 6.0; got != want {
		t.Fatalf("fall distance after upward knockback: got %v, want %v", got, want)
	}

	p.updateFallState(0)
	if got, want := p.FallDistance(), 6.0; got != want {
		t.Fatalf("fall distance after a stationary airborne tick: got %v, want %v", got, want)
	}

	p.updateFallState(-3)
	if got, want := p.FallDistance(), 9.0; got != want {
		t.Fatalf("fall distance after descent resumes: got %v, want %v", got, want)
	}
}

func TestUpdateFallStateResetsAfterAscendingBeyondApex(t *testing.T) {
	p := &Player{playerData: &playerData{fallDistance: 3, mc: &entity.MovementComputer{}}}
	p.updateFallState(3)

	if got := p.FallDistance(); got != 0 {
		t.Fatalf("fall distance after ascending beyond the measured apex: got %v, want 0", got)
	}
}
