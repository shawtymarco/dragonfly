package player

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/potion"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type customConsumeHandler struct {
	NopHandler
	cancelSlot bool
}

func (h customConsumeHandler) HandleHeldSlotChange(ctx *Context, _, _ int) {
	if h.cancelSlot {
		ctx.Cancel()
	}
}

func (customConsumeHandler) HandleItemConsume(ctx *Context, _ item.Stack) {
	// BedWars replaces vanilla consumption to customise potion effects and hide
	// golden-apple particles. Cancellation does not mean that no item was eaten.
	ctx.Cancel()
	p := ctx.Player()
	held, off := p.HeldItems()
	p.SetHeldItems(held.Grow(-1), off)
	p.AddEffect(effect.New(effect.Absorption, 1, time.Minute).WithoutParticles())
}

func TestConsumableStopReachesClientBeforeSwing(t *testing.T) {
	for _, test := range []struct {
		name   string
		held   item.Stack
		custom bool
		stop   func(*testing.T, *Player, *session.Session)
	}{
		{"release", item.NewStack(item.GoldenApple{}, 2), false, func(t *testing.T, p *Player, s *session.Session) {
			if err := (&session.PlayerActionHandler{}).Handle(&packet.PlayerAction{
				EntityRuntimeID: 1, ActionType: protocol.PlayerActionStopItemUseOn,
			}, s, p.Tx(), p); err != nil {
				t.Fatal(err)
			}
		}},
		{"hotbar_switch", item.NewStack(item.GoldenApple{}, 2), false, func(t *testing.T, p *Player, s *session.Session) {
			sword, _ := p.Inventory().Item(1)
			if err := s.VerifyAndSetHeldSlot(1, sword, p); err != nil {
				t.Fatal(err)
			}
		}},
		{"last_food", item.NewStack(item.Apple{}, 1), false, completeConsumption},
		{"last_golden_apple", item.NewStack(item.GoldenApple{}, 1), false, completeConsumption},
		{"potion_bottle", item.NewStack(item.Potion{Type: potion.Water()}, 1), false, completeConsumption},
		{"custom_golden_apple", item.NewStack(item.GoldenApple{}, 1), true, completeConsumption},
		{"custom_potion", item.NewStack(item.Potion{Type: potion.StrongSwiftness()}, 1), true, completeConsumption},
		{"custom_stack_then_switch", item.NewStack(item.GoldenApple{}, 2), true, func(t *testing.T, p *Player, s *session.Session) {
			completeConsumption(t, p, s)
			if !p.UsingItem() {
				t.Fatal("remaining stack lost continuous consumption")
			}
			sword, _ := p.Inventory().Item(1)
			if err := s.VerifyAndSetHeldSlot(1, sword, p); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := world.Config{Synchronous: true}.New()
			t.Cleanup(func() { _ = w.Close() })
			if err := w.Do(func(tx *world.Tx) {
				// Reuse the in-memory session writer fixture. Deliberately keep all
				// packets queued until after the switch and swing, as on a delayed path.
				conn := &swimmingConn{packets: make(chan packet.Packet, 4096)}
				s := (session.Config{MaxChunkRadius: 1, HandleStop: func(*world.Tx, session.Controllable) {}}).New(conn)
				defer s.CloseConnection()
				handle := world.EntitySpawnOpts{ID: uuid.New()}.New(Type, Config{
					Session: s, Position: mgl64.Vec3{0.5, 64, 0.5}, Food: 10,
				})
				p := tx.AddEntity(handle).(*Player)
				s.SetHandle(handle, p.Skin())
				defer func() { s.Close(nil, p); _ = tx.RemoveEntity(p).Close() }()
				if test.custom {
					p.Handle(customConsumeHandler{})
				}
				p.SetHeldItems(test.held, item.Stack{})
				_ = p.Inventory().SetItem(1, item.NewStack(item.Sword{Tier: item.ToolTierIron}, 1))
				drainSwimmingPackets(t, s, conn)
				p.UseItem()
				test.stop(t, p, s)
				p.PunchAir()

				using, started, stopped, swings := false, false, false, 0
				for _, pk := range drainSwimmingPackets(t, s, conn) {
					switch pk := pk.(type) {
					case *packet.SetActorData:
						if pk.EntityRuntimeID != 1 {
							continue
						}
						using = pk.EntityMetadata.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagUsingItem)
						started = started || using
						stopped = stopped || (started && !using)
					case *packet.Animate:
						if pk.ActionType == packet.AnimateActionSwingArm {
							swings++
							if using || !stopped {
								t.Error("swing reached the controlling client before consumption was cleared")
							}
						}
					}
				}
				if p.UsingItem() || using || !started || !stopped || swings != 1 {
					t.Fatalf("server using=%t client using=%t start=%t stop=%t swings=%d", p.UsingItem(), using, started, stopped, swings)
				}
				p.ReleaseItem()
				for _, pk := range drainSwimmingPackets(t, s, conn) {
					if _, ok := pk.(*packet.SetActorData); ok {
						t.Fatal("repeated release sent a second use-state correction")
					}
				}
			}).Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func completeConsumption(_ *testing.T, p *Player, _ *session.Session) {
	p.usingSince = time.Now().Add(-2 * time.Second)
	p.UseItem()
}

func TestConsumableStackReplacementAndCancelledSlot(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	if err := w.Do(func(tx *world.Tx) {
		p := tx.AddEntity(world.EntitySpawnOpts{ID: uuid.New()}.New(Type, Config{
			Position: mgl64.Vec3{0.5, 64, 0.5}, Food: 10,
		})).(*Player)
		viewer := &itemUseStateViewer{}
		loader := world.NewLoader(1, w, viewer)
		defer loader.Close(tx)
		loader.Move(tx, p.Position())
		loader.Load(tx, 16)
		held := item.NewStack(item.GoldenApple{}, 2)
		p.SetHeldItems(held, item.Stack{})
		p.UseItem()
		viewer.states = 0
		p.Handle(customConsumeHandler{cancelSlot: true})
		if err := p.SetHeldSlot(1); err != nil {
			t.Fatal(err)
		}
		p.SetHeldItems(held.Grow(-1), item.Stack{})
		if !p.UsingItem() || viewer.states != 0 {
			t.Fatal("cancelled slot change or comparable stack mutation stopped consumption")
		}
		p.SetHeldItems(item.NewStack(item.Apple{}, 1), item.Stack{})
		if p.UsingItem() || viewer.states != 1 {
			t.Fatal("replacement consumable inherited the old item's use state")
		}
		p.UseItem()
		if !p.UsingItem() {
			t.Fatal("replacement item could not start a fresh use cycle")
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}
