package session

import (
	"fmt"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// MobEquipmentHandler handles the MobEquipment packet.
type MobEquipmentHandler struct{}

// Handle ...
func (*MobEquipmentHandler) Handle(p packet.Packet, s *Session, tx *world.Tx, c Controllable) error {
	pk := p.(*packet.MobEquipment)

	if pk.EntityRuntimeID != selfEntityRuntimeID {
		return errSelfRuntimeID
	}
	switch pk.WindowID {
	case protocol.WindowIDOffHand:
		// This window ID is expected, but we don't handle it.
		return nil
	case protocol.WindowIDInventory:
		slot, expected := int(pk.InventorySlot), stackToItem(s.br, pk.NewItem.Stack)
		actual, _ := s.inv.Item(slot)
		switch actual.Item().(type) {
		case item.Releasable, item.Chargeable:
			// Equipment acknowledgement may overlap a predicted shot. Use the
			// same identity checks as use/release transactions so a durability
			// difference does not resend the bow during its next draw.
			return s.verifyAndSetHeldSlotForInteraction(slot, expected, c)
		}
		return s.VerifyAndSetHeldSlot(slot, expected, c)
	default:
		return fmt.Errorf("only main inventory should be involved in slot change, got window ID %v", pk.WindowID)
	}
}
