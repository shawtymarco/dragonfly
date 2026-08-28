package player

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
)

type replacingAttackKnockBackHandler struct {
	NopHandler
	force, height float64
}

func (h *replacingAttackKnockBackHandler) HandleAttackKnockBack(ctx *Context, _ world.Entity, force, height float64) {
	h.force, h.height = force, height
	ctx.Cancel()
}

func TestHandleAttackKnockBack(t *testing.T) {
	ctx := &Context{Context: &world.Context{}}
	if handleAttackKnockBack(NopHandler{}, ctx, nil, 0.4, 0.41) {
		t.Fatal("default handler replaced attack knockback")
	}

	replacing := &replacingAttackKnockBackHandler{}
	ctx = &Context{Context: &world.Context{}}
	if !handleAttackKnockBack(replacing, ctx, nil, 0.4, 0.41) {
		t.Fatal("custom handler did not replace attack knockback")
	}
	if replacing.force != 0.4 || replacing.height != 0.41 {
		t.Fatalf("hook received force/height %v/%v, want 0.4/0.41", replacing.force, replacing.height)
	}
}
