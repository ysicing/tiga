package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/internal/models"
)

// TestUserContract_List tests GET /api/v1/database/instances/:id/users
func TestUserContract_List(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-user-list",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Create test user
	dbUser := &models.DatabaseUser{
		InstanceID:  instance.ID,
		Username:    "appuser",
		Password:    "encrypted",
		Host:        "%",
		Description: "Application user",
		IsActive:    true,
	}
	err = db.Create(dbUser).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/instances/1/users", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool), "Expected success=true")
	assert.NotNil(t, response["data"], "Expected data field")
}

// TestUserContract_Create tests POST /api/v1/database/instances/:id/users
func TestUserContract_Create(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-create-user",
		Type:     "postgresql",
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Request payload
	payload := map[string]interface{}{
		"username":    "newuser",
		"password":    "SecurePass123!",
		"host":        "%",
		"description": "New database user",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/database/instances/1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response structure
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	_, hasSuccess := response["success"]
	assert.True(t, hasSuccess, "Response should have 'success' field")
}

// TestUserContract_UpdatePassword tests PATCH /api/v1/database/users/:id
func TestUserContract_UpdatePassword(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance and user
	instance := &models.DatabaseInstance{
		Name:     "test-update-pwd",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	dbUser := &models.DatabaseUser{
		InstanceID: instance.ID,
		Username:   "changepass",
		Password:   "old-encrypted",
		IsActive:   true,
	}
	err = db.Create(dbUser).Error
	require.NoError(t, err)

	// Request payload
	payload := map[string]interface{}{
		"old_password": "OldPass123",
		"new_password": "NewPass456!",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/database/users/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response structure
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	_, hasSuccess := response["success"]
	assert.True(t, hasSuccess, "Response should have 'success' field")
}

// TestUserContract_Delete tests DELETE /api/v1/database/users/:id
func TestUserContract_Delete(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance and user
	instance := &models.DatabaseInstance{
		Name:     "test-delete-user",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	dbUser := &models.DatabaseUser{
		InstanceID: instance.ID,
		Username:   "deleteme",
		Password:   "encrypted",
		IsActive:   true,
	}
	err = db.Create(dbUser).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/database/users/1", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response structure
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	_, hasSuccess := response["success"]
	assert.True(t, hasSuccess, "Response should have 'success' field")
}
