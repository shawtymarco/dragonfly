package player

import (
	"math"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestUpdateFallStatePreservesAirborneApexAcrossVerticalKnockback(t *testing.T) {
	p := &Player{playerData: &playerData{fallDistance: 8, mc: &entity.MovementComputer{}}}

	p.updateFallState(2)
	if got, want := p.FallDistance(), 6.0; got != want {
		t.Fatalf("fall distance after upward knockback: got %v, want %v", got, want)
	}

	p.updateFallState(0)
	if got, want := p.FallDistance(), 6.0; got != want {
		t.Fatalf("fall distance after a stationary airborne tick: got %v, want %v", got, want)
	}

	p.updateFallState(-3)
	if got, want := p.FallDistance(), 9.0; got != want {
		t.Fatalf("fall distance after descent resumes: got %v, want %v", got, want)
	}
}

type fallDamageHandler struct {
	NopHandler
	hits int
}

func (h *fallDamageHandler) HandleHurt(_ *Context, _ *float64, _ bool, _ *time.Duration, src world.DamageSource) {
	if _, ok := src.(entity.FallDamageSource); ok {
		h.hits++
	}
}

func TestAuthInputFallDamageWaitsForLanding(t *testing.T) {
	for _, test := range []struct {
		name      string
		positions []mgl32.Vec3
	}{
		{"repeated_airborne_input", []mgl32.Vec3{{0.5, 72, 0.5}, {0.5, 72, 0.5}, {0.5, 68, 0.5}, {0.5, 64, 0.5}}},
		{"passing_block_corner", []mgl32.Vec3{{0.5, 72, 0.5}, {1.5, 70, 1.5}, {1.5, 68, 1.5}, {1.5, 64, 1.5}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := world.Config{Synchronous: true}.New()
			t.Cleanup(func() { _ = w.Close() })
			if err := w.Do(func(tx *world.Tx) {
				for x := range 3 {
					for z := range 3 {
						tx.SetBlock(cube.Pos{x, 63, z}, block.Stone{}, nil)
					}
				}
				tx.SetBlock(cube.Pos{0, 70, 1}, block.Stone{}, nil)
				conn := &swimmingConn{packets: make(chan packet.Packet, 4096)}
				s := (session.Config{MaxChunkRadius: 1, HandleStop: func(*world.Tx, session.Controllable) {}}).New(conn)
				defer s.CloseConnection()
				handle := world.EntitySpawnOpts{ID: uuid.New()}.New(Type, Config{
					Session: s, Position: mgl64.Vec3{0.5, 75.5, 0.5},
				})
				p := tx.AddEntity(handle).(*Player)
				s.SetHandle(handle, p.Skin())
				defer func() { s.Close(nil, p); _ = tx.RemoveEntity(p).Close() }()
				h := &fallDamageHandler{}
				p.Handle(h)
				input := session.PlayerAuthInputHandler{}
				move := func(pos mgl32.Vec3, yaw float32) {
					t.Helper()
					if err := input.Handle(&packet.PlayerAuthInput{Position: pos.Add(mgl32.Vec3{0, 1.62}), Yaw: yaw}, s, tx, p); err != nil {
						t.Fatal(err)
					}
				}
				for i, pos := range test.positions {
					move(pos, 0)
					if i == len(test.positions)-1 {
						break
					}
					if p.OnGround() || p.Health() != 20 || h.hits != 0 {
						t.Fatalf("airborne frame %d: grounded=%v health=%v hits=%d", i, p.OnGround(), p.Health(), h.hits)
					}
					if want := 75.5 - p.Position().Y(); math.Abs(p.FallDistance()-want) > 0.0001 {
						t.Fatalf("airborne frame %d: distance=%v, want %v", i, p.FallDistance(), want)
					}
					// Rotation and the world tick use the same support rule without
					// changing or consuming the measured fall distance.
					distance := p.FallDistance()
					move(pos, 15)
					move(pos, 0)
					p.Tick(tx, int64(i+1))
					if p.OnGround() || p.Health() != 20 || p.FallDistance() != distance {
						t.Fatalf("airborne rotation/tick %d: grounded=%v health=%v distance=%v", i, p.OnGround(), p.Health(), p.FallDistance())
					}
				}
				if !p.OnGround() || p.Health() != 11 || h.hits != 1 || p.FallDistance() != 0 {
					t.Fatalf("landing: grounded=%v health=%v hits=%d distance=%v", p.OnGround(), p.Health(), h.hits, p.FallDistance())
				}
				// Neither duplicate positions nor rotation-only input may settle the same fall twice.
				landed := test.positions[len(test.positions)-1]
				move(landed, 0)
				move(landed, 15)
				if p.Health() != 11 || h.hits != 1 || p.FallDistance() != 0 {
					t.Fatalf("repeated landing: health=%v hits=%d distance=%v", p.Health(), h.hits, p.FallDistance())
				}
			}).Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDescendingStepsDoesNotAccumulateFall(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	if err := w.Do(func(tx *world.Tx) {
		conn := &swimmingConn{packets: make(chan packet.Packet, 4096)}
		s := (session.Config{MaxChunkRadius: 1, HandleStop: func(*world.Tx, session.Controllable) {}}).New(conn)
		defer s.CloseConnection()
		handle := world.EntitySpawnOpts{ID: uuid.New()}.New(Type, Config{
			Session: s, Position: mgl64.Vec3{0.5, 70, 0.5},
		})
		p := tx.AddEntity(handle).(*Player)
		s.SetHandle(handle, p.Skin())
		defer func() { s.Close(nil, p); _ = tx.RemoveEntity(p).Close() }()
		for i := range 13 {
			y := 70 - float64(i)*0.5
			if i%2 == 0 {
				tx.SetBlock(cube.Pos{i, int(y) - 1, 0}, block.Stone{}, nil)
			} else {
				tx.SetBlock(cube.Pos{i, int(y), 0}, block.Slab{Block: block.Stone{}}, nil)
			}
		}
		for i := range 13 {
			pos := mgl32.Vec3{float32(i) + 0.5, 70 - float32(i)*0.5 + 1.62, 0.5}
			if err := (session.PlayerAuthInputHandler{}).Handle(&packet.PlayerAuthInput{Position: pos}, s, tx, p); err != nil {
				t.Fatal(err)
			}
			if !p.OnGround() || p.Health() != 20 || p.FallDistance() != 0 {
				t.Fatalf("step %d: grounded=%v health=%v distance=%v", i, p.OnGround(), p.Health(), p.FallDistance())
			}
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateFallStateResetsAfterAscendingBeyondApex(t *testing.T) {
	p := &Player{playerData: &playerData{fallDistance: 3, mc: &entity.MovementComputer{}}}
	p.updateFallState(3)

	if got := p.FallDistance(); got != 0 {
		t.Fatalf("fall distance after ascending beyond the measured apex: got %v, want 0", got)
	}
}
