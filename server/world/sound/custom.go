package sound

// Custom is a sound identified by name. It may be used to play sounds defined
// by a resource pack.
type Custom struct {
	// Name is the identifier of the sound.
	Name string
	// Volume is the volume of the sound.
	Volume float64
	// Pitch is the pitch of the sound.
	Pitch float64

	sound
}

// LegacyEvent is a raw LevelEvent sound expressed in the current native event
// registry. EventData has event-specific meaning.
type LegacyEvent struct {
	EventType int32
	EventData int32

	sound
}

// Named is a LevelSoundEvent selected by its current native string name.
type Named struct {
	Name                  string
	ExtraData             int32
	DisableRelativeVolume bool

	sound
}
