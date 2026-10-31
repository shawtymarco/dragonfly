package session

import (
	"fmt"
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// TradeRecipe is one offer of a trade window: BuyA and, if it isn't empty, BuyB are exchanged for Sell. The
// counts of the stacks are the counts of a single trade.
type TradeRecipe struct {
	BuyA, BuyB, Sell item.Stack
}

// tradeWindow holds the offers of the trade window currently opened. An offer is identified by its index + 1.
type tradeWindow struct {
	recipes []TradeRecipe
	holder  uint64
}

// tradeCost holds what is left to be paid for the trade of the request being handled.
type tradeCost struct {
	a, b item.Stack
}

// consume takes the items passed off the cost. It returns false if they are not a part of it. Both sides of an
// offer may be the same item, and the client then splits them over the two slots however it likes.
func (c *tradeCost) consume(i item.Stack, count int) bool {
	for _, side := range []*item.Stack{&c.a, &c.b} {
		if n := min(count, side.Count()); n > 0 && side.Comparable(i) {
			*side = side.Grow(-n)
			count -= n
		}
	}
	return count == 0
}

// paid checks if the full cost was consumed.
func (c *tradeCost) paid() bool {
	return c.a.Count() == 0 && c.b.Count() == 0
}

// OpenTrade opens a trade window offering the recipes passed, held by the entity passed, which the client must
// know about. It closes the container currently opened.
func (s *Session) OpenTrade(holder world.Entity, title string, recipes []TradeRecipe, pos cube.Pos, tx *world.Tx) {
	offers, err := encodeTradeOffers(recipes)
	if err != nil {
		s.conf.Log.Error("open trade: encode offers: " + err.Error())
		return
	}
	s.closeCurrentContainer(tx, false)

	nextID := s.nextWindowID()
	id := s.entityRuntimeID(holder)
	s.containerOpened.Store(true)
	s.openedVirtual.Store(true)
	s.openedWindow.Store(inventory.New(1, nil))
	s.openedPos.Store(&pos)
	s.openedContainerID.Store(uint32(protocol.ContainerTypeTrade))
	s.openedTrade.Store(&tradeWindow{recipes: recipes, holder: id})

	// Only this client is told who the holder trades with, so that players may trade with it at the same time.
	s.writeTradeTarget(id, selfEntityRuntimeID)
	s.writePacket(&packet.UpdateTrade{
		WindowID:          nextID,
		WindowType:        protocol.ContainerTypeTrade,
		VillagerUniqueID:  int64(id),
		EntityUniqueID:    -1,
		DisplayName:       title,
		NewTradeUI:        true,
		DemandBasedPrices: true,
		SerialisedOffers:  offers,
	})
}

// writeTradeTarget sends the entity that the holder passed is trading with.
func (s *Session) writeTradeTarget(holder uint64, target int64) {
	s.writePacket(&packet.SetActorData{
		EntityRuntimeID: holder,
		EntityMetadata:  protocol.EntityMetadata{protocol.EntityDataKeyTradeTarget: target},
	})
}

// encodeTradeOffers encodes recipes to the network NBT the client expects in the UpdateTrade packet.
func encodeTradeOffers(recipes []TradeRecipe) ([]byte, error) {
	offers := make([]map[string]any, len(recipes))
	for i, r := range recipes {
		offer := map[string]any{
			"buyA":             item.WriteNBT(r.BuyA, true),
			"buyCountA":        int32(r.BuyA.Count()),
			"buyCountB":        int32(r.BuyB.Count()),
			"sell":             item.WriteNBT(r.Sell, true),
			"uses":             int32(0),
			"maxUses":          int32(math.MaxInt32),
			"rewardExp":        byte(0),
			"demand":           int32(0),
			"tier":             int32(0),
			"traderExp":        int32(0),
			"priceMultiplierA": float32(0),
			"priceMultiplierB": float32(0),
			"netId":            int32(i + 1),
		}
		if !r.BuyB.Empty() {
			offer["buyB"] = item.WriteNBT(r.BuyB, true)
		}
		offers[i] = offer
	}
	return nbt.MarshalEncoding(map[string]any{
		"Recipes":             offers,
		"TierExpRequirements": []map[string]any{},
	}, nbt.NetworkLittleEndian)
}

// handleTrade starts the trade of the recipe with the network ID passed. The items are paid for by the consume
// actions that the client sends along with it, which are checked once the request ends.
func (h *ItemStackRequestHandler) handleTrade(t *tradeWindow, id uint32, trades byte, s *Session, tx *world.Tx) error {
	if id == 0 || int(id) > len(t.recipes) {
		return fmt.Errorf("no trade recipe with network id %v", id)
	}
	if trades < 1 {
		return fmt.Errorf("times traded must be at least 1")
	}
	if h.trade != nil {
		return fmt.Errorf("a trade is already in progress in this request")
	}
	r, n := t.recipes[id-1], int(trades)-1
	result := r.Sell.Grow(r.Sell.Count() * n)
	if result.Count() > math.MaxUint8 {
		return fmt.Errorf("trade result of %v items does not fit a response", result.Count())
	}
	h.trade = &tradeCost{a: r.BuyA.Grow(r.BuyA.Count() * n), b: r.BuyB.Grow(r.BuyB.Count() * n)}
	return h.createResults(s, tx, result)
}

// handleTradeConsume handles a Consume stack request action of a trade.
func (h *ItemStackRequestHandler) handleTradeConsume(a *protocol.ConsumeStackRequestAction, s *Session, tx *world.Tx) error {
	if err := h.verifySlot(a.Source, s, tx); err != nil {
		return fmt.Errorf("source slot out of sync: %w", err)
	}
	i, _ := h.itemInSlot(a.Source, s, tx)
	if i.Count() < int(a.Count) {
		return fmt.Errorf("client tried consuming %v items, but only %v present", a.Count, i.Count())
	}
	if !h.trade.consume(i, int(a.Count)) {
		return fmt.Errorf("client tried paying a trade with %v", i)
	}
	h.setItemInSlot(a.Source, i.Grow(-int(a.Count)), s, tx)
	return nil
}
