package server

import (
	"errors"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

func TestReloadResources(t *testing.T) {
	l := &listener{}
	srv := &Server{listeners: []Listener{l}}
	packs := []*resource.Pack{nil}
	if err := srv.ReloadResources(packs); err != nil {
		t.Fatalf("reload resources: %v", err)
	}
	packs[0] = &resource.Pack{}
	got := l.resourcePacks()
	if len(got) != 1 || got[0] != nil {
		t.Fatalf("resource packs were not cloned: %#v", got)
	}
}

func TestReloadResourcesUnsupported(t *testing.T) {
	if err := (&Server{}).ReloadResources(nil); !errors.Is(err, errResourceReloadUnsupported) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResourcePackCompatible(t *testing.T) {
	tests := []struct {
		name        string
		gameVersion string
		minimum     resource.Version
		want        bool
	}{
		{name: "older client", gameVersion: "1.18.10", minimum: resource.Version{1, 21, 130}, want: false},
		{name: "same version", gameVersion: "1.18.10", minimum: resource.Version{1, 18, 10}, want: true},
		{name: "newer client", gameVersion: "1.26.45", minimum: resource.Version{1, 17, 0}, want: true},
		{name: "unknown version", gameVersion: "development", minimum: resource.Version{1, 26, 45}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resourcePackCompatible(test.gameVersion, test.minimum); got != test.want {
				t.Fatalf("compatible: got %t, want %t", got, test.want)
			}
		})
	}
}
