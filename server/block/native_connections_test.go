package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

type nativeConnectionSource map[cube.Pos]world.Block

func (s nativeConnectionSource) Block(pos cube.Pos) world.Block {
	if b, ok := s[pos]; ok {
		return b
	}
	return Air{}
}

func TestNativeFireConnectionsPreserveAgeAndSupport(t *testing.T) {
	pos := cube.Pos{0, 64, 0}
	src := nativeConnectionSource{pos.Side(cube.FaceEast): Wood{Wood: OakWood()}, pos.Side(cube.FaceUp): Wood{Wood: OakWood()}}
	f := Fire{Age: 14}.DeriveState(pos, src).(Fire)
	if f.Age != 14 || !f.Connections.east || !f.Up {
		t.Fatalf("side fire=%+v", f)
	}
	src[pos.Side(cube.FaceDown)] = Stone{}
	f = f.DeriveState(pos, src).(Fire)
	if f.Age != 14 || f.Connections != (Connections{}) || f.Up {
		t.Fatalf("supported fire=%+v", f)
	}
}

func TestNativeWireConnectionsPreservePowerAndStepUp(t *testing.T) {
	pos := cube.Pos{0, 64, 0}
	src := nativeConnectionSource{pos.Side(cube.FaceEast): RedstoneWire{}, pos.Side(cube.FaceDown): Stone{}}
	r := RedstoneWire{Power: 13}.DeriveState(pos, src).(RedstoneWire)
	if r.Power != 13 || r.East != WireSide || r.West != WireSide || r.North != WireDisconnected {
		t.Fatalf("level wire=%+v", r)
	}
	src[pos.Side(cube.FaceEast)] = Stone{}
	src[pos.Side(cube.FaceEast).Side(cube.FaceUp)] = RedstoneWire{}
	r = r.DeriveState(pos, src).(RedstoneWire)
	if r.Power != 13 || r.East != WireUp || r.West != WireSide {
		t.Fatalf("stepping wire=%+v", r)
	}
}
