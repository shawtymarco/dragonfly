package session

import (
	"fmt"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/inventory"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// InventoryTransactionHandler handles the InventoryTransaction packet.
type InventoryTransactionHandler struct {
	lastRightClick     rightClickSignature
	lastRightClickTime time.Time
}

type rightClickSignature struct {
	face            int32
	playerPosition  mgl32.Vec3
	blockPosition   protocol.BlockPos
	clickedPosition mgl32.Vec3
}

// Handle ...
func (h *InventoryTransactionHandler) Handle(p packet.Packet, s *Session, tx *world.Tx, c Controllable) (err error) {
	pk := p.(*packet.InventoryTransaction)
	trace, traced := s.CurrentPacketTrace()

	if len(pk.LegacySetItemSlots) > 2 {
		return fmt.Errorf("too many slot sync requests in inventory transaction")
	}

	defer func() {
		// The client has requested the server to resend the specified slots even if they haven't changed server-side.
		// Handling these requests is necessary to ensure the client's inventory remains in sync with the server.
		for _, slot := range pk.LegacySetItemSlots {
			if len(slot.Slots) > 2 {
				err = fmt.Errorf("too many slots in slot sync request")
				return
			}
			switch slot.ContainerID {
			case protocol.ContainerOffhand:
				s.sendInv(s.offHand, protocol.WindowIDOffHand)
			case protocol.ContainerInventory:
				for _, slot := range slot.Slots {
					if i, err := s.inv.Item(int(slot)); err == nil {
						s.sendItem(i, int(slot), protocol.WindowIDInventory)
					}
				}
			}
		}
	}()

	switch data := pk.TransactionData.(type) {
	case *protocol.NormalTransactionData:
		h.resendInventories(s)
		// Always resend inventories with normal transactions. Most of the time we do not use these
		// transactions, so we're best off making sure the client and server stay in sync.
		if err := h.handleNormalTransaction(pk, s, c); err != nil {
			s.conf.Log.Debug("process packet: InventoryTransaction: verify Normal transaction actions: " + err.Error())
		}
		return
	case *protocol.MismatchTransactionData:
		// Just resend the inventory and don't do anything.
		h.resendInventories(s)
		return
	case *protocol.UseItemOnEntityTransactionData:
		if err = s.VerifyAndSetHeldSlot(int(data.HotBarSlot), stackToItem(s.br, data.HeldItem.Stack), c); err != nil {
			h.resyncHeldSlot(s, c, int(data.HotBarSlot))
			if traced {
				s.RejectPacketTrace(trace.ID, PacketTraceReasonHeldItem)
			}
			return
		}
		return h.handleUseItemOnEntityTransaction(data, s, tx, c)
	case *protocol.UseItemTransactionData:
		if err = s.VerifyAndSetHeldSlot(int(data.HotBarSlot), stackToItem(s.br, data.HeldItem.Stack), c); err != nil {
			h.resyncHeldSlot(s, c, int(data.HotBarSlot))
			return
		}
		return h.handleUseItemTransaction(data, s, tx, c)
	case *protocol.ReleaseItemTransactionData:
		if err = s.VerifyAndSetHeldSlot(int(data.HotBarSlot), stackToItem(s.br, data.HeldItem.Stack), c); err != nil {
			h.resyncHeldSlot(s, c, int(data.HotBarSlot))
			return
		}
		return h.handleReleaseItemTransaction(c)
	}
	return fmt.Errorf("unhandled inventory transaction type %T", pk.TransactionData)
}

// resendInventories resends all inventories of the player.
func (h *InventoryTransactionHandler) resendInventories(s *Session) {
	s.sendInv(s.inv, protocol.WindowIDInventory)
	s.sendInv(s.ui, protocol.WindowIDUI)
	s.sendInv(s.offHand, protocol.WindowIDOffHand)
	s.sendInv(s.armour.Inventory(), protocol.WindowIDArmour)
}

// resyncHeldSlot repairs only the state involved in a rejected held-item
// transaction. Successful use transactions rely on inventory listeners and
// the client's prediction instead of retransmitting every inventory window.
func (h *InventoryTransactionHandler) resyncHeldSlot(s *Session, c Controllable, requestedSlot int) {
	if requestedSlot >= 0 && requestedSlot <= 8 {
		if actual, err := s.inv.Item(requestedSlot); err == nil {
			s.sendItem(actual, requestedSlot, protocol.WindowIDInventory)
		}
	}
	s.SendHeldSlot(int(*s.heldSlot), c, true)
}

// handleNormalTransaction ...
func (h *InventoryTransactionHandler) handleNormalTransaction(pk *packet.InventoryTransaction, s *Session, c Controllable) error {
	if len(pk.Actions) != 2 {
		return fmt.Errorf("expected two actions for dropping an item, got %d", len(pk.Actions))
	}

	var (
		slot     int
		count    int
		expected item.Stack
	)
	for _, action := range pk.Actions {
		windowID, hasWindowID := action.WindowID.Value()
		switch {
		case action.SourceType == protocol.InventoryActionSourceWorld && action.InventorySlot == 0:
			if old := stackToItem(s.br, action.OldItem.Stack); !old.Empty() {
				return fmt.Errorf("unexpected non-empty old item in transaction action: %#v", action.OldItem)
			}
			count = int(action.NewItem.Stack.Count)
		case action.SourceType == protocol.InventoryActionSourceContainer && hasWindowID && windowID == protocol.WindowIDInventory:
			if expected = stackToItem(s.br, action.OldItem.Stack); expected.Empty() {
				return fmt.Errorf("unexpected empty old item in transaction action: %#v", action.OldItem)
			}
			slot = int(action.InventorySlot)
		default:
			return fmt.Errorf("unexpected action type in drop item transaction")
		}
	}

	actual, _ := s.inv.Item(slot)
	if count < 1 {
		return fmt.Errorf("expected at least one item to be dropped, got %d", count)
	}
	if count > actual.Count() {
		return fmt.Errorf("tried to throw %v items, but held only %v in slot", count, actual.Count())
	}
	if !expected.Equal(actual) {
		return fmt.Errorf("different item thrown than held in slot: %#v was thrown but held %#v", expected, actual)
	}

	// Explicitly don't re-use the thrown variable. This item was supplied by the user, and if some
	// logic in the Comparable() method was flawed, users would be able to cheat with item properties.
	// Only grow or shrink the held item to prevent any such issues.
	res := actual.Grow(count - actual.Count())
	if err := call(event.C(inventory.Holder(c)), int(*s.heldSlot), res, s.inv.Handler().HandleDrop); err != nil {
		return err
	}

	n := c.Drop(res)
	_ = s.inv.SetItem(slot, actual.Grow(-n))
	return nil
}

// handleUseItemOnEntityTransaction ...
func (h *InventoryTransactionHandler) handleUseItemOnEntityTransaction(data *protocol.UseItemOnEntityTransactionData, s *Session, tx *world.Tx, c Controllable) error {
	s.swingingArm.Store(true)
	defer s.swingingArm.Store(false)

	if data.TargetEntityRuntimeID == selfEntityRuntimeID {
		if trace, ok := s.CurrentPacketTrace(); ok {
			s.RejectPacketTrace(trace.ID, PacketTraceReasonSelfTarget)
		}
		return fmt.Errorf("invalid entity interaction: players cannot interact with themselves")
	}

	handle, ok := s.entityFromRuntimeID(data.TargetEntityRuntimeID)
	if !ok {
		// In some cases, for example when a falling block entity solidifies, latency may allow attacking an entity that
		// no longer exists server side. This is expected, so we shouldn't kick the player.
		s.conf.Log.Debug("invalid entity interaction: no entity with runtime ID", "ID", data.TargetEntityRuntimeID)
		if trace, traced := s.CurrentPacketTrace(); traced {
			s.RejectPacketTrace(trace.ID, PacketTraceReasonTargetMissing)
		}
		return nil
	}
	e, ok := handle.Entity(tx)
	if !ok {
		s.conf.Log.Debug("invalid entity interaction: entity is not in the same world (anymore)", "ID", data.TargetEntityRuntimeID)
		if trace, traced := s.CurrentPacketTrace(); traced {
			s.RejectPacketTrace(trace.ID, PacketTraceReasonTargetWorld)
		}
		return nil
	}
	var valid bool
	switch data.ActionType {
	case protocol.UseItemOnEntityActionInteract:
		valid = c.UseItemOnEntity(e)
	case protocol.UseItemOnEntityActionAttack:
		s.beginAttackMetadata(AttackMetadata{
			TargetRuntimeID: data.TargetEntityRuntimeID,
			HotBarSlot:      data.HotBarSlot,
			Position: mgl64.Vec3{
				float64(data.Position[0]),
				float64(data.Position[1]),
				float64(data.Position[2]),
			},
			ClickedPosition: mgl64.Vec3{
				float64(data.ClickedPosition[0]),
				float64(data.ClickedPosition[1]),
				float64(data.ClickedPosition[2]),
			},
			Latency: s.Latency(),
		})
		defer s.endAttackMetadata()
		valid = c.AttackEntity(e)
	default:
		return fmt.Errorf("unhandled UseItemOnEntity ActionType %v", data.ActionType)
	}
	if !valid {
		slot := int(*s.heldSlot)
		it, _ := s.inv.Item(slot)
		s.sendItem(it, slot, protocol.WindowIDInventory)
	}
	if trace, traced := s.CurrentPacketTrace(); traced && !s.PacketTraceFinished(trace.ID) {
		if valid {
			s.FinishPacketTraceAccepted(trace.ID)
		} else {
			s.RejectPacketTrace(trace.ID, PacketTraceReasonHandler)
		}
	}
	return nil
}

// handleUseItemTransaction ...
func (h *InventoryTransactionHandler) handleUseItemTransaction(data *protocol.UseItemTransactionData, s *Session, tx *world.Tx, c Controllable) error {
	pos := cube.Pos{int(data.BlockPosition[0]), int(data.BlockPosition[1]), int(data.BlockPosition[2])}
	if data.ActionType == protocol.UseItemActionClickBlock && h.repeatedRightClick(data, time.Now()) {
		return nil
	}
	var simulationTx *world.Tx
	if data.ActionType == protocol.UseItemActionClickBlock {
		simulationTx = tx
	}
	if (data.ActionType == protocol.UseItemActionClickBlock || data.ActionType == protocol.UseItemActionClickAir) &&
		skipSimulationTick(data.ActionType, data.TriggerType, c, simulationTx, pos) {
		return nil
	}
	if data.ClientPrediction == protocol.ClientPredictionSuccess || data.ActionType == protocol.UseItemActionBreakBlock {
		// Suppress echoing the swing animation only when the client has already predicted it locally.
		s.swingingArm.Store(true)
		defer s.swingingArm.Store(false)
	}

	switch data.ActionType {
	case protocol.UseItemActionBreakBlock:
		c.BreakBlock(pos)
	case protocol.UseItemActionClickBlock:
		c.UseItemOnBlock(pos, cube.Face(data.BlockFace), vec32To64(data.ClickedPosition))
	case protocol.UseItemActionClickAir:
		c.UseItem()
	default:
		return fmt.Errorf("unhandled UseItem ActionType %v", data.ActionType)
	}
	return nil
}

// repeatedRightClick reports whether data has the exact signature emitted by
// Bedrock's continued right-click spam. The 100 ms window and spatial epsilon
// match PocketMine-MP: deliberate clicks at a new position continue normally,
// while holding use against one block produces only the initial interaction.
func (h *InventoryTransactionHandler) repeatedRightClick(data *protocol.UseItemTransactionData, now time.Time) bool {
	current := rightClickSignature{
		face:            data.BlockFace,
		playerPosition:  data.Position,
		blockPosition:   data.BlockPosition,
		clickedPosition: data.ClickedPosition,
	}
	previous, previousTime := h.lastRightClick, h.lastRightClickTime
	h.lastRightClick, h.lastRightClickTime = current, now
	return !previousTime.IsZero() && now.Sub(previousTime) < 100*time.Millisecond &&
		previous.face == current.face && previous.blockPosition == current.blockPosition &&
		vec3DistanceSquared(previous.playerPosition, current.playerPosition) < 0.00001 &&
		vec3DistanceSquared(previous.clickedPosition, current.clickedPosition) < 0.00001
}

func vec3DistanceSquared(a, b mgl32.Vec3) float32 {
	delta := a.Sub(b)
	return delta.Dot(delta)
}

// skipSimulationTick drops Bedrock hold-repeats. A real tap is UNKNOWN; the
// client re-fires use transactions as SIMULATION_TICK while the button is
// held. Air use retains only consumption and active charging, while block use
// keeps its existing fishing-rod and iron-door guards.
func skipSimulationTick(action, trigger uint32, c Controllable, tx *world.Tx, pos cube.Pos) bool {
	if trigger != protocol.TriggerTypeSimulationTick {
		return false
	}
	held, _ := c.HeldItems()
	if action == protocol.UseItemActionClickAir {
		return skipAirSimulationTick(held, c.UsingItem())
	}
	if _, ok := held.Item().(item.FishingRod); ok {
		return true
	}
	if tx != nil {
		if _, ok := tx.Block(pos).(block.IronDoor); ok {
			return true
		}
	}
	return false
}

func skipAirSimulationTick(held item.Stack, using bool) bool {
	switch held.Item().(type) {
	case item.Consumable:
		// Consumption completion is signalled by a hold-repeat.
		return false
	case item.Chargeable:
		// An uncharged crossbow needs repeats until it reaches its charge
		// duration. Once charging ends, a residual repeat must not fire it.
		return !using
	default:
		// Releasable items finish through ReleaseItemTransaction. Plain
		// items, including server controls, must activate once per press.
		return true
	}
}

// handleReleaseItemTransaction ...
func (h *InventoryTransactionHandler) handleReleaseItemTransaction(c Controllable) error {
	c.ReleaseItem()
	return nil
}
