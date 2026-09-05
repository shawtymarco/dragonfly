package session

import (
	"math"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// entityAttributes initializes client health in the spawn packet itself. Some
// actors, including Ender Dragons, start their death presentation when spawned
// without health, even while the server-side entity is alive.
func entityAttributes(e world.Entity) []protocol.AttributeValue {
	h, ok := e.(interface {
		Health() float64
		MaxHealth() float64
	})
	if !ok {
		return nil
	}
	current, maximum := float32(h.Health()), float32(h.MaxHealth())
	if maximum <= 0 || math.IsNaN(float64(maximum)) || math.IsInf(float64(maximum), 0) || math.IsNaN(float64(current)) || math.IsInf(float64(current), 0) {
		return nil
	}
	return []protocol.AttributeValue{{
		Name: "minecraft:health", Min: 0, Max: maximum, Value: min(max(current, 0), maximum),
	}}
}
