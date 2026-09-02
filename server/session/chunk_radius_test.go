package session

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft"
)

type chunkRadiusLimitTestProtocol struct {
	minecraft.Protocol
	limit int
}

func (p chunkRadiusLimitTestProtocol) NetworkChunkRadiusLimit() int { return p.limit }

func TestMaxChunkRadiusForProtocol(t *testing.T) {
	tests := []struct {
		name       string
		configured int
		protocol   minecraft.Protocol
		want       int
	}{
		{name: "no protocol limit", configured: 10, want: 10},
		{name: "lower protocol limit", configured: 10, protocol: chunkRadiusLimitTestProtocol{limit: 9}, want: 9},
		{name: "higher protocol limit", configured: 8, protocol: chunkRadiusLimitTestProtocol{limit: 9}, want: 8},
		{name: "invalid protocol limit", configured: 10, protocol: chunkRadiusLimitTestProtocol{limit: 0}, want: 10},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := maxChunkRadiusForProtocol(test.configured, test.protocol); got != test.want {
				t.Fatalf("max chunk radius = %d, want %d", got, test.want)
			}
		})
	}
}
