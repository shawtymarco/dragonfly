package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type modalResponseTestForm struct {
	submitted []byte
	err       error
}

func (*modalResponseTestForm) MarshalJSON() ([]byte, error) { return json.Marshal(map[string]any{}) }
func (f *modalResponseTestForm) SubmitJSON(data []byte, _ form.Submitter, _ *world.Tx) error {
	f.submitted = bytes.Clone(data)
	return f.err
}

func TestModalFormResponseHandlerIgnoresStaleForm(t *testing.T) {
	h := &ModalFormResponseHandler{forms: map[uint32]form.Form{}}
	for _, response := range []protocol.Optional[[]byte]{
		protocol.Option([]byte(`0`)),
		{},
	} {
		if err := h.Handle(&packet.ModalFormResponse{FormID: 17, ResponseData: response}, nil, nil, nil); err != nil {
			t.Fatalf("stale form response returned an error: %v", err)
		}
	}
}

func TestModalFormResponseHandlerSubmitsKnownForm(t *testing.T) {
	f := &modalResponseTestForm{}
	h := &ModalFormResponseHandler{forms: map[uint32]form.Form{7: f}}
	if err := h.Handle(&packet.ModalFormResponse{FormID: 7, ResponseData: protocol.Option([]byte(`1`))}, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.submitted, []byte(`1`)) {
		t.Fatalf("submitted response = %s, want 1", f.submitted)
	}
	if _, ok := h.forms[7]; ok {
		t.Fatal("submitted form was not removed")
	}
}

func TestModalFormResponseHandlerPreservesSubmissionErrors(t *testing.T) {
	want := errors.New("invalid response")
	f := &modalResponseTestForm{err: want}
	h := &ModalFormResponseHandler{forms: map[uint32]form.Form{9: f}}
	err := h.Handle(&packet.ModalFormResponse{FormID: 9, ResponseData: protocol.Option([]byte(`2`))}, nil, nil, nil)
	if !errors.Is(err, want) {
		t.Fatalf("submission error = %v, want %v", err, want)
	}
}
