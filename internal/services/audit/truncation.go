package audit

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

// TruncationResult represents the result of object truncation
// T018: Object truncation strategy implementation
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T018
//           .claude/specs/006-gitness-tiga/research.md Section 4 (Object Truncation)
type TruncationResult struct {
	TruncatedJSON      string   // Truncated JSON string
	WasTruncated       bool     // Whether truncation occurred
	TruncatedFields    []string // List of field paths that were truncated
	OriginalSize       int      // Original size in bytes
	TruncatedSize      int      // Final size in bytes
	StructurePreserved bool     // Whether JSON structure was preserved
}

const (
	// MaxObjectSize is the maximum size for audit objects (64KB)
	MaxObjectSize = 64 * 1024

	// FieldTruncationLimit is the size limit for individual fields (16KB)
	FieldTruncationLimit = 16 * 1024

	// TruncationMarker is appended to truncated strings
	TruncationMarker = "... [TRUNCATED]"
)

// TruncateObject intelligently truncates a JSON object to fit within 64KB limit
// T018: Smart truncation algorithm
//
// Strategy:
// 1. If object ≤ 64KB: No truncation
// 2. If object > 64KB: Field-level truncation
//    a. Identify large fields (>16KB)
//    b. Truncate large fields individually
//    c. Preserve small fields completely
//    d. Maintain JSON structure validity
//
// Returns:
// - Truncated JSON string
// - Truncation metadata (which fields were truncated)
func TruncateObject(obj interface{}) (*TruncationResult, error) {
	if obj == nil {
		return &TruncationResult{
			TruncatedJSON:      "",
			WasTruncated:       false,
			TruncatedFields:    []string{},
			OriginalSize:       0,
			TruncatedSize:      0,
			StructurePreserved: true,
		}, nil
	}

	// Marshal to JSON
	originalJSON, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object: %w", err)
	}

	originalSize := len(originalJSON)

	// If within limit, no truncation needed
	if originalSize <= MaxObjectSize {
		return &TruncationResult{
			TruncatedJSON:      string(originalJSON),
			WasTruncated:       false,
			TruncatedFields:    []string{},
			OriginalSize:       originalSize,
			TruncatedSize:      originalSize,
			StructurePreserved: true,
		}, nil
	}

	logrus.Debugf("Object size %d bytes exceeds limit, performing field-level truncation", originalSize)

	// Perform field-level truncation
	return truncateFields(originalJSON, originalSize)
}

// truncateFields performs field-level truncation on a JSON object
func truncateFields(originalJSON []byte, originalSize int) (*TruncationResult, error) {
	// Parse into map for field-level access
	var data map[string]interface{}
	if err := json.Unmarshal(originalJSON, &data); err != nil {
		// If not a map, fall back to simple truncation
		return simpleTruncate(originalJSON, originalSize), nil
	}

	truncatedFields := []string{}
	modified := false

	// Recursively truncate large fields
	truncateLargeFields(data, "", &truncatedFields, &modified)

	// If no modifications were made, use simple truncation
	if !modified {
		return simpleTruncate(originalJSON, originalSize), nil
	}

	// Re-marshal truncated object
	truncatedJSON, err := json.Marshal(data)
	if err != nil {
		logrus.Errorf("Failed to marshal truncated object: %v", err)
		return simpleTruncate(originalJSON, originalSize), nil
	}

	truncatedSize := len(truncatedJSON)

	// If still too large after field truncation, use simple truncation
	if truncatedSize > MaxObjectSize {
		logrus.Warnf("Field-level truncation insufficient (%d bytes), using simple truncation", truncatedSize)
		return simpleTruncate(originalJSON, originalSize), nil
	}

	return &TruncationResult{
		TruncatedJSON:      string(truncatedJSON),
		WasTruncated:       true,
		TruncatedFields:    truncatedFields,
		OriginalSize:       originalSize,
		TruncatedSize:      truncatedSize,
		StructurePreserved: true,
	}, nil
}

// truncateLargeFields recursively truncates large string fields in a map
func truncateLargeFields(data map[string]interface{}, prefix string, truncatedFields *[]string, modified *bool) {
	for key, value := range data {
		fieldPath := key
		if prefix != "" {
			fieldPath = prefix + "." + key
		}

		switch v := value.(type) {
		case string:
			// Truncate large string fields
			if len(v) > FieldTruncationLimit {
				truncated := v[:FieldTruncationLimit-len(TruncationMarker)] + TruncationMarker
				data[key] = truncated
				*truncatedFields = append(*truncatedFields, fieldPath)
				*modified = true
				logrus.Debugf("Truncated field %s from %d to %d bytes", fieldPath, len(v), len(truncated))
			}

		case map[string]interface{}:
			// Recursively handle nested maps
			truncateLargeFields(v, fieldPath, truncatedFields, modified)

		case []interface{}:
			// Handle arrays
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					arrayPath := fmt.Sprintf("%s[%d]", fieldPath, i)
					truncateLargeFields(itemMap, arrayPath, truncatedFields, modified)
				}
			}
		}
	}
}

// simpleTruncate performs simple truncation by cutting at MaxObjectSize
func simpleTruncate(originalJSON []byte, originalSize int) *TruncationResult {
	truncateAt := MaxObjectSize - len(TruncationMarker) - 2 // -2 for closing }

	// Try to truncate at a safe position (avoid breaking JSON structure)
	truncatedJSON := originalJSON[:truncateAt]

	// Append truncation marker and attempt to close JSON
	result := string(truncatedJSON) + TruncationMarker + "}"

	logrus.Warnf("Simple truncation applied: %d bytes -> %d bytes (structure may be incomplete)",
		originalSize, len(result))

	return &TruncationResult{
		TruncatedJSON:      result,
		WasTruncated:       true,
		TruncatedFields:    []string{"[entire object]"},
		OriginalSize:       originalSize,
		TruncatedSize:      len(result),
		StructurePreserved: false, // Simple truncation may break structure
	}
}

// TruncateString truncates a string value with marker
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < len(TruncationMarker) {
		return TruncationMarker
	}
	return s[:maxLen-len(TruncationMarker)] + TruncationMarker
}

// ValidateJSONStructure validates that a JSON string is well-formed
func ValidateJSONStructure(jsonStr string) bool {
	var data interface{}
	return json.Unmarshal([]byte(jsonStr), &data) == nil
}
