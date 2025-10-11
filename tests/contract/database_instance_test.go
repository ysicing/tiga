package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/api"
	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/db"
	"github.com/ysicing/tiga/internal/models"
)

// setupTestAPI creates a test router with database
func setupTestAPI(t *testing.T) (*gin.Engine, *gorm.DB, func()) {
	// Create in-memory SQLite database for testing
	cfg := &config.DatabaseConfig{
		Type: "sqlite",
		Name: ":memory:",
	}

	database, err := db.NewDatabase(cfg)
	require.NoError(t, err)

	// Run migrations
	err = database.AutoMigrate()
	require.NoError(t, err)

	// Create test admin user
	adminUser := &models.User{
		Username: "testadmin",
		Email:    "admin@test.com",
		Password: "hashedpassword",
	}
	err = database.DB.Create(adminUser).Error
	require.NoError(t, err)

	// Setup router
	appConfig := &config.Config{
		Server: config.ServerConfig{
			Debug: true,
		},
		DatabaseManagement: config.DatabaseManagementConfig{
			CredentialKey:       "test-key-32-bytes-long-exactly!",
			QueryTimeoutSeconds: 30,
			MaxResultBytes:      10 * 1024 * 1024,
			AuditRetentionDays:  90,
		},
	}

	router, err := api.SetupRouter(appConfig, database)
	require.NoError(t, err)

	cleanup := func() {
		sqlDB, _ := database.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return router, database.DB, cleanup
}

// TestInstanceContract_List tests GET /api/v1/database/instances
func TestInstanceContract_List(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:        "test-mysql",
		Type:        "mysql",
		Host:        "localhost",
		Port:        3306,
		Username:    "root",
		Password:    "encrypted-password",
		Description: "Test MySQL instance",
		Status:      "pending",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/instances", nil)
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

// TestInstanceContract_Create tests POST /api/v1/database/instances
func TestInstanceContract_Create(t *testing.T) {
	router, _, cleanup := setupTestAPI(t)
	defer cleanup()

	payload := map[string]interface{}{
		"name":        "new-postgres",
		"type":        "postgresql",
		"host":        "localhost",
		"port":        5432,
		"username":    "postgres",
		"password":    "secret123",
		"description": "New PostgreSQL instance",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/database/instances", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code, "Expected status 201")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool), "Expected success=true")
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "new-postgres", data["name"])
	assert.Equal(t, "postgresql", data["type"])
}

// TestInstanceContract_GetByID tests GET /api/v1/database/instances/:id
func TestInstanceContract_GetByID(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-redis",
		Type:     "redis",
		Host:     "localhost",
		Port:     6379,
		Username: "default",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/instances/1", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test-redis", data["name"])
}

// TestInstanceContract_Delete tests DELETE /api/v1/database/instances/:id
func TestInstanceContract_Delete(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name: "to-delete",
		Type: "mysql",
		Host: "localhost",
		Port: 3306,
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/database/instances/1", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
}

// TestInstanceContract_TestConnection tests POST /api/v1/database/instances/:id/test
func TestInstanceContract_TestConnection(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-conn",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Status:   "pending",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/database/instances/1/test", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Response code may be 200 (success) or 400/500 (connection failed)
	// We just verify the response structure
	assert.True(t, w.Code == http.StatusOK || w.Code >= 400, "Expected valid response code")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Response should have success field
	_, hasSuccess := response["success"]
	assert.True(t, hasSuccess, "Response should have 'success' field")
}
