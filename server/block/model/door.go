package model

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Door is a model used for doors. It has no solid faces and a bounding box that changes depending on
// the direction of the door, whether it is open, and the side of its hinge.
type Door struct {
	// Facing is the direction that the door is facing when closed.
	Facing cube.Direction
	// Open specifies if the Door is open. The direction it opens towards depends on the Right field.
	Open bool
	// Right specifies the attachment side of the door and, with that, the direction it opens in.
	Right bool
	// Top specifies if the model is of the upper half of the door.
	Top bool
}

// BBox returns a physics.BBox that depends on if the Door is open, what direction it is facing and whether it is
// attached to the right/left side of a block. Like the client, it takes the hinge from the upper half and the open
// state from the lower half, since worlds from vanilla can hold a stale copy of either in the other half.
func (d Door) BBox(pos cube.Pos, s world.BlockSource) []cube.BBox {
	if s != nil {
		if d.Top {
			if lower, ok := s.Block(pos.Side(cube.FaceDown)).Model().(Door); ok && !lower.Top {
				d.Open = lower.Open
			}
		} else if upper, ok := s.Block(pos.Side(cube.FaceUp)).Model().(Door); ok && upper.Top {
			d.Right = upper.Right
		}
	}
	if d.Open {
		if d.Right {
			return []cube.BBox{full.ExtendTowards(d.Facing.RotateLeft().Face(), -0.8125)}
		}
		return []cube.BBox{full.ExtendTowards(d.Facing.RotateRight().Face(), -0.8125)}
	}
	return []cube.BBox{full.ExtendTowards(d.Facing.Face(), -0.8125)}
}

// FaceSolid always returns false.
func (d Door) FaceSolid(cube.Pos, cube.Face, world.BlockSource) bool {
	return false
}
