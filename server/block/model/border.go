package model

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// Border is a wall model whose collision boxes extend vertically from the block to the maximum world height.
type Border struct {
	Wall Wall
}

// BBox ...
func (b Border) BBox(pos cube.Pos, source world.BlockSource) []cube.BBox {
	boxes := b.Wall.BBox(pos, source)
	for i, box := range boxes {
		min, max := box.Min(), box.Max()
		boxes[i] = cube.Box(min.X(), 0, min.Z(), max.X(), math.MaxFloat64, max.Z())
	}
	return boxes
}

// FaceSolid ...
func (b Border) FaceSolid(pos cube.Pos, face cube.Face, source world.BlockSource) bool {
	return b.Wall.FaceSolid(pos, face, source)
}
