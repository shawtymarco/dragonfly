package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type doorSource map[cube.Pos]world.Block

func (s doorSource) Block(pos cube.Pos) world.Block {
	if b, ok := s[pos]; ok {
		return b
	}
	return Air{}
}

// TestDoorHalvesShareHinge checks that an open door's lower half takes its hinge
// from the upper half, as the client does, when the two halves disagree.
func TestDoorHalvesShareHinge(t *testing.T) {
	lower, upper := cube.Pos{0, 0, 0}, cube.Pos{0, 1, 0}
	src := doorSource{
		lower: IronDoor{Facing: cube.North, Open: true, Right: false},
		upper: IronDoor{Facing: cube.North, Open: false, Right: true, Top: true},
	}
	want := IronDoor{Facing: cube.North, Open: true, Right: true}.Model().BBox(lower, nil)
	for _, pos := range []cube.Pos{lower, upper} {
		got := src[pos].Model().BBox(pos, src)
		if len(got) != 1 || got[0] != want[0] {
			t.Fatalf("door half at %v has box %v, want %v", pos, got, want)
		}
	}
}
