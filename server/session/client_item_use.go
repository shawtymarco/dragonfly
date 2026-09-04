package session

import "github.com/df-mc/dragonfly/server/item"

type clientItemUsePrediction struct {
	slot      int
	held      item.Stack
	matchHeld bool
}

// beginClientPredictedItemUse marks item state changes made while processing a
// client use/release transaction. Bedrock has already applied these changes to
// its local player. Echoing matching held-item or using-state updates back to
// that player may arrive during a later use cycle and reset its animation.
func (s *Session) beginClientPredictedItemUse(slot int, held *item.Stack) func() {
	prediction := &clientItemUsePrediction{slot: slot}
	if held != nil {
		prediction.held = *held
		prediction.matchHeld = true
	}
	previous := s.clientItemUsePrediction.Swap(prediction)
	return func() {
		s.clientItemUsePrediction.CompareAndSwap(prediction, previous)
	}
}

// ClientPredictedItemUse reports whether the current item-use state change was
// initiated and already predicted by this session's Bedrock client.
func (s *Session) ClientPredictedItemUse() bool {
	return s.clientItemUsePrediction.Load() != nil
}

func (s *Session) predictedHeldItemMatches(slot int, after item.Stack) bool {
	prediction := s.clientItemUsePrediction.Load()
	return prediction != nil && prediction.matchHeld && prediction.slot == slot &&
		(after.Equal(prediction.held) || interactionPredictionCompatible(after, prediction.held))
}

func (s *Session) markClientItemReleased() {
	s.waitStartUsingItemClear = true
}

// clientStartUsingItemEdge reports a fresh false-to-true StartUsingItem edge.
// A release arms a guard which must first observe an input frame with the flag
// clear. This prevents a stale flag batched with ReleaseItem from immediately
// starting another use cycle after the physical button was released.
func (s *Session) clientStartUsingItemEdge(active bool) (rising, releaseGuard bool) {
	releaseGuard = s.waitStartUsingItemClear
	if releaseGuard && !active {
		s.waitStartUsingItemClear = false
		releaseGuard = false
	}
	rising = active && !s.startUsingItemInput && !releaseGuard
	s.startUsingItemInput = active
	return rising, releaseGuard
}
