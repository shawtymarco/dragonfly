package session

import (
	"testing"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestDataDrivenSoundEncoding(t *testing.T) {
	pos := mgl64.Vec3{1.25, 64.5, -2.75}
	for _, test := range []struct {
		name  string
		sound world.Sound
		check func(*testing.T, packet.Packet)
	}{
		{name: "legacy event", sound: sound.LegacyEvent{EventType: packet.LevelEventSoundClick, EventData: 7}, check: func(t *testing.T, value packet.Packet) {
			pk, ok := value.(*packet.LevelEvent)
			if !ok || pk.EventType != packet.LevelEventSoundClick || pk.EventData != 7 || pk.Position != vec64To32(pos) {
				t.Fatalf("packet = %#v", value)
			}
		}},
		{name: "named level sound", sound: sound.Named{Name: packet.SoundEventLevelUp, ExtraData: 9, DisableRelativeVolume: true}, check: func(t *testing.T, value packet.Packet) {
			pk, ok := value.(*packet.LevelSoundEvent)
			if !ok || pk.SoundType != packet.SoundEventLevelUp || pk.ExtraData != 9 || !pk.DisableRelativeVolume || pk.Position != vec64To32(pos) {
				t.Fatalf("packet = %#v", value)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := &Session{packets: make(chan outboundMessage, 1), closeBackground: make(chan struct{})}
			s.ViewSound(pos, test.sound)
			test.check(t, (<-s.packets).packet)
		})
	}
}
