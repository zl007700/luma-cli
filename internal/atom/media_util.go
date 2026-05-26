package atom

import (
	"encoding/json"
	"fmt"
)

func listFromAny(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}
	return nil
}

func firstPresent(item map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			return value
		}
	}
	return nil
}

func floatFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		out, _ := v.Float64()
		return out
	default:
		var out float64
		fmt.Sscanf(fmt.Sprint(value), "%f", &out)
		return out
	}
}

func intFromAny(value any, fallback int) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		out, _ := v.Int64()
		return int(out)
	default:
		var out int
		if _, err := fmt.Sscanf(fmt.Sprint(value), "%d", &out); err == nil {
			return out
		}
		return fallback
	}
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
