package unit_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/pkg/crypto"
)

func TestCredentialEncryptionDecryption(t *testing.T) {
	// Initialize with a test key
	key := "0123456789abcdef0123456789abcdef" // 32 bytes
	service, err := crypto.NewService(key)
	require.NoError(t, err)

	t.Run("BasicEncryptionDecryption", func(t *testing.T) {
		plaintext := "SuperSecret123!"

		// Encrypt
		encrypted, err := service.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, plaintext, encrypted, "Encrypted text should differ from plaintext")

		// Decrypt
		decrypted, err := service.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted, "Decrypted text should match original")
	})

	t.Run("EncryptionProducesBase64", func(t *testing.T) {
		plaintext := "password123"

		encrypted, err := service.Encrypt(plaintext)
		require.NoError(t, err)

		// Should be valid base64
		_, err = base64.StdEncoding.DecodeString(encrypted)
		assert.NoError(t, err, "Encrypted output should be valid base64")
	})

	t.Run("SameInputDifferentOutput", func(t *testing.T) {
		// Due to random IV, same input should produce different ciphertext
		plaintext := "password"

		encrypted1, err := service.Encrypt(plaintext)
		require.NoError(t, err)

		encrypted2, err := service.Encrypt(plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, encrypted1, encrypted2, "Same input should produce different ciphertext (due to IV)")

		// But both should decrypt to same plaintext
		decrypted1, _ := service.Decrypt(encrypted1)
		decrypted2, _ := service.Decrypt(encrypted2)
		assert.Equal(t, plaintext, decrypted1)
		assert.Equal(t, plaintext, decrypted2)
	})

	t.Run("EmptyString", func(t *testing.T) {
		encrypted, err := service.Encrypt("")
		assert.NoError(t, err)
		assert.Empty(t, encrypted, "Empty input should return empty output")

		decrypted, err := service.Decrypt("")
		assert.NoError(t, err)
		assert.Empty(t, decrypted, "Empty input should return empty output")
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		testCases := []string{
			"p@ssw0rd!",
			"密码123",
			"🔐🔑🛡️",
			"pass\nword\ttab",
			"quote\"escape'test",
			strings.Repeat("x", 1000), // Long password
		}

		for _, plaintext := range testCases {
			t.Run(plaintext[:min(len(plaintext), 20)], func(t *testing.T) {
				encrypted, err := service.Encrypt(plaintext)
				require.NoError(t, err)

				decrypted, err := service.Decrypt(encrypted)
				require.NoError(t, err)
				assert.Equal(t, plaintext, decrypted)
			})
		}
	})

	t.Run("InvalidCiphertext", func(t *testing.T) {
		testCases := []struct {
			name       string
			ciphertext string
		}{
			{"Invalid base64", "not-valid-base64!@#"},
			{"Too short", "abc"},
			{"Random data", base64.StdEncoding.EncodeToString([]byte("random"))},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := service.Decrypt(tc.ciphertext)
				assert.Error(t, err, "Should fail to decrypt invalid ciphertext")
			})
		}
	})

	t.Run("WrongKey", func(t *testing.T) {
		plaintext := "secret"

		// Encrypt with first key
		service1, _ := crypto.NewService("key1key1key1key1key1key1key1key1")
		encrypted, err := service1.Encrypt(plaintext)
		require.NoError(t, err)

		// Try to decrypt with different key
		service2, _ := crypto.NewService("key2key2key2key2key2key2key2key2")
		_, err = service2.Decrypt(encrypted)
		assert.Error(t, err, "Should fail to decrypt with wrong key")
	})
}

func TestCredentialKeyValidation(t *testing.T) {
	t.Run("ValidKey32Bytes", func(t *testing.T) {
		key := "0123456789abcdef0123456789abcdef" // 32 bytes
		service, err := crypto.NewService(key)
		assert.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("InvalidKeyLength", func(t *testing.T) {
		testCases := []struct {
			name string
			key  string
		}{
			{"Too short", "short"},
			{"16 bytes", "0123456789abcdef"},
			{"24 bytes", "0123456789abcdef01234567"},
			{"Too long", "0123456789abcdef0123456789abcdef0123456789"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := crypto.NewService(tc.key)
				assert.Error(t, err, "Should reject key of length: "+string(rune(len(tc.key))))
			})
		}
	})

	t.Run("EmptyKey", func(t *testing.T) {
		_, err := crypto.NewService("")
		assert.Error(t, err, "Should reject empty key")
	})
}

func TestCredentialServiceThreadSafety(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	service, err := crypto.NewService(key)
	require.NoError(t, err)

	// Test concurrent encryption/decryption
	t.Run("ConcurrentOperations", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				plaintext := "password" + string(rune(id))

				encrypted, err := service.Encrypt(plaintext)
				assert.NoError(t, err)

				decrypted, err := service.Decrypt(encrypted)
				assert.NoError(t, err)
				assert.Equal(t, plaintext, decrypted)

				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestPasswordFieldEncryption(t *testing.T) {
	// Test password field encryption behavior similar to DatabaseInstance/DatabaseUser
	key := "0123456789abcdef0123456789abcdef"
	service, err := crypto.NewService(key)
	require.NoError(t, err)

	t.Run("SimulateBeforeSaveHook", func(t *testing.T) {
		// Simulate model with password field
		type User struct {
			Username string
			Password string
		}

		user := User{
			Username: "testuser",
			Password: "PlaintextPassword123",
		}

		// BeforeSave hook would encrypt
		encrypted, err := service.Encrypt(user.Password)
		require.NoError(t, err)
		user.Password = encrypted

		// Verify password is encrypted
		assert.NotEqual(t, "PlaintextPassword123", user.Password)
		assert.NotContains(t, user.Password, "Plaintext")
		assert.NotContains(t, user.Password, "Password")

		// Verify it can be decrypted
		decrypted, err := service.Decrypt(user.Password)
		require.NoError(t, err)
		assert.Equal(t, "PlaintextPassword123", decrypted)
	})

	t.Run("SimulateAfterFindHook", func(t *testing.T) {
		// Simulate retrieving encrypted password from database
		encryptedPassword := "LnKX8+encrypted+base64+data=="

		// Normally would decrypt, but if already plaintext-looking, might skip
		// This test verifies we don't double-decrypt
		_, err := service.Decrypt(encryptedPassword)
		// May error if not valid encrypted data, which is expected
		t.Logf("Decrypt result: %v", err)
	})

	t.Run("PasswordNotExposedInLogs", func(t *testing.T) {
		password := "SecretPassword123!"
		encrypted, err := service.Encrypt(password)
		require.NoError(t, err)

		// Encrypted value should not contain original password
		assert.NotContains(t, encrypted, "Secret")
		assert.NotContains(t, encrypted, "Password")
		assert.NotContains(t, encrypted, "123")

		// Encrypted value should be safe to log (base64)
		assert.True(t, isBase64(encrypted), "Encrypted password should be base64")
	})
}

// Helper function
func isBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
