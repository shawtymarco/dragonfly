package server

import (
	"encoding/base64"
	"image/color"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

func TestParseSkinPreservesPersonaAppearance(t *testing.T) {
	packID := "11111111-1111-1111-1111-111111111111"
	data := login.ClientData{
		SkinImageWidth: 64, SkinImageHeight: 64,
		SkinData:          base64.StdEncoding.EncodeToString(make([]byte, 64*64*4)),
		SkinGeometry:      base64.StdEncoding.EncodeToString([]byte("{}")),
		SkinResourcePatch: base64.StdEncoding.EncodeToString([]byte(`{"geometry":{"default":"geometry.humanoid.custom"}}`)),
		SkinID:            "persona-full-id", PlayFabID: "playfab", PersonaSkin: true, PremiumSkin: true,
		CapeOnClassicSkin: true, CapeID: "cape", ArmSize: "wide", SkinColour: "#123456",
		SkinAnimationData:   base64.StdEncoding.EncodeToString([]byte("animation")),
		SkinGeometryVersion: base64.StdEncoding.EncodeToString([]byte("1.26.44")), ProfileHash: "profile",
		PersonaPieces:    []login.PersonaPiece{{PieceID: "piece", PieceType: "persona_body", PackID: packID, ProductID: "product"}},
		PieceTintColours: []login.PersonaPieceTintColour{{PieceType: "persona_body", Colours: [4]string{"#ff112233", "#0", "#0", "#0"}}},
	}
	parsed := (&Server{}).parseSkin(data)
	if !parsed.Persona || !parsed.Premium || !parsed.PersonaCapeOnClassic || parsed.ArmSize != protocol.ArmSizeWide {
		t.Fatalf("persona flags were not preserved: %+v", parsed)
	}
	if parsed.SkinColour != (color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}) {
		t.Fatalf("skin colour = %#v", parsed.SkinColour)
	}
	if len(parsed.PersonaPieces) != 1 || parsed.PersonaPieces[0].PieceType != protocol.PieceTypeBody {
		t.Fatalf("persona pieces = %#v", parsed.PersonaPieces)
	}
	if len(parsed.PieceTintColours) != 1 || parsed.PieceTintColours[0].Colours[0] != (color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}) {
		t.Fatalf("persona tints = %#v", parsed.PieceTintColours)
	}
	if string(parsed.AnimationData) != "animation" || string(parsed.GeometryDataEngineVersion) != "1.26.44" || parsed.ProfileHash != "profile" {
		t.Fatalf("opaque metadata was not preserved")
	}
}

func TestPersonaPieceTypeCoversLoginNames(t *testing.T) {
	for name, want := range map[string]uint32{
		"persona_body": protocol.PieceTypeBody, "persona_hand": protocol.PieceTypeHands,
		"persona_classic_skin": protocol.PieceTypeClassicSkin, "unknown": protocol.PieceTypeUnknown,
	} {
		if got := personaPieceType(name); got != want {
			t.Errorf("personaPieceType(%q) = %d, want %d", name, got, want)
		}
	}
}
