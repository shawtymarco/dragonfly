package player

import (
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

// TestSpectatorFlightSurvivesClientLanding pins the server side of entering a
// collisionless game mode from the ground.
//
// SendAbilities advertises flight and noclip unconditionally for a mode without
// collision, but the client keeps reporting InputFlagStopFlying for the ticks it
// still believes it is standing on a block. Honouring that cleared the server's
// flying state while the advertised value stayed set, and because StopFlying
// republishes the game mode, the client received a fresh SetPlayerGameType,
// MovePlayer and UpdateAbilities burst every tick. It never settled into noclip,
// so a player who entered spectator from the ground kept colliding and could
// still hit others until they toggled flight by hand.
func TestSpectatorFlightSurvivesClientLanding(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })

	pos := mgl64.Vec3{0.5, 64, 0.5}
	if err := w.Do(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 63, 0}, block.Stone{}, nil)
		pl := tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{
			UUID: uuid.New(), Name: "Spectator", Position: pos, GameMode: world.GameModeSurvival,
		})).(*Player)

		for _, mode := range []world.GameMode{world.GameModeSpectator, world.GameModeNativeSpectator} {
			pl.onGround = true
			pl.SetGameMode(mode)
			if !pl.Flying() {
				t.Fatalf("%T: entering the mode from the ground left the player not flying", mode)
			}
			pl.StopFlying()
			if !pl.Flying() {
				t.Fatalf("%T: a client landing report cleared flight, desyncing it from the advertised abilities", mode)
			}
			if pl.onGround {
				t.Fatalf("%T: entering collisionless mode retained the grounded state", mode)
			}

			// The client emits stationary input every tick. That path and the regular
			// entity tick must not reconstruct onGround from the block below while the
			// mode has no collision.
			pl.onGround = true
			pl.Move(mgl64.Vec3{}, 0, 0)
			if pl.onGround {
				t.Fatalf("%T: stationary input re-grounded the collisionless player", mode)
			}
			pl.onGround = true
			pl.Tick(tx, 1)
			if pl.onGround {
				t.Fatalf("%T: entity tick re-grounded the collisionless player", mode)
			}
		}

		// Leaving a collisionless mode must still be able to clear flight, otherwise a
		// respawning player would keep flying in survival.
		pl.SetGameMode(world.GameModeSurvival)
		if pl.Flying() {
			t.Fatal("returning to survival left the player flying")
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

// TestCreativeFlightStillTogglable guards the ordinary path: a colliding mode that
// allows flight must keep honouring the client's own toggle in both directions.
func TestCreativeFlightStillTogglable(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })

	pos := mgl64.Vec3{0.5, 64, 0.5}
	if err := w.Do(func(tx *world.Tx) {
		pl := tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{
			UUID: uuid.New(), Name: "Builder", Position: pos, GameMode: world.GameModeCreative,
		})).(*Player)

		if pl.Flying() {
			t.Fatal("creative player started out flying")
		}
		pl.StartFlying()
		if !pl.Flying() {
			t.Fatal("creative player could not start flying")
		}
		pl.StopFlying()
		if pl.Flying() {
			t.Fatal("creative player could not stop flying")
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

type spectatorVisibilityViewer struct {
	world.NopViewer
	shown, hidden int
}

func (v *spectatorVisibilityViewer) ViewEntity(e world.Entity) {
	if _, ok := e.(*Player); ok {
		v.shown++
	}
}

func (v *spectatorVisibilityViewer) HideEntity(e world.Entity) {
	if _, ok := e.(*Player); ok {
		v.hidden++
	}
}

func TestSpectatorModeRemovesAndRestoresViewerActor(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })

	viewer := &spectatorVisibilityViewer{}
	loader := world.NewLoader(2, w, viewer)
	pos := mgl64.Vec3{0.5, 64, 0.5}
	if err := w.Do(func(tx *world.Tx) {
		loader.Move(tx, pos)
		loader.Load(tx, 16)
		pl := tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{
			UUID: uuid.New(), Name: "Spectator", Position: pos, GameMode: world.GameModeSurvival,
		})).(*Player)

		initialShown := viewer.shown
		if initialShown == 0 {
			t.Fatal("viewer never received the initial player actor")
		}
		pl.SetGameMode(world.GameModeSpectator)
		if viewer.hidden != 1 {
			t.Fatalf("hide calls after entering faux spectator = %d, want 1", viewer.hidden)
		}
		pl.SetGameMode(world.GameModeNativeSpectator)
		if viewer.hidden != 1 {
			t.Fatalf("hide calls after switching spectator modes = %d, want 1", viewer.hidden)
		}
		pl.SetGameMode(world.GameModeSurvival)
		if viewer.shown != initialShown+1 {
			t.Fatalf("show calls after leaving spectator = %d, want %d", viewer.shown, initialShown+1)
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := w.Do(func(tx *world.Tx) { loader.Close(tx) }).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestSpectatorDoesNotBlockPlacementOrEntityInteraction(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })

	target := cube.Pos{0, 64, 0}
	if err := w.Do(func(tx *world.Tx) {
		spectatorPos := target.Vec3Middle()
		spectator := tx.AddEntity(world.EntitySpawnOpts{Position: spectatorPos}.New(Type, Config{
			UUID: uuid.New(), Name: "Spectator", Position: spectatorPos, GameMode: world.GameModeSurvival,
		})).(*Player)
		builderPos := mgl64.Vec3{0.5, 64, -5.5}
		builder := tx.AddEntity(world.EntitySpawnOpts{Position: builderPos}.New(Type, Config{
			UUID: uuid.New(), Name: "Builder", Position: builderPos, GameMode: world.GameModeCreative,
		})).(*Player)

		for _, mode := range []world.GameMode{world.GameModeSpectator, world.GameModeNativeSpectator} {
			spectator.SetGameMode(mode)
			if world.EntityHasCollision(spectator) {
				t.Fatalf("%T spectator remains collidable", mode)
			}
			builder.PlaceBlock(target, block.Stone{}, nil)
			if _, ok := tx.Block(target).(block.Stone); !ok {
				t.Fatalf("%T spectator blocked placement at its position", mode)
			}
			tx.SetBlock(target, nil, nil)
			if builder.AttackEntity(spectator) {
				t.Fatalf("%T spectator accepted an incoming attack", mode)
			}
			if spectator.AttackEntity(builder) {
				t.Fatalf("%T spectator affected another player", mode)
			}
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}
