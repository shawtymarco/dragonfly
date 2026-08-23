package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// Border is an Education Edition block that forms a vertically unbounded wall. It may only be placed by operators in
// creative mode.
type Border struct {
	transparent
	sourceWaterDisplacer

	NorthConnection WallConnectionType
	EastConnection  WallConnectionType
	SouthConnection WallConnectionType
	WestConnection  WallConnectionType
	Post            bool
}

// EncodeItem ...
func (Border) EncodeItem() (string, int16) {
	return "minecraft:border_block", 0
}

// EncodeBlock ...
func (b Border) EncodeBlock() (string, map[string]any) {
	return "minecraft:border_block", map[string]any{
		"wall_connection_type_north": b.NorthConnection.String(),
		"wall_connection_type_east":  b.EastConnection.String(),
		"wall_connection_type_south": b.SouthConnection.String(),
		"wall_connection_type_west":  b.WestConnection.String(),
		"wall_post_bit":              boolByte(b.Post),
	}
}

// Model ...
func (b Border) Model() world.BlockModel {
	return model.Border{Wall: model.Wall{
		NorthConnection: b.NorthConnection.Height(),
		EastConnection:  b.EastConnection.Height(),
		SouthConnection: b.SouthConnection.Height(),
		WestConnection:  b.WestConnection.Height(),
		Post:            b.Post,
	}}
}

// NeighbourUpdateTick ...
func (b Border) NeighbourUpdateTick(pos, _ cube.Pos, tx *world.Tx) {
	b, connectionsUpdated := b.calculateConnections(tx, pos)
	b, postUpdated := b.calculatePost(tx, pos)
	if connectionsUpdated || postUpdated {
		tx.SetBlock(pos, b, nil)
	}
}

// UseOnBlock ...
func (b Border) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, tx *world.Tx, user item.User, ctx *item.UseContext) (used bool) {
	if operator, ok := user.(interface {
		GameMode() world.GameMode
		Operator() bool
	}); ok && (!operator.GameMode().CreativeInventory() || !operator.Operator()) {
		return false
	}
	pos, _, used = firstReplaceable(tx, pos, face, b)
	if !used {
		return false
	}
	b, _ = b.calculateConnections(tx, pos)
	b, _ = b.calculatePost(tx, pos)
	place(tx, pos, b, user, ctx)
	return placed(ctx)
}

// SideClosed ...
func (Border) SideClosed(cube.Pos, cube.Pos, *world.Tx) bool {
	return false
}

func (b Border) wall() Wall {
	return Wall{
		NorthConnection: b.NorthConnection,
		EastConnection:  b.EastConnection,
		SouthConnection: b.SouthConnection,
		WestConnection:  b.WestConnection,
		Post:            b.Post,
	}
}

func (b Border) withWall(w Wall) Border {
	b.NorthConnection = w.NorthConnection
	b.EastConnection = w.EastConnection
	b.SouthConnection = w.SouthConnection
	b.WestConnection = w.WestConnection
	b.Post = w.Post
	return b
}

func (b Border) calculateConnections(tx *world.Tx, pos cube.Pos) (Border, bool) {
	w, updated := b.wall().calculateConnections(tx, pos)
	return b.withWall(w), updated
}

func (b Border) calculatePost(tx *world.Tx, pos cube.Pos) (Border, bool) {
	w, updated := b.wall().calculatePost(tx, pos)
	return b.withWall(w), updated
}

func allBorders() (borders []world.Block) {
	for _, north := range WallConnectionTypes() {
		for _, east := range WallConnectionTypes() {
			for _, south := range WallConnectionTypes() {
				for _, west := range WallConnectionTypes() {
					borders = append(borders,
						Border{NorthConnection: north, EastConnection: east, SouthConnection: south, WestConnection: west},
						Border{NorthConnection: north, EastConnection: east, SouthConnection: south, WestConnection: west, Post: true},
					)
				}
			}
		}
	}
	return
}
