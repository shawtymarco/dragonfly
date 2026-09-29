package player

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func rangedItemInstance(s item.Stack) protocol.ItemInstance {
	rid, meta, _ := world.ItemRuntimeID(s.Item())
	return protocol.ItemInstance{Stack: protocol.ItemStack{
		ItemType: protocol.ItemType{NetworkID: rid, MetadataValue: uint32(meta)},
		Count:    uint16(s.Count()), NBTData: item.WriteNBT(s, false),
	}}
}

var rangedFlushMarker = &packet.NetworkStackLatency{Timestamp: 987654}

type rangedConn struct{ swimmingConn }

func (c *rangedConn) Flush() error {
	c.packets <- rangedFlushMarker
	return nil
}

func withRangedPlayer(t *testing.T, f func(*Player, *session.Session, *swimmingConn)) {
	t.Helper()
	w := world.Config{Synchronous: true, Entities: entity.DefaultRegistry}.New()
	t.Cleanup(func() { _ = w.Close() })
	if err := w.Do(func(tx *world.Tx) {
		conn := &rangedConn{swimmingConn{packets: make(chan packet.Packet, 4096)}}
		s := (session.Config{MaxChunkRadius: 1, HandleStop: func(*world.Tx, session.Controllable) {}}).New(conn)
		defer s.CloseConnection()
		handle := world.EntitySpawnOpts{ID: uuid.New()}.New(Type, Config{
			Session: s, Position: mgl64.Vec3{0.5, 64, 0.5},
		})
		p := tx.AddEntity(handle).(*Player)
		s.SetHandle(handle, p.Skin())
		defer func() { s.Close(nil, p); _ = tx.RemoveEntity(p).Close() }()
		_ = p.Inventory().SetItem(0, item.NewStack(item.Bow{}, 1))
		_ = p.Inventory().SetItem(1, item.NewStack(item.Arrow{}, 8))
		drainSwimmingPackets(t, s, &conn.swimmingConn)
		f(p, s, &conn.swimmingConn)
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestBowStopActionDoesNotCorrectTheNextDraw(t *testing.T) {
	withRangedPlayer(t, func(p *Player, s *session.Session, conn *swimmingConn) {
		p.UseItem()
		p.usingSince = time.Now().Add(-time.Second)
		drainSwimmingPackets(t, s, conn)
		if err := (&session.PlayerActionHandler{}).Handle(&packet.PlayerAction{
			EntityRuntimeID: 1, ActionType: protocol.PlayerActionStopItemUseOn,
		}, s, p.Tx(), p); err != nil {
			t.Fatal(err)
		}
		bow, _ := p.HeldItems()
		arrows, _ := p.Inventory().Item(1)
		if p.UsingItem() || bow.Durability() != bow.MaxDurability()-1 || arrows.Count() != 7 {
			t.Fatalf("release state: using=%v bow=%v arrows=%d", p.UsingItem(), bow, arrows.Count())
		}
		// Keep release feedback queued until the next physical press, as can
		// happen at any latency when shots are close together.
		h := &session.InventoryTransactionHandler{}
		if err := h.Handle(&packet.InventoryTransaction{TransactionData: &protocol.UseItemTransactionData{
			ActionType: protocol.UseItemActionClickAir, HeldItem: rangedItemInstance(bow),
			TriggerType: protocol.TriggerTypePlayerInput,
		}}, s, p.Tx(), p); err != nil {
			t.Fatal(err)
		}
		if !p.UsingItem() {
			t.Fatal("next draw did not start")
		}
		for _, pk := range drainSwimmingPackets(t, s, conn) {
			if slot, ok := pk.(*packet.InventorySlot); ok && slot.WindowID == protocol.WindowIDInventory && slot.Slot == 0 {
				t.Fatal("release echoed the predicted bow slot into the next draw")
			}
		}
	})
}

func TestBowEquipmentAllowsPredictedDurability(t *testing.T) {
	withRangedPlayer(t, func(p *Player, s *session.Session, conn *swimmingConn) {
		p.UseItem()
		started := p.usingSince
		drainSwimmingPackets(t, s, conn)
		bow, _ := p.HeldItems()
		if err := (&session.MobEquipmentHandler{}).Handle(&packet.MobEquipment{
			EntityRuntimeID: 1, WindowID: protocol.WindowIDInventory,
			NewItem: rangedItemInstance(bow.Damage(1)),
		}, s, p.Tx(), p); err != nil {
			t.Fatal(err)
		}
		if !p.UsingItem() || !p.usingSince.Equal(started) {
			t.Fatal("equipment acknowledgement changed the active draw")
		}
		for _, pk := range drainSwimmingPackets(t, s, conn) {
			if _, ok := pk.(*packet.InventorySlot); ok {
				t.Fatal("predicted bow durability caused a held-slot correction")
			}
		}
		actual, _ := p.HeldItems()
		if actual.Durability() != bow.Durability() {
			t.Fatal("client prediction overwrote authoritative durability")
		}
		// Identity/custom-data differences still require an authoritative repair.
		if err := (&session.MobEquipmentHandler{}).Handle(&packet.MobEquipment{
			EntityRuntimeID: 1, WindowID: protocol.WindowIDInventory,
			NewItem: rangedItemInstance(bow.WithValue("forged", true)),
		}, s, p.Tx(), p); err != nil {
			t.Fatal(err)
		}
		corrected := false
		for _, pk := range drainSwimmingPackets(t, s, conn) {
			if slot, ok := pk.(*packet.InventorySlot); ok && slot.Slot == 0 {
				corrected = true
			}
		}
		if !corrected {
			t.Fatal("changed item data did not receive an authoritative repair")
		}
	})
}

func TestCrossbowReleaseCompletesReadyCharge(t *testing.T) {
	withRangedPlayer(t, func(p *Player, s *session.Session, conn *swimmingConn) {
		_ = p.Inventory().SetItem(0, item.NewStack(item.Crossbow{}, 1))
		p.UseItem()
		p.usingSince = time.Now().Add(-2 * time.Second)
		drainSwimmingPackets(t, s, conn)
		held, _ := p.HeldItems()
		if err := (&session.InventoryTransactionHandler{}).Handle(&packet.InventoryTransaction{
			TransactionData: &protocol.ReleaseItemTransactionData{HeldItem: rangedItemInstance(held)},
		}, s, p.Tx(), p); err != nil {
			t.Fatal(err)
		}
		held, _ = p.HeldItems()
		arrows, _ := p.Inventory().Item(1)
		if p.UsingItem() || !held.Item().(item.Crossbow).Charged() || arrows.Count() != 7 {
			t.Fatalf("completed charge lost on release: using=%v charged=%v arrows=%d", p.UsingItem(), held.Item().(item.Crossbow).Charged(), arrows.Count())
		}
		for e := range p.Tx().Entities() {
			if e.H().Type() == entity.ArrowType {
				t.Fatal("completing a charge also fired it")
			}
		}
		loadedUpdates := 0
		for _, pk := range drainSwimmingPackets(t, s, conn) {
			if slot, ok := pk.(*packet.InventorySlot); ok && slot.WindowID == protocol.WindowIDInventory && slot.Slot == 0 {
				loadedUpdates++
			}
		}
		if loadedUpdates != 1 {
			t.Fatalf("loaded crossbow updates=%d, want one", loadedUpdates)
		}
		p.ReleaseItem()
		arrows, _ = p.Inventory().Item(1)
		if arrows.Count() != 7 {
			t.Fatal("duplicate release loaded another arrow")
		}
		// A subsequent real press fires the loaded arrow once.
		p.UseItem()
		held, _ = p.HeldItems()
		projectiles := 0
		for e := range p.Tx().Entities() {
			if e.H().Type() == entity.ArrowType {
				projectiles++
			}
		}
		if held.Item().(item.Crossbow).Charged() || projectiles != 1 {
			t.Fatalf("loaded shot charged=%v projectiles=%d", held.Item().(item.Crossbow).Charged(), projectiles)
		}
	})
}

func TestEarlyCrossbowReleaseDoesNotLoad(t *testing.T) {
	withRangedPlayer(t, func(p *Player, s *session.Session, conn *swimmingConn) {
		_ = p.Inventory().SetItem(0, item.NewStack(item.Crossbow{}, 1))
		p.UseItem()
		p.ReleaseItem()
		held, _ := p.HeldItems()
		arrows, _ := p.Inventory().Item(1)
		if p.UsingItem() || held.Item().(item.Crossbow).Charged() || arrows.Count() != 8 {
			t.Fatalf("early release: using=%v charged=%v arrows=%d", p.UsingItem(), held.Item().(item.Crossbow).Charged(), arrows.Count())
		}
	})
}

func TestBowFeedbackFlushFollowsShotPackets(t *testing.T) {
	withRangedPlayer(t, func(p *Player, s *session.Session, conn *swimmingConn) {
		p.UseItem()
		p.usingSince = time.Now().Add(-time.Second)
		drainSwimmingPackets(t, s, conn)
		held, _ := p.HeldItems()
		release := &packet.InventoryTransaction{TransactionData: &protocol.ReleaseItemTransactionData{
			HeldItem: rangedItemInstance(held),
		}}
		h := &session.InventoryTransactionHandler{}
		if err := h.Handle(release, s, p.Tx(), p); err != nil {
			t.Fatal(err)
		}
		// Repeated release is idempotent and cannot create a second flush or arrow.
		if err := h.Handle(release, s, p.Tx(), p); err != nil {
			t.Fatal(err)
		}
		packets := drainSwimmingPackets(t, s, conn)
		ammoIndex, flushIndex, flushes := -1, -1, 0
		for i, pk := range packets {
			if pk == rangedFlushMarker {
				flushIndex, flushes = i, flushes+1
			}
			if slot, ok := pk.(*packet.InventorySlot); ok && slot.WindowID == protocol.WindowIDInventory && slot.Slot == 1 {
				ammoIndex = i
			}
		}
		if ammoIndex < 0 || flushIndex <= ammoIndex || flushes != 1 {
			t.Fatalf("shot feedback was not flushed in order: ammo=%d flush=%d count=%d", ammoIndex, flushIndex, flushes)
		}
	})
}
