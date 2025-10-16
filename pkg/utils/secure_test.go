package utils

import "testing"

func TestEncryptDecryptStringWithKey(t *testing.T) {
	original := "Hello, World!asdas"
	testKey := "test-encryption-key-32-bytes!"
	encrypted := EncryptStringWithKey(original, testKey)
	decrypted, err := DecryptStringWithKey(encrypted, testKey)
	t.Log("Encrypted:", encrypted)
	t.Log("Decrypted:", decrypted)
	if err != nil {
		t.Fatalf("DecryptStringWithKey() failed: %v", err)
	}
	if decrypted != original {
		t.Errorf("DecryptStringWithKey() = %q, want %q", decrypted, original)
	}
}
