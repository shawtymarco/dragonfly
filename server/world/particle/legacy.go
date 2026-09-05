package particle

// Legacy is a Bedrock legacy particle event expressed in the current native
// particle registry. ID is the particle ID without the LevelEvent particle
// mask. Data has particle-specific meaning.
//
// Prefer a concrete particle type when one exists. Legacy is intended for
// data-driven systems that must preserve a particle selected at runtime.
type Legacy struct {
	particle

	ID   int32
	Data int32
}
