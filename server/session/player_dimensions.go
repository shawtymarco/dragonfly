package session

import "github.com/sandertv/gophertunnel/minecraft/protocol"

type playerDimensions struct {
	width, height float32
}

// filterPlayerDimensions keeps state-only updates from repeatedly assigning an
// unchanged collider to a player embedded in blocks. PocketMine-MP sends dirty
// metadata fields; name tags, item use and effects do not resend its dimensions.
// Spawn and actual pose/size changes still send both dimensions together. Flags
// remain in every update so predicted item use and spectator corrections retain
// their existing semantics.
func (s *Session) filterPlayerDimensions(id uint64, metadata protocol.EntityMetadata, full bool) {
	if id == 0 {
		return
	}
	width, widthOK := metadata[protocol.EntityDataKeyWidth].(float32)
	height, heightOK := metadata[protocol.EntityDataKeyHeight].(float32)
	if !widthOK || !heightOK {
		return
	}
	current := playerDimensions{width: width, height: height}
	s.entityMutex.Lock()
	previous, known := s.playerDimensions[id]
	if s.playerDimensions == nil {
		s.playerDimensions = make(map[uint64]playerDimensions)
	}
	s.playerDimensions[id] = current
	s.entityMutex.Unlock()
	if known && previous == current && !full {
		delete(metadata, protocol.EntityDataKeyWidth)
		delete(metadata, protocol.EntityDataKeyHeight)
	}
}

func (s *Session) forgetPlayerDimensions(id uint64) {
	s.entityMutex.Lock()
	delete(s.playerDimensions, id)
	s.entityMutex.Unlock()
}
