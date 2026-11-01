package world

import "testing"

// TestUnknownBlockEncodesNonNilNBT verifies that an unknown block placed without
// NBT, as BlockByName returns it, encodes a map the provider can add its
// position to when saving the block entity.
func TestUnknownBlockEncodesNonNilNBT(t *testing.T) {
	if data := (unknownBlock{}).EncodeNBT(); data == nil {
		t.Fatal("expected a non-nil map for an unknown block without NBT")
	}
}
