package config

import (
	"testing"
)

func TestIsWeakJWTSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expected bool
	}{
		{
			name:     "Common weak password 'secret'",
			secret:   "secret",
			expected: true,
		},
		{
			name:     "Common weak password 'password'",
			secret:   "password",
			expected: true,
		},
		{
			name:     "Case insensitive - 'SECRET'",
			secret:   "SECRET",
			expected: true,
		},
		{
			name:     "Weak repeated password",
			secret:   "secretsecret",
			expected: true,
		},
		{
			name:     "Low entropy - all same character",
			secret:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expected: true,
		},
		{
			name:     "Low entropy - mostly same character",
			secret:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaabbb",
			expected: true,
		},
		{
			name:     "Strong random secret",
			secret:   "xK9#mP2$nL5@qR8&vT3!wY6^zA1%bC4*",
			expected: false,
		},
		{
			name:     "Base64 encoded strong secret",
			secret:   "7v8x/9y+0z=1a-2b_3c~4d!5e@6f#7g$8h%9i^0j&",
			expected: false,
		},
		{
			name:     "UUID-like strong secret",
			secret:   "550e8400-e29b-41d4-a716-446655440000",
			expected: false,
		},
		{
			name:     "Empty string",
			secret:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWeakJWTSecret(tt.secret)
			if result != tt.expected {
				t.Errorf("isWeakJWTSecret(%q) = %v, want %v", tt.secret, result, tt.expected)
			}
		})
	}
}

func TestIsLowEntropy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "All same character (long)",
			input:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expected: true,
		},
		{
			name:     "Low unique characters (just at threshold)",
			input:    "aaaabbbbccccddddeeeeffffgggghhhh",
			expected: false, // 8 unique chars is acceptable (threshold is < 8)
		},
		{
			name:     "Single character dominates",
			input:    "aaaaaaaaaaaaaaaaaabcdefg",
			expected: true,
		},
		{
			name:     "Good entropy",
			input:    "abcdefghijklmnopqrstuvwxyz123456",
			expected: false,
		},
		{
			name:     "High entropy random",
			input:    "xK9#mP2$nL5@qR8&vT3!wY6^zA1%bC4*",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLowEntropy(tt.input)
			if result != tt.expected {
				t.Errorf("isLowEntropy(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfigValidate_JWTSecret(t *testing.T) {
	tests := []struct {
		name      string
		jwtSecret string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Empty JWT secret",
			jwtSecret: "",
			wantError: true,
			errorMsg:  "JWT_SECRET is not set",
		},
		{
			name:      "Too short JWT secret",
			jwtSecret: "short",
			wantError: true,
			errorMsg:  "JWT_SECRET must be at least 32 characters",
		},
		{
			name:      "Weak JWT secret",
			jwtSecret: "passwordpasswordpasswordpassword", // 32 chars but weak
			wantError: true,
			errorMsg:  "JWT_SECRET is too weak",
		},
		{
			name:      "Valid strong JWT secret",
			jwtSecret: "xK9#mP2$nL5@qR8&vT3!wY6^zA1%bC4*",
			wantError: false,
		},
		{
			name:      "Valid base64 JWT secret",
			jwtSecret: "7v8x/9y+0z=1a-2b_3c~4d!5e@6f#7g$8h%9i^0j&",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				JWT: JWTConfig{
					Secret: tt.jwtSecret,
				},
				Security: SecurityConfig{
					EncryptionKey: "", // Empty to avoid other validation errors
				},
				DatabaseManagement: DatabaseManagementConfig{
					CredentialKey: "",
				},
			}

			err := cfg.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errorMsg)
				} else if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOfSubstring(s, substr) >= 0))
}

// indexOfSubstring returns the index of the first instance of substr in s, or -1 if substr is not present
func indexOfSubstring(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
