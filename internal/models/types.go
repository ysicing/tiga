package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ysicing/tiga/pkg/utils"
)

// JSONB is stored as TEXT in all databases (JSON string)
type JSONB map[string]interface{}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("failed to unmarshal JSONB value")
	}

	result := make(JSONB)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(make(JSONB))
	}
	return json.Marshal(j)
}

// String returns the JSON string representation
func (j JSONB) String() string {
	bytes, err := json.Marshal(j)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

// StringArray is stored as TEXT in all databases (JSON array string)
type StringArray []string

// Scan implements the sql.Scanner interface for StringArray
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}

	// Parse as JSON array
	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface for StringArray
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

// EncryptedField is a custom type for encrypted database fields
// Encryption/decryption will be implemented in a separate crypto package
type EncryptedField struct {
	Plaintext string // Decrypted value
	Encrypted bool   // Whether the value is encrypted in database
}

// Scan implements the sql.Scanner interface for EncryptedField
func (e *EncryptedField) Scan(value interface{}) error {
	if value == nil {
		e.Plaintext = ""
		e.Encrypted = false
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return errors.New("failed to scan EncryptedField value")
		}
		bytes = []byte(str)
	}

	// TODO: Implement decryption using application-layer encryption
	// For now, store as-is (will be implemented in pkg/crypto)
	e.Plaintext = string(bytes)
	e.Encrypted = true
	return nil
}

// Value implements the driver.Valuer interface for EncryptedField
func (e EncryptedField) Value() (driver.Value, error) {
	if e.Plaintext == "" {
		return nil, nil
	}

	// TODO: Implement encryption using application-layer encryption
	// For now, store as-is (will be implemented in pkg/crypto)
	return e.Plaintext, nil
}

// String returns the decrypted string value
func (e EncryptedField) String() string {
	return e.Plaintext
}

// MarshalJSON implements json.Marshaler
func (e EncryptedField) MarshalJSON() ([]byte, error) {
	// Never expose encrypted values in JSON
	return json.Marshal("***")
}

// UnmarshalJSON implements json.Unmarshaler
func (e *EncryptedField) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	e.Plaintext = str
	e.Encrypted = false
	return nil
}

// SecretString is a custom type for encrypted secrets in database
type SecretString string

// Scan implements the sql.Scanner interface for reading encrypted data from database
func (s *SecretString) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	var encryptedStr string
	switch v := value.(type) {
	case string:
		encryptedStr = v
	case []byte:
		encryptedStr = string(v)
	default:
		return fmt.Errorf("cannot scan %T into SecretString", value)
	}
	// If the string is empty, just set it directly
	if encryptedStr == "" {
		*s = ""
		return nil
	}

	// Decrypt the string
	decrypted, err := utils.DecryptString(encryptedStr)
	if err != nil {
		return fmt.Errorf("failed to decrypt SecretString: %w", err)
	}
	*s = SecretString(decrypted)
	return nil
}

// Value implements the driver.Valuer interface for writing encrypted data to database
func (s SecretString) Value() (driver.Value, error) {
	if s == "" {
		return "", nil
	}
	encrypted := utils.EncryptString(string(s))
	if len(encrypted) > 17 && encrypted[:17] == "encryption_error:" {
		return nil, fmt.Errorf("encryption failed: %s", encrypted[17:])
	}
	return encrypted, nil
}

// LowerCaseString is a custom type that automatically converts to lowercase
type LowerCaseString string

func (s *LowerCaseString) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan %T into LowerCaseString", value)
	}
	*s = LowerCaseString(strings.ToLower(str))
	return nil
}

func (s LowerCaseString) Value() (driver.Value, error) {
	return strings.ToLower(string(s)), nil
}

// SliceString is a custom type for comma-separated strings stored as TEXT
type SliceString []string

func (s *SliceString) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var strArray []string
	switch v := value.(type) {
	case string:
		if v == "" {
			*s = []string{}
			return nil
		}
		strArray = strings.Split(v, ",")
	case []byte:
		if len(v) == 0 {
			*s = []string{}
			return nil
		}
		strArray = strings.Split(string(v), ",")
	default:
		return fmt.Errorf("cannot scan %T into SliceString", value)
	}
	*s = SliceString(strArray)
	return nil
}

func (s SliceString) Value() (driver.Value, error) {
	if s == nil || len(s) == 0 {
		return "", nil
	}
	return strings.Join(s, ","), nil
}
