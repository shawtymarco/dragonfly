package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type traceOnlyStartUsingControllable struct {
	Controllable
	useCalls int
}

func (c *traceOnlyStartUsingControllable) HeldItems() (item.Stack, item.Stack) {
	return item.NewStack(item.Bow{}, 1), item.Stack{}
}

func (*traceOnlyStartUsingControllable) UsingItem() bool { return false }
func (*traceOnlyStartUsingControllable) Sneaking() bool  { return false }

func (c *traceOnlyStartUsingControllable) UseItem() { c.useCalls++ }

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
	expected := item.NewStack(item.Snowball{}, 2)
	end := s.beginClientPredictedItemUse(4, &expected)
	defer end()

	if !s.predictedHeldItemMatches(4, expected.Grow(-1)) {
		t.Fatal("prediction did not tolerate a client-predicted stack count")
	}
}

func TestConsumableTransactionsKeepUpstreamInventoryEcho(t *testing.T) {
	slot := uint32(2)
	s := &Session{heldSlot: &slot, inv: inventory.New(36, nil)}
	held := item.NewStack(item.GoldenApple{}, 2)
	_ = s.inv.SetItem(int(slot), held)
	for _, input := range []*item.Stack{&held, nil} {
		end := s.beginClientPredictedItemUse(int(slot), input)
		if s.ClientPredictedItemUse() || s.predictedHeldItemMatches(int(slot), held.Grow(-1)) {
			t.Fatal("consumable transaction suppressed upstream inventory synchronisation")
		}
		end()
	}
}

func TestClientItemUsePredictionRestoresNestedScope(t *testing.T) {
	s := &Session{}
	outerHeld := item.NewStack(item.Snowball{}, 2)
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

func TestStartUsingItemInputIsTraceOnly(t *testing.T) {
	flags := protocol.NewInputFlags(packet.InputFlagCount)
	flags.Set(packet.InputFlagStartUsingItem)
	c := &traceOnlyStartUsingControllable{}
	(PlayerAuthInputHandler{}).handleInputFlags(flags, &Session{}, c)
	if c.useCalls != 0 {
		t.Fatalf("StartUsingItem input synthesized %d item uses", c.useCalls)
	}
}
