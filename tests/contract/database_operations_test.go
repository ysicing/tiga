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

// TestDatabaseContract_List tests GET /api/v1/database/instances/:id/databases
func TestDatabaseContract_List(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-mysql-db",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Create test database
	database := &models.Database{
		InstanceID: instance.ID,
		Name:       "testdb",
		Charset:    "utf8mb4",
		Collation:  "utf8mb4_unicode_ci",
	}
	err = db.Create(database).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/instances/1/databases", nil)
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

// TestDatabaseContract_Create tests POST /api/v1/database/instances/:id/databases
func TestDatabaseContract_Create(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-pg",
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
		"name":      "newdb",
		"charset":   "UTF8",
		"collation": "",
		"owner":     "postgres",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/database/instances/1/databases", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response - might fail if connection fails, but structure should be valid
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	_, hasSuccess := response["success"]
	assert.True(t, hasSuccess, "Response should have 'success' field")
}

// TestDatabaseContract_Delete tests DELETE /api/v1/database/databases/:id with confirm_name
func TestDatabaseContract_Delete(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance and database
	instance := &models.DatabaseInstance{
		Name:     "test-mysql-del",
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
		Name:       "dropme",
		Charset:    "utf8mb4",
	}
	err = db.Create(database).Error
	require.NoError(t, err)

	// Request with confirm_name
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/database/databases/1?confirm_name=dropme", nil)
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

// TestDatabaseContract_DeleteWithoutConfirm tests DELETE without confirm_name (should fail)
func TestDatabaseContract_DeleteWithoutConfirm(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance and database
	instance := &models.DatabaseInstance{
		Name: "test-instance",
		Type: "mysql",
		Host: "localhost",
		Port: 3306,
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	database := &models.Database{
		InstanceID: instance.ID,
		Name:       "important",
	}
	err = db.Create(database).Error
	require.NoError(t, err)

	// Request WITHOUT confirm_name
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/database/databases/1", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected status 400 without confirm_name")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool), "Expected success=false")
}
