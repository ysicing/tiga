package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// TestNewPasswordHasher tests the creation of a password hasher with default cost
func TestNewPasswordHasher(t *testing.T) {
	hasher := NewPasswordHasher()

	assert.NotNil(t, hasher)
	assert.Equal(t, bcrypt.DefaultCost, hasher.GetCost())
}

// TestNewPasswordHasherWithCost tests the creation of a password hasher with custom cost
func TestNewPasswordHasherWithCost(t *testing.T) {
	testCases := []struct {
		name        string
		cost        int
		shouldError bool
	}{
		{"Valid Cost 4", bcrypt.MinCost, false},
		{"Valid Cost 10", 10, false},
		{"Valid Cost 31", bcrypt.MaxCost, false},
		{"Invalid Cost Too Low", 3, true},
		{"Invalid Cost Too High", 32, true},
		{"Invalid Cost Zero", 0, true},
		{"Invalid Cost Negative", -1, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasher, err := NewPasswordHasherWithCost(tc.cost)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, hasher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, hasher)
				assert.Equal(t, tc.cost, hasher.GetCost())
			}
		})
	}
}

// TestPasswordHasher_Hash tests password hashing
func TestPasswordHasher_Hash(t *testing.T) {
	hasher := NewPasswordHasher()

	testCases := []struct {
		name        string
		password    string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid Password",
			password:    "TestPassword123!",
			shouldError: false,
		},
		{
			name:        "Empty Password",
			password:    "",
			shouldError: true,
			errorMsg:    "password cannot be empty",
		},
		{
			name:        "Password Too Long",
			password:    strings.Repeat("a", 73),
			shouldError: true,
			errorMsg:    "password too long",
		},
		{
			name:        "Max Length Password",
			password:    strings.Repeat("a", 72),
			shouldError: false,
		},
		{
			name:        "Password With Special Characters",
			password:    "P@ssw0rd!#$%",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := hasher.Hash(tc.password)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Empty(t, hash)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				// Hash should start with $2a$ (bcrypt prefix)
				assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"))
			}
		})
	}
}

// TestPasswordHasher_Verify tests password verification
func TestPasswordHasher_Verify(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "TestPassword123!"

	// Generate hash
	hash, err := hasher.Hash(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	testCases := []struct {
		name        string
		password    string
		hash        string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid Password",
			password:    password,
			hash:        hash,
			shouldError: false,
		},
		{
			name:        "Wrong Password",
			password:    "WrongPassword123!",
			hash:        hash,
			shouldError: true,
			errorMsg:    "invalid password",
		},
		{
			name:        "Empty Password",
			password:    "",
			hash:        hash,
			shouldError: true,
			errorMsg:    "password cannot be empty",
		},
		{
			name:        "Empty Hash",
			password:    password,
			hash:        "",
			shouldError: true,
			errorMsg:    "hash cannot be empty",
		},
		{
			name:        "Invalid Hash Format",
			password:    password,
			hash:        "invalid-hash",
			shouldError: true,
			errorMsg:    "failed to verify password",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := hasher.Verify(tc.password, tc.hash)

			if tc.shouldError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPasswordHasher_NeedsRehash tests whether a hash needs to be regenerated
func TestPasswordHasher_NeedsRehash(t *testing.T) {
	// Create hasher with cost 10
	hasher10, _ := NewPasswordHasherWithCost(10)
	hash10, _ := hasher10.Hash("password")

	// Create hasher with cost 12
	hasher12, _ := NewPasswordHasherWithCost(12)
	_, _ = hasher12.Hash("password") // Create hash but don't use it

	testCases := []struct {
		name         string
		hasher       *PasswordHasher
		hash         string
		shouldRehash bool
	}{
		{
			name:         "Same Cost - No Rehash",
			hasher:       hasher10,
			hash:         hash10,
			shouldRehash: false,
		},
		{
			name:         "Different Cost - Needs Rehash",
			hasher:       hasher12,
			hash:         hash10,
			shouldRehash: true,
		},
		{
			name:         "Invalid Hash - Needs Rehash",
			hasher:       hasher10,
			hash:         "invalid-hash",
			shouldRehash: true,
		},
		{
			name:         "Empty Hash - Needs Rehash",
			hasher:       hasher10,
			hash:         "",
			shouldRehash: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			needsRehash := tc.hasher.NeedsRehash(tc.hash)
			assert.Equal(t, tc.shouldRehash, needsRehash)
		})
	}
}

// TestValidatePasswordStrength tests password strength validation
func TestValidatePasswordStrength(t *testing.T) {
	testCases := []struct {
		name        string
		password    string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid Strong Password",
			password:    "StrongP@ss123",
			shouldError: false,
		},
		{
			name:        "Too Short",
			password:    "Short1!",
			shouldError: true,
			errorMsg:    "at least 8 characters",
		},
		{
			name:        "Too Long",
			password:    "P@ssw0rd" + strings.Repeat("a", 65),
			shouldError: true,
			errorMsg:    "must not exceed 72 characters",
		},
		{
			name:        "No Uppercase",
			password:    "password123!",
			shouldError: true,
			errorMsg:    "uppercase letter",
		},
		{
			name:        "No Lowercase",
			password:    "PASSWORD123!",
			shouldError: true,
			errorMsg:    "lowercase letter",
		},
		{
			name:        "No Digit",
			password:    "Password!",
			shouldError: true,
			errorMsg:    "digit",
		},
		{
			name:        "No Special Character",
			password:    "Password123",
			shouldError: true,
			errorMsg:    "special character",
		},
		{
			name:        "All Requirements Met",
			password:    "MyP@ssw0rd",
			shouldError: false,
		},
		{
			name:        "With Multiple Special Chars",
			password:    "P@ssw0rd!#$%",
			shouldError: false,
		},
		{
			name:        "Exactly 8 Characters",
			password:    "Pass123!",
			shouldError: false,
		},
		{
			name:        "Exactly 72 Characters",
			password:    "P@ssw0rd" + strings.Repeat("a", 56) + "1A",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tc.password)

			if tc.shouldError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGlobalFunctions tests the global hash and verify functions
func TestGlobalFunctions(t *testing.T) {
	password := "TestPassword123!"

	// Test HashPassword
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Test VerifyPassword with correct password
	err = VerifyPassword(password, hash)
	assert.NoError(t, err)

	// Test VerifyPassword with wrong password
	err = VerifyPassword("WrongPassword", hash)
	assert.Error(t, err)
}

// TestPasswordHashConsistency tests that the same password produces different hashes (due to salt)
func TestPasswordHashConsistency(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "SamePassword123!"

	hash1, err1 := hasher.Hash(password)
	hash2, err2 := hasher.Hash(password)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEmpty(t, hash1)
	assert.NotEmpty(t, hash2)

	// Hashes should be different due to salt
	assert.NotEqual(t, hash1, hash2)

	// But both should verify correctly
	assert.NoError(t, hasher.Verify(password, hash1))
	assert.NoError(t, hasher.Verify(password, hash2))
}

// TestPasswordHasherConcurrency tests concurrent hashing
func TestPasswordHasherConcurrency(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "ConcurrentTest123!"

	// Run 10 concurrent hash operations
	results := make(chan string, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			hash, err := hasher.Hash(password)
			if err != nil {
				errors <- err
				return
			}
			results <- hash
		}()
	}

	// Collect results
	hashes := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		select {
		case hash := <-results:
			hashes = append(hashes, hash)
		case err := <-errors:
			t.Fatalf("Concurrent hashing failed: %v", err)
		}
	}

	// All hashes should be valid
	assert.Equal(t, 10, len(hashes))

	// All hashes should verify correctly
	for _, hash := range hashes {
		assert.NoError(t, hasher.Verify(password, hash))
	}
}

// BenchmarkPasswordHash benchmarks password hashing
func BenchmarkPasswordHash(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "BenchmarkPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Hash(password)
	}
}

// BenchmarkPasswordVerify benchmarks password verification
func BenchmarkPasswordVerify(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "BenchmarkPassword123!"
	hash, _ := hasher.Hash(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.Verify(password, hash)
	}
}

// BenchmarkPasswordStrengthValidation benchmarks password strength validation
func BenchmarkPasswordStrengthValidation(b *testing.B) {
	password := "BenchmarkP@ssw0rd123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePasswordStrength(password)
	}
}
