package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Service is an alias for EncryptionService for compatibility
type Service = EncryptionService

// EncryptionService handles application-layer encryption/decryption
type EncryptionService struct {
	key []byte // 32 bytes for AES-256
}

// NewService creates a new encryption service with a key string (must be 32 bytes)
// This is a compatibility wrapper for NewEncryptionService
func NewService(key string) (*Service, error) {
	return NewEncryptionService([]byte(key))
}

// SetDefaultService sets the default encryption service
// This is a compatibility wrapper for setting defaultService
func SetDefaultService(service *Service) {
	defaultService = service
}

// NewEncryptionService creates a new encryption service with a key
func NewEncryptionService(key []byte) (*EncryptionService, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes for AES-256")
	}

	return &EncryptionService{
		key: key,
	}, nil
}

// GenerateKey generates a new 256-bit encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}
	return key, nil
}

// GenerateKeyBase64 generates a new key and returns it as base64 string
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// LoadKeyFromBase64 loads a key from base64 string
func LoadKeyFromBase64(keyString string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(keyString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("decoded key must be 32 bytes, got %d", len(key))
	}
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Create AES cipher
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and seal
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encoded, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (s *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptBytes encrypts byte data
func (s *EncryptionService) EncryptBytes(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBytes decrypts byte data
func (s *EncryptionService) DecryptBytes(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, nil
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// RotateKey creates a new encryption service with a new key
// and provides both old and new services for data migration
func (s *EncryptionService) RotateKey() (*EncryptionService, error) {
	newKey, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	return NewEncryptionService(newKey)
}

// ReEncrypt re-encrypts data with a new key
func ReEncrypt(oldService, newService *EncryptionService, ciphertext string) (string, error) {
	// Decrypt with old key
	plaintext, err := oldService.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt with old key: %w", err)
	}

	// Encrypt with new key
	newCiphertext, err := newService.Encrypt(plaintext)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt with new key: %w", err)
	}

	return newCiphertext, nil
}

// Global default encryption service (should be initialized at startup)
var defaultService *EncryptionService

// InitDefaultService initializes the default encryption service
func InitDefaultService(key []byte) error {
	service, err := NewEncryptionService(key)
	if err != nil {
		return err
	}
	defaultService = service
	return nil
}

// InitDefaultServiceFromBase64 initializes from base64 key string
func InitDefaultServiceFromBase64(keyString string) error {
	key, err := LoadKeyFromBase64(keyString)
	if err != nil {
		return err
	}
	return InitDefaultService(key)
}

// Encrypt encrypts using the default service
func Encrypt(plaintext string) (string, error) {
	if defaultService == nil {
		return "", fmt.Errorf("default encryption service not initialized")
	}
	return defaultService.Encrypt(plaintext)
}

// Decrypt decrypts using the default service
func Decrypt(ciphertext string) (string, error) {
	if defaultService == nil {
		return "", fmt.Errorf("default encryption service not initialized")
	}
	return defaultService.Decrypt(ciphertext)
}

// GetDefaultService returns the default encryption service
func GetDefaultService() *EncryptionService {
	return defaultService
}
