package player

import (
	"testing"

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
		pl := tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{
			UUID: uuid.New(), Name: "Spectator", Position: pos, GameMode: world.GameModeSurvival,
		})).(*Player)

		for _, mode := range []world.GameMode{world.GameModeSpectator, world.GameModeNativeSpectator} {
			pl.SetGameMode(mode)
			if !pl.Flying() {
				t.Fatalf("%T: entering the mode from the ground left the player not flying", mode)
			}
			pl.StopFlying()
			if !pl.Flying() {
				t.Fatalf("%T: a client landing report cleared flight, desyncing it from the advertised abilities", mode)
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
