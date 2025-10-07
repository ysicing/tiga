package crypto

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateKey tests key generation
func TestGenerateKey(t *testing.T) {
	testCases := []struct {
		name string
		runs int
	}{
		{
			name: "Generate Single Key",
			runs: 1,
		},
		{
			name: "Generate Multiple Keys",
			runs: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			keys := make(map[string]bool)

			for i := 0; i < tc.runs; i++ {
				key, err := GenerateKey()
				assert.NoError(t, err)
				assert.NotNil(t, key)
				assert.Equal(t, 32, len(key), "Key must be 32 bytes for AES-256")

				// Verify uniqueness
				keyStr := string(key)
				assert.False(t, keys[keyStr], "Keys should be unique")
				keys[keyStr] = true
			}
		})
	}
}

// TestGenerateKeyBase64 tests base64 key generation
func TestGenerateKeyBase64(t *testing.T) {
	key, err := GenerateKeyBase64()
	assert.NoError(t, err)
	assert.NotEmpty(t, key)

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(key)
	assert.NoError(t, err)
	assert.Equal(t, 32, len(decoded))
}

// TestLoadKeyFromBase64 tests loading key from base64 string
func TestLoadKeyFromBase64(t *testing.T) {
	// Generate a valid key
	originalKey, err := GenerateKey()
	require.NoError(t, err)
	keyBase64 := base64.StdEncoding.EncodeToString(originalKey)

	testCases := []struct {
		name        string
		keyString   string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid Base64 Key",
			keyString:   keyBase64,
			shouldError: false,
		},
		{
			name:        "Invalid Base64",
			keyString:   "not-valid-base64!!!",
			shouldError: true,
			errorMsg:    "failed to decode key",
		},
		{
			name:        "Wrong Key Length",
			keyString:   base64.StdEncoding.EncodeToString([]byte("short")),
			shouldError: true,
			errorMsg:    "must be 32 bytes",
		},
		{
			name:        "Empty String",
			keyString:   "",
			shouldError: true,
			errorMsg:    "must be 32 bytes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key, err := LoadKeyFromBase64(tc.keyString)

			if tc.shouldError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
				assert.Nil(t, key)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
				assert.Equal(t, 32, len(key))
			}
		})
	}
}

// TestNewEncryptionService tests encryption service creation
func TestNewEncryptionService(t *testing.T) {
	testCases := []struct {
		name        string
		key         []byte
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid 32-byte Key",
			key:         make([]byte, 32),
			shouldError: false,
		},
		{
			name:        "Too Short Key",
			key:         make([]byte, 16),
			shouldError: true,
			errorMsg:    "must be exactly 32 bytes",
		},
		{
			name:        "Too Long Key",
			key:         make([]byte, 64),
			shouldError: true,
			errorMsg:    "must be exactly 32 bytes",
		},
		{
			name:        "Empty Key",
			key:         []byte{},
			shouldError: true,
			errorMsg:    "must be exactly 32 bytes",
		},
		{
			name:        "Nil Key",
			key:         nil,
			shouldError: true,
			errorMsg:    "must be exactly 32 bytes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewEncryptionService(tc.key)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, service)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

// TestEncryptDecrypt tests encryption and decryption
func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "Simple Text",
			plaintext: "Hello, World!",
		},
		{
			name:      "Empty String",
			plaintext: "",
		},
		{
			name:      "Unicode Text",
			plaintext: "你好，世界！🌍",
		},
		{
			name:      "Long Text",
			plaintext: strings.Repeat("Lorem ipsum dolor sit amet. ", 100),
		},
		{
			name:      "Special Characters",
			plaintext: "!@#$%^&*()_+-={}[]|\\:\";<>?,./",
		},
		{
			name:      "JSON Data",
			plaintext: `{"username":"admin","password":"secret123","api_key":"sk-123456"}`,
		},
		{
			name:      "Multiline Text",
			plaintext: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := service.Encrypt(tc.plaintext)
			assert.NoError(t, err)

			if tc.plaintext == "" {
				assert.Empty(t, ciphertext)
			} else {
				assert.NotEmpty(t, ciphertext)
				assert.NotEqual(t, tc.plaintext, ciphertext)
				// Verify it's valid base64
				_, err := base64.StdEncoding.DecodeString(ciphertext)
				assert.NoError(t, err)
			}

			// Decrypt
			decrypted, err := service.Decrypt(ciphertext)
			assert.NoError(t, err)
			assert.Equal(t, tc.plaintext, decrypted)
		})
	}
}

// TestEncryptConsistency tests that same plaintext produces different ciphertexts
func TestEncryptConsistency(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	plaintext := "consistent test"

	// Encrypt same plaintext multiple times
	ciphertext1, err := service.Encrypt(plaintext)
	assert.NoError(t, err)

	ciphertext2, err := service.Encrypt(plaintext)
	assert.NoError(t, err)

	// Ciphertexts should be different (due to random nonce)
	assert.NotEqual(t, ciphertext1, ciphertext2)

	// But both should decrypt to same plaintext
	decrypted1, err := service.Decrypt(ciphertext1)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := service.Decrypt(ciphertext2)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

// TestDecryptErrors tests decryption error scenarios
func TestDecryptErrors(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		ciphertext string
		shouldFail bool
		errorMsg   string
	}{
		{
			name:       "Invalid Base64",
			ciphertext: "not-valid-base64!!!",
			shouldFail: true,
			errorMsg:   "failed to decode ciphertext",
		},
		{
			name:       "Too Short Ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString([]byte("short")),
			shouldFail: true,
			errorMsg:   "ciphertext too short",
		},
		{
			name:       "Corrupted Ciphertext",
			ciphertext: base64.StdEncoding.EncodeToString(make([]byte, 50)),
			shouldFail: true,
			errorMsg:   "failed to decrypt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decrypted, err := service.Decrypt(tc.ciphertext)

			if tc.shouldFail {
				assert.Error(t, err)
				assert.Empty(t, decrypted)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEncryptDecryptBytes tests byte encryption/decryption
func TestEncryptDecryptBytes(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "Simple Bytes",
			plaintext: []byte("Hello, World!"),
		},
		{
			name:      "Empty Bytes",
			plaintext: []byte{},
		},
		{
			name:      "Nil Bytes",
			plaintext: nil,
		},
		{
			name:      "Binary Data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := service.EncryptBytes(tc.plaintext)
			assert.NoError(t, err)

			if len(tc.plaintext) == 0 {
				assert.Nil(t, ciphertext)
			} else {
				assert.NotNil(t, ciphertext)
				assert.NotEqual(t, tc.plaintext, ciphertext)
			}

			// Decrypt
			decrypted, err := service.DecryptBytes(ciphertext)
			assert.NoError(t, err)
			// For empty/nil inputs, both should be nil or empty
			if len(tc.plaintext) == 0 {
				assert.Empty(t, decrypted)
			} else {
				assert.Equal(t, tc.plaintext, decrypted)
			}
		})
	}
}

// TestRotateKey tests key rotation
func TestRotateKey(t *testing.T) {
	oldKey, err := GenerateKey()
	require.NoError(t, err)

	oldService, err := NewEncryptionService(oldKey)
	require.NoError(t, err)

	// Rotate key
	newService, err := oldService.RotateKey()
	assert.NoError(t, err)
	assert.NotNil(t, newService)

	// Verify old and new services have different keys
	assert.NotEqual(t, oldService.key, newService.key)

	// Encrypt with old service
	plaintext := "test rotation"
	ciphertext, err := oldService.Encrypt(plaintext)
	require.NoError(t, err)

	// Should not decrypt with new service
	_, err = newService.Decrypt(ciphertext)
	assert.Error(t, err)

	// Should decrypt with old service
	decrypted, err := oldService.Decrypt(ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// TestReEncrypt tests re-encryption with new key
func TestReEncrypt(t *testing.T) {
	oldKey, err := GenerateKey()
	require.NoError(t, err)
	oldService, err := NewEncryptionService(oldKey)
	require.NoError(t, err)

	newKey, err := GenerateKey()
	require.NoError(t, err)
	newService, err := NewEncryptionService(newKey)
	require.NoError(t, err)

	plaintext := "sensitive data"

	// Encrypt with old key
	oldCiphertext, err := oldService.Encrypt(plaintext)
	require.NoError(t, err)

	// Re-encrypt with new key
	newCiphertext, err := ReEncrypt(oldService, newService, oldCiphertext)
	assert.NoError(t, err)
	assert.NotEmpty(t, newCiphertext)
	assert.NotEqual(t, oldCiphertext, newCiphertext)

	// Verify decryption with new key
	decrypted, err := newService.Decrypt(newCiphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// Old ciphertext should not decrypt with new key
	_, err = newService.Decrypt(oldCiphertext)
	assert.Error(t, err)
}

// TestDefaultService tests global default service
func TestDefaultService(t *testing.T) {
	// Reset default service
	defaultService = nil

	// Test before initialization
	_, err := Encrypt("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")

	_, err = Decrypt("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")

	// Initialize default service
	key, err := GenerateKey()
	require.NoError(t, err)

	err = InitDefaultService(key)
	assert.NoError(t, err)
	assert.NotNil(t, GetDefaultService())

	// Test encryption/decryption with default service
	plaintext := "test default service"
	ciphertext, err := Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := Decrypt(ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// Clean up
	defaultService = nil
}

// TestDefaultServiceFromBase64 tests initialization from base64
func TestDefaultServiceFromBase64(t *testing.T) {
	// Reset default service
	defaultService = nil

	// Generate base64 key
	keyBase64, err := GenerateKeyBase64()
	require.NoError(t, err)

	// Initialize from base64
	err = InitDefaultServiceFromBase64(keyBase64)
	assert.NoError(t, err)
	assert.NotNil(t, GetDefaultService())

	// Test functionality
	plaintext := "test base64 init"
	ciphertext, err := Encrypt(plaintext)
	assert.NoError(t, err)

	decrypted, err := Decrypt(ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// Test with invalid base64
	err = InitDefaultServiceFromBase64("invalid-base64")
	assert.Error(t, err)

	// Clean up
	defaultService = nil
}

// TestConcurrentEncryption tests concurrent encryption
func TestConcurrentEncryption(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	service, err := NewEncryptionService(key)
	require.NoError(t, err)

	const goroutines = 100
	results := make(chan string, goroutines)
	errors := make(chan error, goroutines)

	plaintext := "concurrent test"

	// Encrypt concurrently
	for i := 0; i < goroutines; i++ {
		go func() {
			ciphertext, err := service.Encrypt(plaintext)
			if err != nil {
				errors <- err
				return
			}
			results <- ciphertext
		}()
	}

	// Collect results
	ciphertexts := make([]string, 0, goroutines)
	for i := 0; i < goroutines; i++ {
		select {
		case ciphertext := <-results:
			ciphertexts = append(ciphertexts, ciphertext)
		case err := <-errors:
			t.Fatalf("Concurrent encryption failed: %v", err)
		}
	}

	// All should be valid and decrypt correctly
	assert.Equal(t, goroutines, len(ciphertexts))

	for _, ciphertext := range ciphertexts {
		decrypted, err := service.Decrypt(ciphertext)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	}
}

// BenchmarkEncrypt benchmarks encryption performance
func BenchmarkEncrypt(b *testing.B) {
	key, _ := GenerateKey()
	service, _ := NewEncryptionService(key)
	plaintext := "benchmark encryption test data"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Encrypt(plaintext)
	}
}

// BenchmarkDecrypt benchmarks decryption performance
func BenchmarkDecrypt(b *testing.B) {
	key, _ := GenerateKey()
	service, _ := NewEncryptionService(key)
	plaintext := "benchmark decryption test data"
	ciphertext, _ := service.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Decrypt(ciphertext)
	}
}

// BenchmarkGenerateKey benchmarks key generation
func BenchmarkGenerateKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateKey()
	}
}
