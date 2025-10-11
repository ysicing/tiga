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

// TestPermissionContract_Grant tests POST /api/v1/database/permissions
func TestPermissionContract_Grant(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance, database, and user
	instance := &models.DatabaseInstance{
		Name:     "test-perm-grant",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	database := &models.Database{
		InstanceID: instance.ID,
		Name:       "appdb",
		Charset:    "utf8mb4",
	}
	err = db.Create(database).Error
	require.NoError(t, err)

	dbUser := &models.DatabaseUser{
		InstanceID: instance.ID,
		Username:   "appuser",
		Password:   "encrypted",
		IsActive:   true,
	}
	err = db.Create(dbUser).Error
	require.NoError(t, err)

	// Request payload
	payload := map[string]interface{}{
		"user_id":     1,
		"database_id": 1,
		"role":        "readonly",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/database/permissions", bytes.NewReader(body))
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

// TestPermissionContract_Revoke tests DELETE /api/v1/database/permissions/:id
func TestPermissionContract_Revoke(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test data
	instance := &models.DatabaseInstance{
		Name:     "test-perm-revoke",
		Type:     "postgresql",
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	database := &models.Database{
		InstanceID: instance.ID,
		Name:       "proddb",
	}
	err = db.Create(database).Error
	require.NoError(t, err)

	dbUser := &models.DatabaseUser{
		InstanceID: instance.ID,
		Username:   "devuser",
		Password:   "encrypted",
		IsActive:   true,
	}
	err = db.Create(dbUser).Error
	require.NoError(t, err)

	permission := &models.PermissionPolicy{
		UserID:     dbUser.ID,
		DatabaseID: database.ID,
		Role:       "readonly",
		GrantedBy:  "admin",
	}
	err = db.Create(permission).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/database/permissions/1", nil)
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

// TestPermissionContract_ListUserPermissions tests GET /api/v1/database/users/:id/permissions
func TestPermissionContract_ListUserPermissions(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test data
	instance := &models.DatabaseInstance{
		Name:     "test-perm-list",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	database := &models.Database{
		InstanceID: instance.ID,
		Name:       "testdb",
	}
	err = db.Create(database).Error
	require.NoError(t, err)

	dbUser := &models.DatabaseUser{
		InstanceID: instance.ID,
		Username:   "testuser",
		Password:   "encrypted",
		IsActive:   true,
	}
	err = db.Create(dbUser).Error
	require.NoError(t, err)

	permission := &models.PermissionPolicy{
		UserID:     dbUser.ID,
		DatabaseID: database.ID,
		Role:       "readwrite",
		GrantedBy:  "admin",
	}
	err = db.Create(permission).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/users/1/permissions", nil)
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
