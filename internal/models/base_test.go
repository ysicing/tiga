package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestModel is a test model using BaseModel
type TestModel struct {
	BaseModel
	Name string `gorm:"type:varchar(255)" json:"name"`
}

// TestModelWithoutSoftDelete is a test model without soft delete
type TestModelWithoutSoftDelete struct {
	BaseModelWithoutSoftDelete
	Token string `gorm:"type:varchar(255)" json:"token"`
}

// TestAppendOnlyModel is a test model for append-only data
type TestAppendOnlyModel struct {
	AppendOnlyModel
	Action string `gorm:"type:varchar(128)" json:"action"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Migrate test models
	err = db.AutoMigrate(&TestModel{}, &TestModelWithoutSoftDelete{}, &TestAppendOnlyModel{})
	assert.NoError(t, err)

	return db
}

func TestBaseModel_AutoGenerateUUID(t *testing.T) {
	db := setupTestDB(t)

	// Test BaseModel
	tm := &TestModel{Name: "Test"}
	err := db.Create(tm).Error
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tm.ID)
	assert.False(t, tm.CreatedAt.IsZero())
	assert.False(t, tm.UpdatedAt.IsZero())
}

func TestBaseModel_SoftDelete(t *testing.T) {
	db := setupTestDB(t)

	// Create a record
	tm := &TestModel{Name: "Test"}
	err := db.Create(tm).Error
	assert.NoError(t, err)

	// Soft delete
	err = db.Delete(tm).Error
	assert.NoError(t, err)

	// Should not find with normal query
	var found TestModel
	err = db.First(&found, "id = ?", tm.ID).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Should find with Unscoped
	err = db.Unscoped().First(&found, "id = ?", tm.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, tm.Name, found.Name)
	assert.False(t, found.DeletedAt.Time.IsZero())
}

func TestBaseModel_UpdatesTimestamp(t *testing.T) {
	db := setupTestDB(t)

	// Create a record
	tm := &TestModel{Name: "Original"}
	err := db.Create(tm).Error
	assert.NoError(t, err)

	originalUpdatedAt := tm.UpdatedAt

	// Update the record
	tm.Name = "Updated"
	err = db.Save(tm).Error
	assert.NoError(t, err)

	// UpdatedAt should have changed
	assert.True(t, tm.UpdatedAt.After(originalUpdatedAt))
}

func TestBaseModelWithoutSoftDelete_NoSoftDelete(t *testing.T) {
	db := setupTestDB(t)

	// Create a record
	tm := &TestModelWithoutSoftDelete{Token: "test-token"}
	err := db.Create(tm).Error
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tm.ID)

	// Delete (hard delete)
	err = db.Delete(tm).Error
	assert.NoError(t, err)

	// Should not find with normal query (hard deleted)
	var found TestModelWithoutSoftDelete
	err = db.First(&found, "id = ?", tm.ID).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Should not find even with Unscoped (hard deleted)
	err = db.Unscoped().First(&found, "id = ?", tm.ID).Error
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestAppendOnlyModel_CreatedAtOnly(t *testing.T) {
	db := setupTestDB(t)

	// Create a record
	tm := &TestAppendOnlyModel{Action: "login"}
	err := db.Create(tm).Error
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tm.ID)
	assert.False(t, tm.CreatedAt.IsZero())

	// Verify it was created
	var found TestAppendOnlyModel
	err = db.First(&found, "id = ?", tm.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "login", found.Action)
}

func TestBaseModel_BeforeCreateHook(t *testing.T) {
	db := setupTestDB(t)

	// Test with pre-set UUID
	presetID := uuid.New()
	tm := &TestModel{
		BaseModel: BaseModel{ID: presetID},
		Name:      "Test",
	}
	err := db.Create(tm).Error
	assert.NoError(t, err)
	assert.Equal(t, presetID, tm.ID, "Should preserve pre-set UUID")

	// Test with nil UUID (should auto-generate)
	tm2 := &TestModel{Name: "Test2"}
	err = db.Create(tm2).Error
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tm2.ID, "Should auto-generate UUID")
	assert.NotEqual(t, presetID, tm2.ID, "Should generate different UUID")
}
