package session

import (
	"testing"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestPacketTracePublishesOneTerminalResult(t *testing.T) {
	s := &Session{packets: make(chan outboundMessage, 4), closeBackground: make(chan struct{})}
	now := time.Now()
	s.beginPacketTrace(PacketTrace{ID: 42, ReceivedAt: now.Add(-time.Millisecond)}, now)
	s.RejectPacketTrace(42, PacketTraceReasonUnreachable)
	s.RejectPacketTrace(42, PacketTraceReasonHandler)
	s.endPacketTrace()

	message := <-s.packets
	if message.traceResult == nil {
		t.Fatal("expected a trace result")
	}
	result := *message.traceResult
	if result.ID != 42 || result.Accepted || !result.Terminal || result.Reason != PacketTraceReasonUnreachable {
		t.Fatalf("unexpected result: %#v", result)
	}
	select {
	case duplicate := <-s.packets:
		t.Fatalf("unexpected duplicate result: %#v", duplicate)
	default:
	}
}

func TestPacketTraceMissingResultFailsClosed(t *testing.T) {
	s := &Session{packets: make(chan outboundMessage, 1), closeBackground: make(chan struct{})}
	now := time.Now()
	s.beginPacketTrace(PacketTrace{ID: 7, ReceivedAt: now}, now)
	s.endPacketTrace()
	result := (<-s.packets).traceResult
	if result == nil || result.Reason != PacketTraceReasonInternal || result.Accepted {
		t.Fatalf("unexpected fail-closed result: %#v", result)
	}
}

func TestTraceBarrierFollowsPackets(t *testing.T) {
	s := &Session{packets: make(chan outboundMessage, 2), closeBackground: make(chan struct{})}
	pk := &packet.Text{}
	s.writePacket(pk)
	s.QueuePacketTraceFeedback(9, PacketTraceRoleVictim)
	if first := <-s.packets; first.packet != pk {
		t.Fatalf("first writer item = %#v, want packet", first)
	}
	if second := <-s.packets; second.traceResult == nil || second.traceResult.ID != 9 {
		t.Fatalf("second writer item = %#v, want trace barrier", second)
	}
}
