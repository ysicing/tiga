package cluster

import (
	"testing"
)

func TestSecureClear(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Clear sensitive kubeconfig data",
			input: "apiVersion: v1\nclusters:\n- cluster:\n    server: https://k8s.example.com\n    certificate-authority-data: SENSITIVE_CERT_DATA\n",
		},
		{
			name:  "Clear password",
			input: "password: SuperSecret123!@#",
		},
		{
			name:  "Clear token",
			input: "token: eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9...",
		},
		{
			name:  "Empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to byte slice
			data := []byte(tt.input)
			originalLen := len(data)

			// Clear the data
			secureClear(data)

			// Verify all bytes are zero
			for i, b := range data {
				if b != 0 {
					t.Errorf("secureClear() failed: byte at index %d is %d, expected 0", i, b)
				}
			}

			// Verify length didn't change
			if len(data) != originalLen {
				t.Errorf("secureClear() changed length: got %d, want %d", len(data), originalLen)
			}
		})
	}
}

func TestHashConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSame string // Another input that should produce the same hash
		wantDiff string // Another input that should produce a different hash
	}{
		{
			name:     "Same content produces same hash",
			input:    "apiVersion: v1\nkind: Config",
			wantSame: "apiVersion: v1\nkind: Config",
			wantDiff: "apiVersion: v1\nkind: ConfigMap",
		},
		{
			name:     "Whitespace matters",
			input:    "apiVersion: v1\n kind: Config",
			wantSame: "apiVersion: v1\n kind: Config",
			wantDiff: "apiVersion: v1\nkind: Config",
		},
		{
			name:     "Empty string",
			input:    "",
			wantSame: "",
			wantDiff: " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashConfig(tt.input)
			hash2 := hashConfig(tt.wantSame)
			hash3 := hashConfig(tt.wantDiff)

			// Verify hash is not empty and is hex string
			if len(hash1) != 64 { // SHA256 produces 64 hex characters
				t.Errorf("hashConfig() produced hash of length %d, expected 64", len(hash1))
			}

			// Verify same input produces same hash
			if hash1 != hash2 {
				t.Errorf("hashConfig() produced different hashes for same input:\nhash1=%s\nhash2=%s", hash1, hash2)
			}

			// Verify different input produces different hash
			if hash1 == hash3 {
				t.Errorf("hashConfig() produced same hash for different inputs:\nhash1=%s\nhash3=%s", hash1, hash3)
			}
		})
	}
}

func TestHashConfigDeterministic(t *testing.T) {
	// Verify that hashing is deterministic
	input := "apiVersion: v1\nclusters:\n- cluster:\n    server: https://kubernetes.default.svc\n"

	hash1 := hashConfig(input)
	hash2 := hashConfig(input)
	hash3 := hashConfig(input)

	if hash1 != hash2 || hash2 != hash3 {
		t.Errorf("hashConfig() is not deterministic:\nhash1=%s\nhash2=%s\nhash3=%s", hash1, hash2, hash3)
	}
}

func TestSecureClearMemoryProtection(t *testing.T) {
	// Test that sensitive data is actually cleared from memory
	sensitiveData := "super-secret-password-12345"
	data := []byte(sensitiveData)

	// Create a copy to verify original value
	original := make([]byte, len(data))
	copy(original, data)

	// Clear the data
	secureClear(data)

	// Verify data was modified
	if string(data) == string(original) {
		t.Error("secureClear() did not modify the data")
	}

	// Verify data is now all zeros
	for i, b := range data {
		if b != 0 {
			t.Errorf("secureClear() left non-zero byte at index %d: %d", i, b)
		}
	}
}
