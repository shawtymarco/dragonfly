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
