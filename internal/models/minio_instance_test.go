package models_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/db"
	m "github.com/ysicing/tiga/internal/models"
)

// This test verifies that SecretString is stored encrypted and decrypted on read
func TestSecretString_EncryptedAtRest(t *testing.T) {
	database, err := db.NewDatabase(&config.DatabaseConfig{Type: "sqlite", Name: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	require.NoError(t, database.AutoMigrate())

	// create owner user for Instance FK
	owner := &m.User{Username: "owner1", Email: "owner1@example.com", Password: "x"}
	require.NoError(t, database.DB.Create(owner).Error)

	// create instance owned by owner
	inst := &m.Instance{Name: "minio-test", Type: "minio", Connection: m.JSONB{"host": "localhost", "port": 9000}, OwnerID: owner.ID}
	require.NoError(t, database.DB.Create(inst).Error)

	// create minio user with encrypted secret
	u := &m.MinIOUser{InstanceID: inst.ID, Username: "u1", AccessKey: "u1", SecretKey: m.SecretString("s3cr3t"), Status: "enabled"}

	require.NoError(t, database.DB.Create(u).Error)

	// Read back raw stored value
	var raw string
	err = database.DB.Table("minio_users").Select("secret_key").Where("id = ?", u.ID).Scan(&raw).Error
	require.NoError(t, err)
	if raw == "s3cr3t" || raw == "" {
		t.Fatalf("expected encrypted secret in DB, got: %q", raw)
	}

	// Read through model and expect decrypted value
	var got m.MinIOUser
	require.NoError(t, database.DB.First(&got, "id = ?", u.ID).Error)
	if string(got.SecretKey) != "s3cr3t" {
		t.Fatalf("expected decrypted secret, got %q", string(got.SecretKey))
	}
}
