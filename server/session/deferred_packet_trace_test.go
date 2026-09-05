package session

import (
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestDeferredPacketTraceKeepsSingleOrderedResult(t *testing.T) {
	s := &Session{packets: make(chan outboundMessage, 8), closeBackground: make(chan struct{})}
	now := time.Now()
	s.beginPacketTrace(PacketTrace{ID: 42, ReceivedAt: now.Add(-time.Millisecond)}, now)
	d := s.DeferPacketTrace()
	if d == nil {
		t.Fatal("incoming action was not deferred")
	}
	s.RejectPacketTrace(42, "buffered")
	s.endPacketTrace()
	if len(s.packets) != 0 {
		t.Fatal("original handler published an early result")
	}
	// Another incoming packet must complete independently while the action waits.
	s.beginPacketTrace(PacketTrace{ID: 43, ReceivedAt: now}, now)
	s.RejectPacketTrace(43, PacketTraceReasonCooldown)
	s.endPacketTrace()
	if result := (<-s.packets).traceResult; result == nil || result.ID != 43 {
		t.Fatalf("unrelated packet result = %#v", result)
	}
	feedback := &packet.HurtArmour{}
	d.Run(func() {
		trace, ok := s.CurrentPacketTrace()
		if !ok || trace.ID != 42 {
			t.Fatal("deferred action lost its original trace")
		}
		s.writePacket(feedback)
		s.FinishPacketTraceAccepted(trace.ID)
	})
	d.Run(func() { t.Fatal("deferred action executed twice") })
	d.Cancel("cancelled")
	if first := <-s.packets; first.packet != feedback {
		t.Fatal("result overtook gameplay feedback")
	}
	result := (<-s.packets).traceResult
	if result == nil || result.ID != 42 || !result.Accepted || !result.Terminal || result.Reason != "accepted_deferred" || result.QueueDuration != time.Millisecond {
		t.Fatalf("deferred result = %#v", result)
	}
	if len(s.packets) != 0 {
		t.Fatal("duplicate deferred result")
	}
	if _, ok := s.CurrentPacketTrace(); ok {
		t.Fatal("deferred trace leaked into later packets")
	}
}

func TestDeferredPacketTraceCancellationAndMissingResult(t *testing.T) {
	for _, cancel := range []bool{true, false} {
		s := &Session{packets: make(chan outboundMessage, 4), closeBackground: make(chan struct{})}
		now := time.Now()
		s.beginPacketTrace(PacketTrace{ID: 7, ReceivedAt: now}, now)
		d := s.DeferPacketTrace()
		s.endPacketTrace()
		want := PacketTraceReasonInternal
		if cancel {
			want = "buffer_cancelled"
			d.Cancel(want)
			d.Run(func() { t.Fatal("cancelled action ran") })
		} else {
			d.Run(func() {})
		}
		d.Cancel("duplicate")
		result := (<-s.packets).traceResult
		if result == nil || result.Accepted || result.Reason != want || len(s.packets) != 0 {
			t.Fatalf("terminal result = %#v, want %s", result, want)
		}
	}
}
