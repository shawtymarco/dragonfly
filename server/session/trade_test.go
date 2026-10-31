package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestTradeCostSplitsTheSameItemOverBothSides(t *testing.T) {
	gold := item.NewStack(item.GoldIngot{}, 1)
	c := tradeCost{a: gold.Grow(9), b: gold.Grow(4)}
	if !c.consume(gold, 12) || c.paid() {
		t.Fatal("12 of the 15 ingots should be taken, leaving the cost unpaid")
	}
	if !c.consume(gold, 3) || !c.paid() {
		t.Fatal("the last 3 ingots should pay the cost")
	}
	if c.consume(gold, 1) {
		t.Fatal("a paid cost should refuse more items")
	}
}

func TestTradeCostRefusesOtherItems(t *testing.T) {
	c := tradeCost{a: item.NewStack(item.GoldIngot{}, 2)}
	if c.consume(item.NewStack(item.IronIngot{}, 1), 1) {
		t.Fatal("an item outside the offer should not pay for it")
	}
	if c.consume(item.NewStack(item.GoldIngot{}, 1).WithCustomName("fake"), 1) {
		t.Fatal("a renamed item should not pay for it")
	}
}

func TestTradeOffersCarryNetworkIDsAndCounts(t *testing.T) {
	data, err := encodeTradeOffers([]TradeRecipe{
		{BuyA: item.NewStack(item.GoldIngot{}, 3), Sell: item.NewStack(item.Diamond{}, 1)},
		{BuyA: item.NewStack(item.GoldIngot{}, 1), BuyB: item.NewStack(item.IronIngot{}, 2), Sell: item.NewStack(item.Emerald{}, 4)},
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := nbt.UnmarshalEncoding(data, &m, nbt.NetworkLittleEndian); err != nil {
		t.Fatal(err)
	}
	offers, ok := m["Recipes"].([]any)
	if !ok || len(offers) != 2 {
		t.Fatalf("Recipes = %#v", m["Recipes"])
	}
	first, second := offers[0].(map[string]any), offers[1].(map[string]any)
	if first["netId"] != int32(1) || second["netId"] != int32(2) {
		t.Fatalf("net ids = %v, %v", first["netId"], second["netId"])
	}
	if first["buyCountA"] != int32(3) || second["buyCountB"] != int32(2) {
		t.Fatalf("counts = %v, %v", first["buyCountA"], second["buyCountB"])
	}
	if _, ok := first["buyB"]; ok {
		t.Fatal("an offer with one cost should not have buyB")
	}
	if _, ok := second["buyB"]; !ok {
		t.Fatal("an offer with two costs should have buyB")
	}
}
