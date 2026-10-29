package chunk

// testBlockRegistry is a minimal BlockRegistry for chunk tests.
type testBlockRegistry struct{}

func (testBlockRegistry) BlockCount() int                       { return 11 }
func (testBlockRegistry) AirRuntimeID() uint32                  { return 0 }
func (testBlockRegistry) FilteringBlock(uint32) uint8           { return 0 }
func (testBlockRegistry) LightBlock(uint32) uint8               { return 0 }
func (testBlockRegistry) RandomTickBlock(uint32) bool           { return false }
func (testBlockRegistry) NBTBlock(uint32) bool                  { return false }
func (testBlockRegistry) LiquidDisplacingBlock(uint32) bool     { return false }
func (testBlockRegistry) LiquidBlock(uint32) bool               { return false }
func (testBlockRegistry) HashToRuntimeID(uint32) (uint32, bool) { return 0, false }
func (testBlockRegistry) RuntimeIDToHash(uint32) (uint32, bool) { return 0, false }
func (testBlockRegistry) RuntimeIDToState(runtimeID uint32) (string, map[string]any, bool) {
	return "minecraft:test", map[string]any{"id": int32(runtimeID)}, true
}

func (testBlockRegistry) StateToRuntimeID(string, map[string]any) (uint32, bool) {
	return 0, false
}

func (testBlockRegistry) DefaultProperties(string) (map[string]any, bool) {
	return nil, false
}
