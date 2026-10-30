package item

// OminousBottle is a bottle dropped by raid captains. Its meta is the level of
// Bad Omen it gives, from 0 to 4. Drinking it is not implemented, so using it
// does nothing by itself.
type OminousBottle struct {
	// Amplifier is the Bad Omen amplifier of the bottle, from 0 to 4.
	Amplifier int
}

// EncodeItem ...
func (o OminousBottle) EncodeItem() (name string, meta int16) {
	return "minecraft:ominous_bottle", int16(o.Amplifier)
}
