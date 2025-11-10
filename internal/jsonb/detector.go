package jsonb

import (
	"encoding/json"
	"strings"
)

// IsJSONB checks if a string value looks like JSONB
func IsJSONB(value string) bool {
	if value == "" {
		return false
	}

	// Quick check for JSON-like start
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return false
	}

	first := value[0]
	if first != '{' && first != '[' && first != '"' {
		// Could be null, true, false, or number
		if value == "null" || value == "true" || value == "false" {
			return true
		}
		// Try parsing as number
		var f float64
		err := json.Unmarshal([]byte(value), &f)
		return err == nil
	}

	// Try to parse as JSON
	var parsed interface{}
	err := json.Unmarshal([]byte(value), &parsed)
	return err == nil
}

// Type returns the type of a JSONB value (object, array, string, number, boolean, null)
func Type(value interface{}) string {
	if value == nil {
		return "null"
	}

	var parsed interface{}
	switch v := value.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return "unknown"
		}
	case []byte:
		if err := json.Unmarshal(v, &parsed); err != nil {
			return "unknown"
		}
	default:
		parsed = v
	}

	switch parsed.(type) {
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}
