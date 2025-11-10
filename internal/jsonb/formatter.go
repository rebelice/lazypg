package jsonb

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Format formats a JSONB value as a pretty-printed JSON string
func Format(value interface{}) (string, error) {
	if value == nil {
		return "null", nil
	}

	// Convert to JSON bytes
	var jsonBytes []byte
	switch v := value.(type) {
	case string:
		// Already a JSON string, parse it first
		var parsed interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}
		var err error
		jsonBytes, err = json.MarshalIndent(parsed, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
	case []byte:
		// JSON bytes, parse and re-format
		var parsed interface{}
		if err := json.Unmarshal(v, &parsed); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}
		var err error
		jsonBytes, err = json.MarshalIndent(parsed, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
	default:
		// Other types, marshal directly
		var err error
		jsonBytes, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to format: %w", err)
		}
	}

	return string(jsonBytes), nil
}

// Compact formats JSONB as compact (single-line) JSON
func Compact(value interface{}) (string, error) {
	if value == nil {
		return "null", nil
	}

	var jsonBytes []byte
	switch v := value.(type) {
	case string:
		var parsed interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}
		var err error
		jsonBytes, err = json.Marshal(parsed)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
	case []byte:
		var parsed interface{}
		if err := json.Unmarshal(v, &parsed); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}
		var err error
		jsonBytes, err = json.Marshal(parsed)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
	default:
		var err error
		jsonBytes, err = json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to compact: %w", err)
		}
	}

	return string(jsonBytes), nil
}

// Truncate truncates a JSON string for table display
func Truncate(jsonStr string, maxLen int) string {
	if len(jsonStr) <= maxLen {
		return jsonStr
	}

	// Try to truncate at a reasonable boundary
	truncated := jsonStr[:maxLen-3]

	// Find last space, comma, or bracket
	lastGood := strings.LastIndexAny(truncated, " ,{}[]")
	if lastGood > maxLen/2 {
		truncated = truncated[:lastGood]
	}

	return truncated + "..."
}
