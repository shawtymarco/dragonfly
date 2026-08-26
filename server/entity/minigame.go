package entity

import (
	"time"

	"github.com/df-mc/dragonfly/server/world"
)

// MinigameTick provides a generic entity tick variant for minigame worlds. At
// present generic entities only need a specialised path for portal removal;
// otherwise the regular Tick implementation is used verbatim.
func (e *Ent) MinigameTick(tx *world.Tx, current int64, conf world.MinigameConfig) {
	if !conf.DisablePortalTicks {
		e.Tick(tx, current)
		return
	}

	y := e.data.Pos[1]
	if y < float64(tx.Range()[0]) && current%10 == 0 {
		_ = e.Close()
		return
	}
	e.SetOnFire(e.OnFireDuration() - time.Second/20)

	before := e.Position()
	m := e.Behaviour().Tick(e, tx)
	if m != nil {
		m.Send()
	}
	e.data.Age += time.Second / 20
	if !before.ApproxEqual(e.Position()) {
		world.MarkEntityMovementDirty(e)
	}
}
