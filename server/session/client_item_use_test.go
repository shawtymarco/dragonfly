package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
)

func TestClientItemUsePredictionMatchesSafeHeldResult(t *testing.T) {
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
	if !s.predictedHeldItemMatches(2, expected.Damage(1)) {
		t.Fatal("prediction did not tolerate client-predicted durability")
	}
	if s.predictedHeldItemMatches(2, item.NewStack(item.Apple{}, 1)) {
		t.Fatal("prediction matched a different item")
	}
	if s.predictedHeldItemMatches(2, expected.WithValue("variant", 1)) {
		t.Fatal("prediction matched different custom data")
	}

	end()
	if s.ClientPredictedItemUse() {
		t.Fatal("client item-use prediction remained active")
	}
}

func TestClientItemUsePredictionMatchesPredictedCount(t *testing.T) {
	s := &Session{}
	expected := item.NewStack(item.GoldenApple{}, 2)
	end := s.beginClientPredictedItemUse(4, &expected)
	defer end()

	if !s.predictedHeldItemMatches(4, expected.Grow(-1)) {
		t.Fatal("prediction did not tolerate a client-predicted stack count")
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

func TestClientStartUsingItemRequiresClearFrameAfterRelease(t *testing.T) {
	s := &Session{}
	if rising, guarded := s.clientStartUsingItemEdge(true); !rising || guarded {
		t.Fatalf("initial edge rising=%v guarded=%v", rising, guarded)
	}
	if rising, _ := s.clientStartUsingItemEdge(true); rising {
		t.Fatal("held input produced a second rising edge")
	}

	s.markClientItemReleased()
	if rising, guarded := s.clientStartUsingItemEdge(true); rising || !guarded {
		t.Fatalf("stale release flag rising=%v guarded=%v", rising, guarded)
	}
	if rising, guarded := s.clientStartUsingItemEdge(false); rising || guarded {
		t.Fatalf("clear frame rising=%v guarded=%v", rising, guarded)
	}
	if rising, guarded := s.clientStartUsingItemEdge(true); !rising || guarded {
		t.Fatalf("fresh post-clear edge rising=%v guarded=%v", rising, guarded)
	}
}
