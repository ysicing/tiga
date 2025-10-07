package handlers

import (
	"fmt"

	"github.com/google/uuid"
)

// ParseUUID parses a UUID string
func ParseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID format: %w", err)
	}
	return id, nil
}

// ParseUUIDPtr parses a UUID string and returns a pointer
func ParseUUIDPtr(s string) (*uuid.UUID, error) {
	if s == "" {
		return nil, nil
	}
	id, err := ParseUUID(s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}

// deref returns the value of the pointer or the zero value if nil
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// contains checks if a slice contains a value
func contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// defaultIfEmpty returns the default value if the value is empty
func defaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// defaultInt returns the default value if the value is 0
func defaultInt(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// clamp clamps a value between min and max
func clamp(value, minValue, maxValue int) int {
	return max(minValue, min(value, maxValue))
}
