package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/pkg/crypto"
)

func TestCredentialEncryption(t *testing.T) {
	// Initialize crypto service for testing
	cryptoService, err := crypto.NewService("0123456789abcdef0123456789abcdef") // 32-byte key
	require.NoError(t, err)
	crypto.SetDefaultService(cryptoService)

	t.Run("PasswordEncryptionDecryption", func(t *testing.T) {
		password := "SuperSecret123!"

		// Encrypt
		encrypted, err := cryptoService.Encrypt(password)
		require.NoError(t, err)
		assert.NotEqual(t, password, encrypted, "Encrypted password should differ from plaintext")
		assert.NotEmpty(t, encrypted, "Encrypted password should not be empty")

		// Decrypt
		decrypted, err := cryptoService.Decrypt(encrypted)
		require.NoError(t, err)
		assert.Equal(t, password, decrypted, "Decrypted password should match original")
	})

	t.Run("DatabaseInstancePasswordEncryption", func(t *testing.T) {
		// Simulate DatabaseInstance password encryption
		instance := &models.DatabaseInstance{
			Name:     "test-instance",
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Password: "PlaintextPassword123",
		}

		// Encrypt password before save (simulating BeforeSave hook)
		encrypted, err := cryptoService.Encrypt(instance.Password)
		require.NoError(t, err)
		instance.Password = encrypted

		// Verify password is encrypted
		assert.NotEqual(t, "PlaintextPassword123", instance.Password)
		assert.NotContains(t, instance.Password, "Plaintext")

		// Decrypt for use (simulating AfterFind hook)
		decrypted, err := cryptoService.Decrypt(instance.Password)
		require.NoError(t, err)
		assert.Equal(t, "PlaintextPassword123", decrypted)
	})

	t.Run("DatabaseUserPasswordEncryption", func(t *testing.T) {
		// Simulate DatabaseUser password encryption
		user := &models.DatabaseUser{
			Username: "testuser",
			Password: "UserPassword456",
		}

		// Encrypt password
		encrypted, err := cryptoService.Encrypt(user.Password)
		require.NoError(t, err)
		user.Password = encrypted

		// Verify encryption
		assert.NotEqual(t, "UserPassword456", user.Password)

		// Decrypt and verify
		decrypted, err := cryptoService.Decrypt(user.Password)
		require.NoError(t, err)
		assert.Equal(t, "UserPassword456", decrypted)
	})

	t.Run("EncryptionWithDifferentKeys", func(t *testing.T) {
		password := "TestPassword"

		// Encrypt with first key
		service1, _ := crypto.NewService("key1key1key1key1key1key1key1key1") // 32 bytes
		encrypted1, err := service1.Encrypt(password)
		require.NoError(t, err)

		// Encrypt with second key
		service2, _ := crypto.NewService("key2key2key2key2key2key2key2key2") // 32 bytes
		encrypted2, err := service2.Encrypt(password)
		require.NoError(t, err)

		// Same password, different keys should produce different ciphertexts
		assert.NotEqual(t, encrypted1, encrypted2)

		// Each service can decrypt its own
		decrypted1, _ := service1.Decrypt(encrypted1)
		decrypted2, _ := service2.Decrypt(encrypted2)
		assert.Equal(t, password, decrypted1)
		assert.Equal(t, password, decrypted2)

		// Cross-decryption should fail
		_, err = service1.Decrypt(encrypted2)
		assert.Error(t, err, "Should not decrypt with wrong key")
	})

	t.Run("EmptyPasswordHandling", func(t *testing.T) {
		// Empty password should return empty
		encrypted, err := cryptoService.Encrypt("")
		assert.NoError(t, err)
		assert.Empty(t, encrypted)

		decrypted, err := cryptoService.Decrypt("")
		assert.NoError(t, err)
		assert.Empty(t, decrypted)
	})
}

func TestCredentialStorageSecurity(t *testing.T) {
	// This test would require actual database connection
	// For now, we test the encryption logic is properly integrated
	t.Run("PasswordNeverStoredInPlaintext", func(t *testing.T) {
		cryptoService, _ := crypto.NewService("testkeytest keytest keytest key") // 32 bytes
		crypto.SetDefaultService(cryptoService)

		plainPassword := "MySecretPassword123"

		// Simulate what happens in CreateInstance
		encrypted, err := cryptoService.Encrypt(plainPassword)
		require.NoError(t, err)

		// The encrypted value should NOT contain the plaintext
		assert.NotContains(t, encrypted, "MySecretPassword123")
		assert.NotContains(t, encrypted, "Secret")
		assert.NotContains(t, encrypted, "Password")

		// The encrypted value should be longer (includes IV and auth tag)
		assert.Greater(t, len(encrypted), len(plainPassword))
	})
}
