package server

import (
	"errors"
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

func TestReloadResources(t *testing.T) {
	l := &listener{}
	srv := &Server{listeners: []Listener{l}}
	initial, replacement := new(resource.Pack), new(resource.Pack)
	packs := []*resource.Pack{initial}
	if err := srv.ReloadResources(packs); err != nil {
		t.Fatalf("reload resources: %v", err)
	}
	packs[0] = replacement
	got := l.resourcePacks()
	if len(got) != 1 || got[0] != initial {
		t.Fatalf("resource packs were not cloned: %#v", got)
	}
	if err := srv.ReloadResources(packs); err != nil {
		t.Fatal(err)
	}
	if got[0] != initial {
		t.Fatal("reload changed the snapshot held by an existing negotiation")
	}
	got = l.resourcePacks()
	got[0] = initial
	if l.resourcePacks()[0] != replacement {
		t.Fatal("editing a returned snapshot changed the offered resources")
	}
	if err := srv.ReloadResources(nil); err != nil {
		t.Fatal(err)
	}
	if len(l.resourcePacks()) != 0 {
		t.Fatal("empty replacement retained the previous packs")
	}
}

func TestReloadResourcesUnsupported(t *testing.T) {
	if err := (&Server{}).ReloadResources(nil); !errors.Is(err, errResourceReloadUnsupported) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReloadResourcesForwardsThroughWrapper(t *testing.T) {
	l := &listener{}
	wrapped := &forwardingResourceListener{Listener: &forwardingResourceListener{Listener: l}}
	srv := &Server{listeners: []Listener{wrapped}}
	pack := new(resource.Pack)
	if err := srv.ReloadResources([]*resource.Pack{pack}); err != nil {
		t.Fatalf("reload wrapped listener: %v", err)
	}
	if got := l.resourcePacks(); len(got) != 1 || got[0] != pack {
		t.Fatalf("wrapper did not forward resources: %v", got)
	}
}

type forwardingResourceListener struct{ Listener }

func (l *forwardingResourceListener) ReloadResources(packs []*resource.Pack) {
	l.Listener.(ResourceReloadingListener).ReloadResources(packs)
}

func TestReloadResourcesSerializesListeners(t *testing.T) {
	a, b := new(resource.Pack), new(resource.Pack)
	firstEntered, secondEntered, releaseFirst := make(chan struct{}), make(chan struct{}), make(chan struct{})
	first := &observedResourceListener{listener: &listener{}, before: func(packs []*resource.Pack) {
		if packs[0] == b {
			close(secondEntered)
		}
	}}
	second := &observedResourceListener{listener: &listener{}, before: func(packs []*resource.Pack) {
		if packs[0] == a {
			close(firstEntered)
			<-releaseFirst
		}
	}}
	srv := &Server{listeners: []Listener{first, second}}
	firstDone, secondDone := make(chan error, 1), make(chan error, 1)
	go func() { firstDone <- srv.ReloadResources([]*resource.Pack{a}) }()
	<-firstEntered
	secondStarted := make(chan struct{})
	go func() {
		close(secondStarted)
		secondDone <- srv.ReloadResources([]*resource.Pack{b})
	}()
	<-secondStarted
	select {
	case <-secondEntered:
		t.Error("second reload changed a listener before the first reload completed")
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
	for index, l := range []*listener{first.listener, second.listener} {
		if got := l.resourcePacks(); len(got) != 1 || got[0] != b {
			t.Errorf("listener %d retained a different resource generation", index)
		}
	}
}

type observedResourceListener struct {
	*listener
	before func([]*resource.Pack)
}

func (l *observedResourceListener) ReloadResources(packs []*resource.Pack) {
	l.before(packs)
	l.listener.ReloadResources(packs)
}
