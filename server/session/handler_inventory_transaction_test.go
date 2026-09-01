package session

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/item"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestRepeatedRightClickSignature(t *testing.T) {
	h := &InventoryTransactionHandler{}
	data := &protocol.UseItemTransactionData{
		BlockFace:       1,
		BlockPosition:   protocol.BlockPos{1, 2, 3},
		Position:        mgl32.Vec3{4, 5, 6},
		ClickedPosition: mgl32.Vec3{0.5, 1, 0.5},
	}
	now := time.Unix(100, 0)
	if h.repeatedRightClick(data, now) {
		t.Fatal("first click was classified as a repeat")
	}
	if !h.repeatedRightClick(data, now.Add(50*time.Millisecond)) {
		t.Fatal("identical click inside 100 ms was not filtered")
	}
	data.BlockPosition[0]++
	if h.repeatedRightClick(data, now.Add(75*time.Millisecond)) {
		t.Fatal("click on a different block was filtered")
	}
	if h.repeatedRightClick(data, now.Add(200*time.Millisecond)) {
		t.Fatal("click outside the 100 ms window was filtered")
	}
}

func TestSimulationTickAirUsePolicy(t *testing.T) {
	tests := []struct {
		name  string
		stack item.Stack
		using bool
		want  bool
	}{
		{name: "plain control", stack: item.NewStack(item.Arrow{}, 1), want: true},
		{name: "bow release", stack: item.NewStack(item.Bow{}, 1), using: true, want: true},
		{name: "crossbow charging", stack: item.NewStack(item.Crossbow{}, 1), using: true, want: false},
		{name: "crossbow charged", stack: item.NewStack(item.Crossbow{}, 1), want: true},
		{name: "food consumption", stack: item.NewStack(item.Apple{}, 1), using: true, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := skipAirSimulationTick(test.stack, test.using); got != test.want {
				t.Fatalf("skipAirSimulationTick() = %v, want %v", got, test.want)
			}
		})
	}
}
