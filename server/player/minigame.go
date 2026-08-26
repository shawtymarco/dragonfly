package player

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/go-gl/mathgl/mgl64"
)

// MinigameTick is an opt-in Player tick used by world when at least one
// minigame player/entity optimisation is enabled. It keeps combat, input,
// breaking, void damage, HUD state and server-controlled movement intact while
// allowing survival-only work to be skipped independently.
func (p *Player) MinigameTick(tx *world.Tx, current int64, conf world.MinigameConfig) {
	if p.Dead() {
		return
	}

	if !conf.DisablePlayerSurvivalTicks {
		if _, ok := p.tx.Liquid(cube.PosFromVec3(p.Position())); !ok {
			p.StopSwimming()
			if _, ok := p.Armour().Helmet().Item().(item.TurtleShell); ok {
				p.AddEffect(effect.New(effect.WaterBreathing, 1, time.Second*10).WithoutParticles())
			}
		}
		if _, ok := p.Armour().Chestplate().Item().(item.Elytra); ok && p.Gliding() {
			if p.glideTicks += 1; p.glideTicks%20 == 0 {
				d := p.damageItem(p.Armour().Chestplate(), 1)
				p.armour.SetChestplate(d)
				if d.Durability() < 2 {
					p.StopGliding()
				}
			}
		}
	}

	if !conf.DeduplicatePlayerCollisionTicks || p.session() == session.Nop {
		p.checkBlockCollisions(p.data.Vel)
		p.onGround = p.checkOnGround(mgl64.Vec3{})
	}
	p.checkEntitySteppers()

	if !conf.DisablePlayerEffectTicks {
		p.effects.Tick(p, p.tx)
	}
	if !conf.DisablePlayerSurvivalTicks {
		p.tickFood()
		p.tickAirSupply()
	}

	if p.Position()[1] < float64(p.tx.Range()[0]) {
		p.Hurt(4, entity.VoidDamageSource{})
	}
	if !conf.DisablePlayerSurvivalTicks {
		if p.insideOfSolid() {
			p.Hurt(1, entity.SuffocationDamageSource{})
		}
		if p.OnFireDuration() > 0 {
			p.fireTicks -= 1
			if !p.GameMode().AllowsTakingDamage() || p.OnFireDuration() <= 0 || p.tx.RainingAt(cube.PosFromVec3(p.Position())) {
				p.Extinguish()
			}
			if p.OnFireDuration()%time.Second == 0 {
				p.Hurt(1, block.FireDamageSource{})
			}
		}
	}

	held, _ := p.HeldItems()
	if current%4 == 0 && p.usingItem {
		if _, ok := held.Item().(item.Consumable); ok {
			for _, v := range p.viewers() {
				v.ViewEntityAction(p, entity.EatAction{})
			}
		}
	}
	if p.usingItem {
		if c, ok := held.Item().(item.Chargeable); ok {
			c.ContinueCharge(p, tx, p.useContext(), p.useDuration())
		}
	}
	if p.breaking {
		p.ContinueBreaking(p.breakingFace)
	}

	for it, ti := range p.cooldowns {
		if time.Now().After(ti) {
			delete(p.cooldowns, it)
		}
	}

	p.session().SendDebugShapes(tx.World().Dimension())
	p.session().SendHudUpdates()
	if p.prevWorld != tx.World() && p.prevWorld != nil {
		p.Handler().HandleChangeWorld(p, p.prevWorld, tx.World())
	}
	p.prevWorld = tx.World()

	before := p.Position()
	if p.session() == session.Nop && !p.Immobile() {
		m := p.mc.TickMovement(p, p.Position(), p.Velocity(), p.Rotation(), p.tx)
		m.Send()
		p.data.Vel = m.Velocity()
		p.Move(m.Position().Sub(p.Position()), 0, 0)
	} else {
		p.data.Vel = mgl64.Vec3{}
	}
	if !before.ApproxEqual(p.Position()) {
		world.MarkEntityMovementDirty(p)
	}
	if !conf.DisablePortalTicks {
		p.portalTravel.StopPortalContact()
	}
}

// FastFinishBreaking mirrors FinishBreaking's state validation before using the
// reduced minigame break path.
func (p *Player) FastFinishBreaking() {
	if !p.breaking {
		p.resendNearbyBlock(p.breakingPos)
		return
	}
	pos := p.breakingPos
	p.AbortBreaking()
	p.FastBreakBlock(pos)
}

// FastBreakBlock breaks a block using the reduced minigame path when enabled.
// It preserves reach/edit/border validation, HandleBlockBreak cancellation,
// swing animation and block-break particles, but intentionally skips vanilla
// drops, XP, BreakHandler callbacks, exhaustion and tool durability.
func (p *Player) FastBreakBlock(pos cube.Pos) {
	if !p.tx.MinigameConfig().FastBreakBlock {
		p.BreakBlock(pos)
		return
	}
	b := p.tx.Block(pos)
	if _, air := b.(block.Air); air {
		return
	}
	if !p.canReach(pos.Vec3Centre()) || !p.GameMode().AllowsEditing() {
		p.resendNearbyBlocks(pos)
		return
	}
	if _, border := b.(block.Border); border && (!p.GameMode().CreativeInventory() || !p.Operator()) {
		p.resendNearbyBlocks(pos)
		return
	}
	if _, breakable := b.(block.Breakable); !breakable && !p.GameMode().CreativeInventory() {
		p.resendNearbyBlocks(pos)
		return
	}

	var drops []item.Stack
	xp := 0
	ctx := NewEventContext(p.tx, p)
	if p.Handler().HandleBlockBreak(ctx, pos, &drops, &xp); ctx.Cancelled() {
		p.resendNearbyBlocks(pos)
		return
	}
	p.SwingArm()
	p.tx.FastSetBlock(pos, nil)
	p.tx.AddParticle(pos.Vec3Centre(), particle.BlockBreak{Block: b})
}
