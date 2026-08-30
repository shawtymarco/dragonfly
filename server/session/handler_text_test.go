package session

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type textTestControllable struct {
	Controllable
	messages []any
}

func (c *textTestControllable) Chat(message ...any) {
	c.messages = append(c.messages, message...)
}

func TestTextHandlerIgnoresClientSideText(t *testing.T) {
	for _, textType := range []byte{
		packet.TextTypeRaw,
		packet.TextTypeTip,
		packet.TextTypeSystem,
		packet.TextTypeObject,
		packet.TextTypeObjectWhisper,
		packet.TextTypeObjectAnnouncement,
	} {
		if err := (TextHandler{}).Handle(&packet.Text{TextType: textType, Message: "local hotkey"}, nil, nil, nil); err != nil {
			t.Fatalf("text type %d returned an error: %v", textType, err)
		}
	}
}

func TestTextHandlerAttributesChatToControllable(t *testing.T) {
	controllable := &textTestControllable{}
	pk := &packet.Text{
		TextType:   packet.TextTypeChat,
		SourceName: "stale client name",
		Message:    "hello",
		XUID:       "stale client xuid",
	}
	if err := (TextHandler{}).Handle(pk, nil, nil, controllable); err != nil {
		t.Fatal(err)
	}
	if len(controllable.messages) != 1 || controllable.messages[0] != "hello" {
		t.Fatalf("chat messages = %#v, want [hello]", controllable.messages)
	}
}
