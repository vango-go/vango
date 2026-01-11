package urlstate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// DefaultSerializers

func DefaultSerializer[T any](zero T) func(T) string {
	return func(v T) string {
		switch val := any(v).(type) {
		case string:
			return val
		case int, int64, int32:
			return fmt.Sprintf("%d", val)
		case float64, float32:
			return fmt.Sprintf("%g", val)
		case bool:
			return fmt.Sprintf("%v", val)
		default:
			// Fallback to JSON for complex types or unknown
			b, err := json.Marshal(v)
			if err != nil {
				return ""
			}
			return string(b)
		}
	}
}

func DefaultDeserializer[T any](zero T) func(string) T {
	return func(s string) T {
		var result any = zero

		switch result.(type) {
		case string:
			return any(s).(T)
		case int:
			i, err := strconv.Atoi(s)
			if err != nil {
				return zero
			}
			return any(i).(T)
		case int64:
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return zero
			}
			return any(i).(T)
		case float64:
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return zero
			}
			return any(f).(T)
		case bool:
			b, err := strconv.ParseBool(s)
			if err != nil {
				return zero
			}
			return any(b).(T)
		case []string:
			// Basic comma separation
			return any(strings.Split(s, ",")).(T)
		default:
			// Try JSON
			var val T
			if err := json.Unmarshal([]byte(s), &val); err == nil {
				return val
			}
			return zero
		}
	}
}
