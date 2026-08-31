package entity

import (
	"iter"
	"math"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/block/cube/trace"
	"github.com/df-mc/dragonfly/server/block/model"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	fishingHookGravity   = 0.03
	fishingHookCastSpeed = 1.8
	fishingHookSize      = 0.25
	fishingTick          = time.Second / 20
)

var fishingKB = struct {
	sync.RWMutex
	playerXZ, playerY float64
	itemXZ, itemY     float64
}{
	playerXZ: 0.25,
	playerY:  0.15,
	itemXZ:   0.2,
	itemY:    0.1,
}

var fishingHooks sync.Map // owner UUID -> *world.EntityHandle

// SetFishingKnockback sets the horizontal and vertical pull applied when a
// fishing hook is retrieved while attached to a player or item.
func SetFishingKnockback(playerXZ, playerY, itemXZ, itemY float64) {
	fishingKB.Lock()
	fishingKB.playerXZ, fishingKB.playerY = playerXZ, playerY
	fishingKB.itemXZ, fishingKB.itemY = itemXZ, itemY
	fishingKB.Unlock()
}

// FishingKnockback returns the current fishing-hook pull values.
func FishingKnockback() (playerXZ, playerY, itemXZ, itemY float64) {
	fishingKB.RLock()
	defer fishingKB.RUnlock()
	return fishingKB.playerXZ, fishingKB.playerY, fishingKB.itemXZ, fishingKB.itemY
}

// UseFishingRod casts a hook for owner or retrieves the existing one. It
// returns true when a new hook was cast so the rod can take durability.
func UseFishingRod(tx *world.Tx, owner world.Entity) bool {
	if owner == nil {
		return false
	}
	if hook := fishingHookOf(tx, owner); hook != nil {
		if b, ok := hook.Behaviour().(*FishingHookBehaviour); ok && b.Hooked() {
			b.retrieve(hook)
			return false
		}
		_ = hook.Close()
		return false
	}

	eye := EyePosition(owner)
	dir := owner.Rotation().Vec3()
	desired := eye.Add(dir.Mul(0.5))
	spawn := clampFishingSpawn(tx, eye, desired)
	vel := dir.Mul(fishingHookCastSpeed)
	if v, ok := owner.(interface{ Velocity() mgl64.Vec3 }); ok {
		vel = vel.Add(v.Velocity().Mul(0.5))
	}
	opts := world.EntitySpawnOpts{Position: spawn, Velocity: vel, Rotation: owner.Rotation()}
	tx.AddEntity(NewFishingHook(opts, owner))
	return true
}

// NewFishingHook creates a fishing hook owned by the entity passed.
func NewFishingHook(opts world.EntitySpawnOpts, owner world.Entity) *world.EntityHandle {
	conf := FishingHookBehaviourConfig{Owner: owner.H(), Gravity: fishingHookGravity}
	h := opts.New(FishingHookType, conf)
	if owner != nil {
		fishingHooks.Store(owner.H().UUID(), h)
	}
	return h
}

func fishingHookOf(tx *world.Tx, owner world.Entity) *Ent {
	v, ok := fishingHooks.Load(owner.H().UUID())
	if !ok {
		return nil
	}
	handle := v.(*world.EntityHandle)
	if handle.Closed() {
		fishingHooks.Delete(owner.H().UUID())
		return nil
	}
	e, ok := handle.Entity(tx)
	if !ok {
		return nil
	}
	ent, ok := e.(*Ent)
	if !ok {
		return nil
	}
	if _, ok := ent.Behaviour().(*FishingHookBehaviour); !ok {
		return nil
	}
	return ent
}

func unbindFishingHook(owner *world.EntityHandle, hook *world.EntityHandle) {
	if owner == nil || hook == nil {
		return
	}
	if v, ok := fishingHooks.Load(owner.UUID()); ok && v == hook {
		fishingHooks.Delete(owner.UUID())
	}
}

// FishingHookType is a world.EntityType implementation for fishing hooks.
var FishingHookType fishingHookType

type fishingHookType struct{}

func (t fishingHookType) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	return &Ent{tx: tx, handle: handle, data: data}
}

func (fishingHookType) EncodeEntity() string { return "minecraft:fishing_hook" }
func (fishingHookType) BBox(world.Entity) cube.BBox {
	h := fishingHookSize / 2
	return cube.Box(-h, 0, -h, h, fishingHookSize, h)
}

func (fishingHookType) DecodeNBT(_ map[string]any, data *world.EntityData) {
	data.Data = FishingHookBehaviourConfig{Gravity: fishingHookGravity}.New()
}
func (fishingHookType) EncodeNBT(*world.EntityData) map[string]any { return nil }

// FishingHookBehaviourConfig holds optional parameters for a FishingHookBehaviour.
type FishingHookBehaviourConfig struct {
	Owner   *world.EntityHandle
	Gravity float64
}

func (conf FishingHookBehaviourConfig) Apply(data *world.EntityData) {
	data.Data = conf.New()
}

// New creates a FishingHookBehaviour using conf.
func (conf FishingHookBehaviourConfig) New() *FishingHookBehaviour {
	if conf.Gravity == 0 {
		conf.Gravity = fishingHookGravity
	}
	return &FishingHookBehaviour{
		BaseBehaviour: NewBaseBehaviour(),
		owner:         conf.Owner,
		gravity:       conf.Gravity,
		mc: &MovementComputer{
			Gravity:           conf.Gravity,
			Drag:              0,
			DragBeforeGravity: true,
		},
	}
}

// FishingHookBehaviour implements the UTOPIA-style hook: fly, attach to a
// player or dropped item, then pull that entity on retrieve.
type FishingHookBehaviour struct {
	BaseBehaviour

	owner   *world.EntityHandle
	hooked  *world.EntityHandle
	gravity float64
	mc      *MovementComputer

	arcing, released, close, hookedIsItem bool
	releasedAge                           time.Duration
}

// Owner ...
func (b *FishingHookBehaviour) Owner() *world.EntityHandle {
	return b.owner
}

// Hooked reports whether the hook is attached to an entity.
func (b *FishingHookBehaviour) Hooked() bool {
	return b.hooked != nil
}

// Tick ...
func (b *FishingHookBehaviour) Tick(e *Ent, tx *world.Tx) *Movement {
	if b.close {
		b.unbind(e)
		_ = e.Close()
		return nil
	}

	owner, ok := b.ownerEntity(tx)
	if !ok || !holdingFishingRod(owner) || livingDead(owner) {
		b.close = true
		b.unbind(e)
		_ = e.Close()
		return nil
	}

	if !b.Hooked() && !b.released && !b.arcing && e.Age() > 5*fishingTick && owner.Position().Sub(e.Position()).Len() > 8 {
		vel := e.Velocity()
		e.SetVelocity(mgl64.Vec3{vel[0] * 0.7, vel[1], vel[2] * 0.7})
		b.gravity = 0.15
		b.mc.Gravity = 0.15
		b.arcing = true
	}

	if !b.Hooked() && !b.released {
		b.scanNearby(e, tx, owner)
	}

	if b.hooked != nil {
		hooked, ok := b.hooked.Entity(tx)
		if !ok {
			b.close = true
			b.unbind(e)
			_ = e.Close()
			return nil
		}
		if hooked.Position().Sub(owner.Position()).Len() > 8 {
			b.hooked = nil
			b.hookedIsItem = false
			b.released = true
			b.releasedAge = e.Age()
			b.gravity = fishingHookGravity
			b.mc.Gravity = fishingHookGravity
			e.SetVelocity(mgl64.Vec3{})
		} else {
			pos := hooked.Position()
			pos[1] += hooked.H().Type().BBox(hooked).Height()
			e.Teleport(pos)
			e.SetVelocity(mgl64.Vec3{})
			return nil
		}
	}

	if b.released && e.Age()-b.releasedAge > 40*fishingTick {
		b.close = true
		b.unbind(e)
		_ = e.Close()
		return nil
	}

	m, hit := b.fly(e, tx)
	e.data.Pos, e.data.Vel, e.data.Rot = m.pos, m.vel, m.rot
	if hit == nil {
		return m
	}

	switch r := hit.(type) {
	case trace.EntityResult:
		b.hitEntity(e, tx, r.Entity())
	case trace.BlockResult:
		b.close = true
	default:
		// Full-cell door intercepts return a generic BBox result.
		b.close = true
	}
	if b.close {
		b.unbind(e)
		_ = e.Close()
		return nil
	}
	return m
}

func (b *FishingHookBehaviour) retrieve(e *Ent) {
	owner, ok := b.ownerEntity(e.tx)
	if !ok || b.hooked == nil {
		b.close = true
		b.unbind(e)
		_ = e.Close()
		return
	}
	target, ok := b.hooked.Entity(e.tx)
	if !ok {
		b.close = true
		b.unbind(e)
		_ = e.Close()
		return
	}
	diff := owner.Position().Sub(target.Position())
	dist := diff.Len()
	if dist < 0.5 {
		b.close = true
		b.unbind(e)
		_ = e.Close()
		return
	}
	playerXZ, playerY, itemXZ, itemY := FishingKnockback()
	xz, y := playerXZ, playerY
	if b.hookedIsItem {
		xz, y = itemXZ, itemY
	}
	base := math.Min(dist/15, 1)
	motion := diff.Mul(base * xz / dist)
	motion[1] = base * y
	if v, ok := target.(interface{ SetVelocity(mgl64.Vec3) }); ok {
		v.SetVelocity(motion)
	}
	b.close = true
	b.unbind(e)
	_ = e.Close()
}

func (b *FishingHookBehaviour) unbind(e *Ent) {
	unbindFishingHook(b.owner, e.H())
}

func (b *FishingHookBehaviour) ownerEntity(tx *world.Tx) (world.Entity, bool) {
	if b.owner == nil {
		return nil, false
	}
	return b.owner.Entity(tx)
}

func (b *FishingHookBehaviour) attach(e *Ent, target world.Entity, isItem bool) {
	b.hooked = target.H()
	b.hookedIsItem = isItem
	e.SetVelocity(mgl64.Vec3{})
	b.gravity = 0
	b.mc.Gravity = 0
}

func (b *FishingHookBehaviour) hitEntity(e *Ent, tx *world.Tx, other world.Entity) {
	if other.H() == b.owner {
		b.close = true
		return
	}
	if isItemEntity(other) {
		b.attach(e, other, true)
		return
	}
	if !isPlayerEntity(other) {
		b.close = true
		return
	}
	if b.arcing {
		return
	}
	owner, ok := b.ownerEntity(tx)
	if ok && !fishingPathClear(tx, EyePosition(owner), other.Position()) {
		b.close = true
		return
	}
	b.attach(e, other, false)
	src := FishingHookDamageSource{Owner: owner, Projectile: e}
	HurtEntity(other, 0.1, src)
	if v, ok := other.(interface {
		Velocity() mgl64.Vec3
		SetVelocity(mgl64.Vec3)
	}); ok {
		vel := v.Velocity()
		vel[1] += 0.35
		v.SetVelocity(vel)
	}
}

func (b *FishingHookBehaviour) scanNearby(e *Ent, tx *world.Tx, owner world.Entity) {
	box := e.H().Type().BBox(e).Translate(e.Position()).Grow(0.5)
	for other := range tx.EntitiesWithin(box.Grow(1)) {
		if other.H() == e.H() || other.H() == owner.H() || !world.EntityHasCollision(other) {
			continue
		}
		if !other.H().Type().BBox(other).Translate(other.Position()).IntersectsWith(box) {
			continue
		}
		itemEnt := isItemEntity(other)
		playerEnt := isPlayerEntity(other)
		if playerEnt && b.arcing {
			continue
		}
		if !itemEnt && !playerEnt {
			continue
		}
		if !fishingPathClear(tx, e.Position(), other.Position()) {
			continue
		}
		if itemEnt {
			b.attach(e, other, true)
			return
		}
		b.hitEntity(e, tx, other)
		return
	}
}

func (b *FishingHookBehaviour) fly(e *Ent, tx *world.Tx) (*Movement, trace.Result) {
	pos, vel := e.Position(), e.Velocity()
	viewers := tx.Viewers(pos)

	velBefore := vel
	vel = b.mc.applyHorizontalForces(tx, pos, b.mc.applyVerticalForces(vel))
	rot := cube.Rotation{
		mgl64.RadToDeg(math.Atan2(vel[0], vel[2])),
		mgl64.RadToDeg(math.Atan2(vel[1], math.Hypot(vel[0], vel[2]))),
	}

	end := pos.Add(vel)
	var hit trace.Result
	if !mgl64.FloatEqual(end.Sub(pos).LenSqr(), 0) {
		if result, ok := fishingTrace(pos, end, tx, e.H().Type().BBox(e).Grow(1.0), b.ignores(e)); ok {
			hit = result
			if _, isBlock := result.(trace.BlockResult); isBlock {
				vel[1] = (vel[1] + b.mc.Gravity) / (1 - b.mc.Drag)
				vel = mgl64.Vec3{}
			} else {
				vel = mgl64.Vec3{}
			}
			end = result.Position()
		}
	}
	return &Movement{v: viewers, e: e, pos: end, vel: vel, dpos: end.Sub(pos), dvel: vel.Sub(velBefore), rot: rot}, hit
}

func (b *FishingHookBehaviour) ignores(e *Ent) trace.EntityFilter {
	return func(seq iter.Seq[world.Entity]) iter.Seq[world.Entity] {
		return func(yield func(world.Entity) bool) {
			for other := range seq {
				if e.H() == other.H() || b.owner == other.H() {
					continue
				}
				if !world.EntityHasCollision(other) {
					continue
				}
				if b.arcing && isPlayerEntity(other) {
					continue
				}
				if !yield(other) {
					return
				}
			}
		}
	}
}

func fishingTrace(start, end mgl64.Vec3, tx *world.Tx, box cube.BBox, filter trace.EntityFilter) (hit trace.Result, ok bool) {
	if !mgl64.FloatEqual(start.Sub(end).LenSqr(), 0) {
		trace.TraverseBlocks(start, end, func(pos cube.Pos) bool {
			if result, hitBlock := fishingBlockIntercept(tx, pos, start, end); hitBlock {
				hit = result
				end = hit.Position()
				return false
			}
			return true
		})
	}

	dist := math.MaxFloat64
	bb := box.Translate(start).Extend(end.Sub(start))
	entities := tx.EntitiesWithin(bb.Grow(8.0))
	if filter != nil {
		entities = filter(entities)
	}
	for ent := range entities {
		if !ent.H().Type().BBox(ent).Translate(ent.Position()).IntersectsWith(bb) {
			continue
		}
		result, hitEnt := trace.EntityIntercept(ent, start, end)
		if !hitEnt {
			continue
		}
		if d := start.Sub(result.Position()).LenSqr(); d < dist {
			dist = d
			hit = result
		}
	}
	return hit, hit != nil
}

func fishingBlockIntercept(tx *world.Tx, pos cube.Pos, start, end mgl64.Vec3) (trace.Result, bool) {
	b := tx.Block(pos)
	switch m := b.Model().(type) {
	case model.Door:
		if m.Open {
			return nil, false
		}
		return fishingFullCellIntercept(pos, start, end)
	case model.Trapdoor:
		if m.Open {
			return nil, false
		}
		return fishingFullCellIntercept(pos, start, end)
	case model.FenceGate:
		if m.Open {
			return nil, false
		}
		return fishingFullCellIntercept(pos, start, end)
	case model.Thin:
		return fishingFullCellIntercept(pos, start, end)
	}
	res, ok := trace.BlockIntercept(pos, tx, b, start, end)
	if !ok {
		return nil, false
	}
	return res, true
}

func fishingFullCellIntercept(pos cube.Pos, start, end mgl64.Vec3) (trace.Result, bool) {
	res, ok := trace.BBoxIntercept(cube.Box(0, 0, 0, 1, 1, 1).Translate(pos.Vec3()), start, end)
	if !ok {
		return nil, false
	}
	return res, true
}

func clampFishingSpawn(tx *world.Tx, eye, desired mgl64.Vec3) mgl64.Vec3 {
	if mgl64.FloatEqual(eye.Sub(desired).LenSqr(), 0) {
		return desired
	}
	spawn := desired
	trace.TraverseBlocks(eye, desired, func(pos cube.Pos) bool {
		hit, ok := fishingBlockIntercept(tx, pos, eye, desired)
		if !ok {
			return true
		}
		dir := desired.Sub(eye)
		if n := dir.Len(); n > 0 {
			spawn = hit.Position().Sub(dir.Mul(0.05 / n))
		} else {
			spawn = hit.Position()
		}
		return false
	})
	return spawn
}

func fishingPathClear(tx *world.Tx, start, end mgl64.Vec3) bool {
	if mgl64.FloatEqual(start.Sub(end).LenSqr(), 0) {
		return true
	}
	clear := true
	trace.TraverseBlocks(start, end, func(pos cube.Pos) bool {
		if _, ok := fishingBlockIntercept(tx, pos, start, end); ok {
			clear = false
			return false
		}
		return true
	})
	return clear
}

func holdingFishingRod(e world.Entity) bool {
	c, ok := e.(item.Carrier)
	if !ok {
		return false
	}
	held, _ := c.HeldItems()
	_, ok = held.Item().(item.FishingRod)
	return ok
}

func livingDead(e world.Entity) bool {
	l, ok := e.(Living)
	return ok && l.Dead()
}

func isItemEntity(e world.Entity) bool {
	return e.H().Type() == ItemType
}

func isPlayerEntity(e world.Entity) bool {
	_, living := e.(Living)
	_, gm := e.(interface{ GameMode() world.GameMode })
	return living && gm
}
