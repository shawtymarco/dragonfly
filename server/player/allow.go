package player

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// canEdit checks if the player may place or break a block at pos. Adventure mode forbids editing, except
// above an allow block, as it does on Bedrock clients.
func (p *Player) canEdit(pos cube.Pos) bool {
	if p.GameMode().AllowsEditing() {
		return true
	}
	if p.GameMode() != world.GameModeAdventure {
		return false
	}
	return aboveAllow(pos, p.tx.Range().Min(), func(pos cube.Pos) string {
		name, _ := p.tx.Block(pos).EncodeBlock()
		return name
	})
}

// aboveAllow searches down the column under pos. The first allow or deny block found decides.
func aboveAllow(pos cube.Pos, minY int, name func(pos cube.Pos) string) bool {
	for y := pos.Y() - 1; y >= minY; y-- {
		switch name(cube.Pos{pos.X(), y, pos.Z()}) {
		case "minecraft:allow":
			return true
		case "minecraft:deny":
			return false
		}
	}
	return false
}
