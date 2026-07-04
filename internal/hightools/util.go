package hightools

import "encoding/json"

// toInt64 attempts to convert a JSON-unmarshaled value to int64.
// JSON numbers decode as float64 by default; this handles both float64
// (whole numbers only) and int64 representations.
func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case float64:
		if val != float64(int64(val)) {
			return 0, false // reject non-integer floats like 1.5
		}
		return int64(val), true
	case int64:
		return val, true
	case int:
		return int64(val), true
	case json.Number:
		n, err := val.Int64()
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}