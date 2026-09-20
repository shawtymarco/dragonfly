package block

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item"
)

func TestItemFrameHashDistinguishesMapState(t *testing.T) {
	for _, glowing := range []bool{false, true} {
		for facing := cube.Face(0); facing <= cube.Face(5); facing++ {
			empty := ItemFrame{Facing: facing, Glowing: glowing}
			mapped := empty
			mapped.Item = item.NewStack(frameMapItem{}, 1)
			base, state := empty.Hash()
			mapBase, mapState := mapped.Hash()
			if base != mapBase || mapState != state|1<<4 {
				t.Fatalf("map state lost its stable bit: facing=%d glowing=%t hashes=%d/%d", facing, glowing, state, mapState)
			}
		}
	}
}
