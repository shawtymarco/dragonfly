package player

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

type invisibilityArmourHandler struct{ NopHandler }

func (invisibilityArmourHandler) HandleArmourVisibility(subject, viewer *Player) bool {
	return !subject.Invisible()
}

type armourTransitionViewer struct {
	world.NopViewer
	subject, viewer *world.EntityHandle
	visible         []bool
}

func (v *armourTransitionViewer) ViewEntityArmour(e world.Entity) {
	if e.H() == v.subject {
		v.visible = append(v.visible, e.(*Player).ArmourVisible(v.viewer))
	}
}

func TestArmourVisibilityTracksEffectLifetime(t *testing.T) {
	w := world.Config{Synchronous: true}.New()
	t.Cleanup(func() { _ = w.Close() })
	if err := w.Do(func(tx *world.Tx) {
		pos := mgl64.Vec3{0.5, 64, 0.5}
		spawn := func(name string) *Player {
			return tx.AddEntity(world.EntitySpawnOpts{Position: pos}.New(Type, Config{UUID: uuid.New(), Name: name, Position: pos})).(*Player)
		}
		subject, observer := spawn("Subject"), spawn("Observer")
		v := &armourTransitionViewer{subject: subject.H(), viewer: observer.H()}
		loader := world.NewLoader(2, w, v)
		defer loader.Close(tx)
		loader.Move(tx, pos)
		loader.Load(tx, 16)
		subject.Armour().SetHelmet(item.NewStack(item.Helmet{Tier: item.ArmourTierIron{}}, 1))
		subject.Handle(invisibilityArmourHandler{})
		v.visible = nil
		subject.AddEffect(effect.New(effect.Invisibility, 1, time.Second).WithoutParticles())
		if len(v.visible) != 1 || v.visible[0] {
			t.Fatalf("effect start did not immediately hide armour: %v", v.visible)
		}
		// Effect replacement calls End then Start, but must never expose armour
		// between those calls or at the superseded effect's expiry.
		subject.AddEffect(effect.New(effect.Invisibility, 1, 30*time.Second).WithoutParticles())
		subject.SetVisible()
		for range 21 {
			subject.effects.Tick(subject, tx)
		}
		if !subject.Invisible() || subject.ArmourVisible(observer.H()) || len(v.visible) != 1 {
			t.Fatalf("replacement effect exposed armour: %v", v.visible)
		}
		for range 580 {
			subject.effects.Tick(subject, tx)
		}
		if subject.Invisible() || !subject.ArmourVisible(observer.H()) || len(v.visible) != 2 || !v.visible[1] {
			t.Fatalf("actual effect expiry did not restore armour: %v", v.visible)
		}
		for _, mode := range []world.GameMode{world.GameModeSurvival, world.GameModeSpectator, world.GameModeNativeSpectator} {
			subject.SetGameMode(world.GameModeSurvival)
			subject.AddEffect(effect.New(effect.Invisibility, 1, time.Minute))
			subject.SetGameMode(mode)
			subject.RemoveEffect(effect.Invisibility)
			if mode.Visible() == subject.Invisible() {
				t.Fatalf("effect removal in %T left invisible=%t", mode, subject.Invisible())
			}
			subject.SetGameMode(world.GameModeSurvival)
			if subject.Invisible() || !subject.ArmourVisible(observer.H()) {
				t.Fatal("respawn retained invisibility")
			}
		}
		subject.AddEffect(effect.New(effect.Invisibility, 1, time.Minute))
		if !subject.ArmourVisible(subject.H()) || subject.ArmourVisible(nil) {
			t.Fatal("self/unavailable viewer policy is incorrect")
		}
		// Removing the handler must correct armour already hidden on clients.
		v.visible = nil
		subject.Handle(nil)
		if len(v.visible) != 1 || !v.visible[0] || !subject.ArmourVisible(observer.H()) {
			t.Fatal("handler replacement retained the old armour policy")
		}
		if subject.Armour().Helmet().Empty() {
			t.Fatal("visibility changed the authoritative armour inventory")
		}
	}).Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}
