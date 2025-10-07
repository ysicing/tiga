package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EncryptionService handles encryption and decryption of sensitive data
type EncryptionService struct {
	keyPath string
	key     []byte
	gcm     cipher.AEAD
}

// NewEncryptionService creates a new encryption service with AES-GCM
func NewEncryptionService(keyPath string) (*EncryptionService, error) {
	service := &EncryptionService{
		keyPath: keyPath,
	}

	// Load or generate encryption key
	key, err := service.loadOrGenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encryption key: %w", err)
	}

	service.key = key

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	service.gcm = gcm

	return service, nil
}

// loadOrGenerateKey loads existing key or generates a new one
func (s *EncryptionService) loadOrGenerateKey() ([]byte, error) {
	// Try to load existing key
	if _, err := os.Stat(s.keyPath); err == nil {
		keyData, err := os.ReadFile(s.keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}

		key, err := base64.StdEncoding.DecodeString(string(keyData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode key: %w", err)
		}

		if len(key) != 32 { // AES-256 requires 32 bytes
			return nil, errors.New("invalid key length, expected 32 bytes")
		}

		return key, nil
	}

	// Generate new key
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Save key to file
	if err := s.saveKey(key); err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	return key, nil
}

// saveKey saves the encryption key to file with secure permissions
func (s *EncryptionService) saveKey(key []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(s.keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Encode key as base64
	encodedKey := base64.StdEncoding.EncodeToString(key)

	// Write key with secure permissions (0600 = owner read/write only)
	if err := os.WriteFile(s.keyPath, []byte(encodedKey), 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// Encrypt encrypts plaintext data using AES-GCM
func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Generate nonce
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := s.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode as base64
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return encoded, nil
}

// Decrypt decrypts ciphertext data using AES-GCM
func (s *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Decode from base64
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Check minimum length (nonce + data)
	nonceSize := s.gcm.NonceSize()
	if len(decoded) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, cipherData := decoded[:nonceSize], decoded[nonceSize:]

	// Decrypt data
	plaintext, err := s.gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptBytes encrypts binary data
func (s *EncryptionService) EncryptBytes(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	// Generate nonce
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := s.gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// DecryptBytes decrypts binary data
func (s *EncryptionService) DecryptBytes(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, nil
	}

	// Check minimum length
	nonceSize := s.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, cipherData := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt data
	plaintext, err := s.gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// RotateKey generates a new encryption key and re-encrypts all data
// This should be called periodically for security
func (s *EncryptionService) RotateKey() error {
	// Generate new key
	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Create new cipher
	block, err := aes.NewCipher(newKey)
	if err != nil {
		return fmt.Errorf("failed to create new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create new GCM: %w", err)
	}

	// Backup old key
	backupPath := s.keyPath + ".old"
	if err := os.Rename(s.keyPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup old key: %w", err)
	}

	// Save new key
	if err := s.saveKey(newKey); err != nil {
		// Restore backup on failure
		_ = os.Rename(backupPath, s.keyPath)
		return fmt.Errorf("failed to save new key: %w", err)
	}

	// Update service
	s.key = newKey
	s.gcm = gcm

	// Remove backup
	_ = os.Remove(backupPath)

	return nil
}

// VerifyIntegrity verifies that encryption/decryption works correctly
func (s *EncryptionService) VerifyIntegrity() error {
	testData := "test-encryption-integrity-check"

	encrypted, err := s.Encrypt(testData)
	if err != nil {
		return fmt.Errorf("encryption test failed: %w", err)
	}

	decrypted, err := s.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("decryption test failed: %w", err)
	}

	if decrypted != testData {
		return errors.New("integrity check failed: decrypted data does not match original")
	}

	return nil
}
