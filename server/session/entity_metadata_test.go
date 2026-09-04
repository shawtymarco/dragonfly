package session

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestSuppressSelfUsingItemMetadata(t *testing.T) {
	t.Run("self", func(t *testing.T) {
		metadata := protocol.NewEntityMetadata()
		metadata.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagUsingItem)
		if !suppressSelfUsingItemMetadata(selfEntityRuntimeID, metadata) {
			t.Fatal("self using-item metadata was not suppressed")
		}
		if metadata.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagUsingItem) {
			t.Fatal("self using-item flag remained set")
		}
	})

	t.Run("remote", func(t *testing.T) {
		metadata := protocol.NewEntityMetadata()
		metadata.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagUsingItem)
		if suppressSelfUsingItemMetadata(selfEntityRuntimeID+1, metadata) {
			t.Fatal("remote using-item metadata was suppressed")
		}
		if !metadata.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagUsingItem) {
			t.Fatal("remote using-item flag was cleared")
		}
	})

	t.Run("not using", func(t *testing.T) {
		metadata := protocol.NewEntityMetadata()
		if suppressSelfUsingItemMetadata(selfEntityRuntimeID, metadata) {
			t.Fatal("metadata without using-item flag reported suppression")
		}
	})
}
