package jsonb

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Path represents a JSON path (e.g., $.user.address.city)
type Path struct {
	Parts []string
}

// String returns the PostgreSQL JSONB path notation
func (p Path) String() string {
	if len(p.Parts) == 0 {
		return "$"
	}

	result := "$"
	for _, part := range p.Parts {
		// Check if part is numeric (array index)
		if _, err := strconv.Atoi(part); err == nil {
			result += "[" + part + "]"
		} else {
			result += "." + part
		}
	}
	return result
}

// PostgreSQLPath returns the PostgreSQL #> operator notation
func (p Path) PostgreSQLPath() string {
	if len(p.Parts) == 0 {
		return "{}"
	}
	return "{" + joinWithQuotes(p.Parts) + "}"
}

func joinWithQuotes(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ","
		}
		result += part
	}
	return result
}

// ExtractPaths extracts all possible paths from a JSONB value
func ExtractPaths(value interface{}) []Path {
	var paths []Path

	var parsed interface{}
	switch v := value.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return paths
		}
	case []byte:
		if err := json.Unmarshal(v, &parsed); err != nil {
			return paths
		}
	default:
		parsed = v
	}

	extractPathsRecursive(parsed, []string{}, &paths)
	return paths
}

func extractPathsRecursive(value interface{}, currentPath []string, paths *[]Path) {
	if value == nil {
		*paths = append(*paths, Path{Parts: currentPath})
		return
	}

	switch v := value.(type) {
	case map[string]interface{}:
		// Add path for the object itself
		*paths = append(*paths, Path{Parts: currentPath})

		// Recurse into each key
		for key, val := range v {
			newPath := append([]string{}, currentPath...)
			newPath = append(newPath, key)
			extractPathsRecursive(val, newPath, paths)
		}

	case []interface{}:
		// Add path for the array itself
		*paths = append(*paths, Path{Parts: currentPath})

		// Recurse into array elements (limit to first 5 for performance)
		limit := len(v)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			newPath := append([]string{}, currentPath...)
			newPath = append(newPath, strconv.Itoa(i))
			extractPathsRecursive(v[i], newPath, paths)
		}

	default:
		// Leaf value (string, number, boolean)
		*paths = append(*paths, Path{Parts: currentPath})
	}
}

// GetValueAtPath retrieves a value at a specific path
func GetValueAtPath(value interface{}, path Path) (interface{}, error) {
	var parsed interface{}
	switch v := value.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	case []byte:
		if err := json.Unmarshal(v, &parsed); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
	default:
		parsed = v
	}

	current := parsed
	for _, part := range path.Parts {
		switch curr := current.(type) {
		case map[string]interface{}:
			val, ok := curr[part]
			if !ok {
				return nil, fmt.Errorf("key '%s' not found", part)
			}
			current = val
		case []interface{}:
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", part)
			}
			if idx < 0 || idx >= len(curr) {
				return nil, fmt.Errorf("array index out of bounds: %d", idx)
			}
			current = curr[idx]
		default:
			return nil, fmt.Errorf("cannot traverse into %T", curr)
		}
	}

	return current, nil
}
