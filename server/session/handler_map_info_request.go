package session

import (
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// MapInfoRequestHandler forwards map texture requests to the controllable.
type MapInfoRequestHandler struct{}

func (*MapInfoRequestHandler) Handle(p packet.Packet, _ *Session, _ *world.Tx, c Controllable) error {
	c.RequestMapInfo(p.(*packet.MapInfoRequest).MapID)
	return nil
}
