package server

import "slices"

// cloneItemData preserves the concrete NBT types in a native item definition.
func cloneItemData(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	out := make(map[string]any, len(data))
	for key, value := range data {
		out[key] = cloneItemValue(value)
	}
	return out
}

func cloneItemValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		return cloneItemData(value)
	case []any:
		out := make([]any, len(value))
		for i, entry := range value {
			out[i] = cloneItemValue(entry)
		}
		return out
	case []byte:
		return slices.Clone(value)
	case []int32:
		return slices.Clone(value)
	case []int64:
		return slices.Clone(value)
	default:
		return value
	}
}
