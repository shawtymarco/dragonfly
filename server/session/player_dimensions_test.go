package session

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestPlayerStateUpdatesPreserveColliderUntilSizeChanges(t *testing.T) {
	s := &Session{}
	metadata := func(height float32) protocol.EntityMetadata {
		m := protocol.NewEntityMetadata()
		m[protocol.EntityDataKeyWidth], m[protocol.EntityDataKeyHeight] = float32(0.6), height
		m[protocol.EntityDataKeyName] = "Player"
		m.SetFlag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagUsingItem)
		return m
	}
	first := metadata(1.8)
	s.filterPlayerDimensions(1, first, true)
	if _, ok := first[protocol.EntityDataKeyHeight]; !ok {
		t.Fatal("spawn lost its collision dimensions")
	}
	state := metadata(1.8)
	state[protocol.EntityDataKeyName] = "Player 19 HP"
	s.filterPlayerDimensions(1, state, false)
	if _, ok := state[protocol.EntityDataKeyHeight]; ok {
		t.Fatal("name/item update reapplied the unchanged collider height")
	}
	if _, ok := state[protocol.EntityDataKeyWidth]; ok {
		t.Fatal("name/item update reapplied the unchanged collider width")
	}
	if !state.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagUsingItem) || state[protocol.EntityDataKeyName] != "Player 19 HP" {
		t.Fatal("collider filtering changed unrelated player metadata")
	}
	pose := metadata(1.5)
	s.filterPlayerDimensions(1, pose, false)
	if pose[protocol.EntityDataKeyHeight] != float32(1.5) || pose[protocol.EntityDataKeyWidth] != float32(0.6) {
		t.Fatal("a real pose change lost its dimensions")
	}
	respawn := metadata(1.5)
	s.filterPlayerDimensions(1, respawn, true)
	if _, ok := respawn[protocol.EntityDataKeyHeight]; !ok {
		t.Fatal("respawn inherited an omitted size from the previous actor")
	}
	s.forgetPlayerDimensions(1)
	next := metadata(1.5)
	s.filterPlayerDimensions(1, next, false)
	if _, ok := next[protocol.EntityDataKeyHeight]; !ok {
		t.Fatal("a newly tracked player inherited stale collider state")
	}
}
