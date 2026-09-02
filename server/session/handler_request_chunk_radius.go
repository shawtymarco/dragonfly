package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// RequestChunkRadiusHandler handles the RequestChunkRadius packet.
type RequestChunkRadiusHandler struct{}

// Handle ...
func (*RequestChunkRadiusHandler) Handle(p packet.Packet, s *Session, tx *world.Tx, _ Controllable) error {
	pk := p.(*packet.RequestChunkRadius)

	s.requestedChunkRadius = pk.ChunkRadius
	radius := effectiveChunkRadius(pk.ChunkRadius, s.maxChunkRadius)
	s.chunkRadius = radius
	s.chunkLoader.ChangeRadius(tx, int(radius))
	s.writePacket(&packet.ChunkRadiusUpdated{ChunkRadius: radius})
	return nil
}
