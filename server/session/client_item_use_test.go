package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestClientItemUsePredictionMatchesOnlyExactHeldResult(t *testing.T) {
	s := &Session{}
	expected := item.NewStack(item.Bow{}, 1).Damage(1)
	end := s.beginClientPredictedItemUse(2, &expected)

	if !s.ClientPredictedItemUse() {
		t.Fatal("client item-use prediction was not active")
	}
	if !s.predictedHeldItemMatches(2, expected) {
		t.Fatal("exact predicted held result was not matched")
	}
	if s.predictedHeldItemMatches(3, expected) {
		t.Fatal("prediction matched a different hotbar slot")
	}
	if s.predictedHeldItemMatches(2, expected.Damage(1)) {
		t.Fatal("prediction suppressed an authoritative durability correction")
	}

	end()
	if s.ClientPredictedItemUse() {
		t.Fatal("client item-use prediction remained active")
	}
}

func TestClientItemUsePredictionRestoresNestedScope(t *testing.T) {
	s := &Session{}
	outerHeld := item.NewStack(item.Apple{}, 2)
	outerEnd := s.beginClientPredictedItemUse(1, &outerHeld)
	innerEnd := s.beginClientPredictedItemUse(-1, nil)

	if s.predictedHeldItemMatches(1, outerHeld) {
		t.Fatal("metadata-only prediction exposed an outer held result")
	}
	innerEnd()
	if !s.predictedHeldItemMatches(1, outerHeld) {
		t.Fatal("nested prediction did not restore its outer scope")
	}
	outerEnd()
}
