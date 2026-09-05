package sound

import "testing"

func TestNameFromIDMatchesPocketMineRegistry(t *testing.T) {
	for _, test := range []struct {
		id   int32
		name string
		ok   bool
	}{
		{id: 0, name: "item.use.on", ok: true},
		{id: 62, name: "levelup", ok: true},
		{id: 314, name: "record.pigstep", ok: true},
		{id: 378, ok: false},
		{id: 610, name: "geyser_continuous_eruption_active", ok: true},
		{id: 611, ok: false},
		{id: -1, ok: false},
	} {
		name, ok := NameFromID(test.id)
		if ok != test.ok || name != test.name {
			t.Fatalf("NameFromID(%d) = %q, %v; want %q, %v", test.id, name, ok, test.name, test.ok)
		}
	}
}
