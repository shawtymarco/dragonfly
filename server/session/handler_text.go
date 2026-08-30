package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// TextHandler handles the Text packet.
type TextHandler struct{}

// Handle ...
func (TextHandler) Handle(p packet.Packet, _ *Session, _ *world.Tx, c Controllable) error {
	pk := p.(*packet.Text)

	if pk.TextType != packet.TextTypeChat {
		// Client modifications may emit display-only Text packets when a local
		// text hotkey is used. These packets do not represent chat and must not
		// turn an otherwise harmless client-side feature into a disconnect.
		return nil
	}
	// SourceName and XUID are client-provided and are not used to attribute the
	// message. Chat is always executed by the authenticated Controllable, so
	// accepting empty or stale identity fields cannot spoof another player.
	c.Chat(pk.Message)
	return nil
}
