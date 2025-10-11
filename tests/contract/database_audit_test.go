package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/internal/models"
)

// TestAuditContract_List tests GET /api/v1/database/audit-logs with pagination
func TestAuditContract_List(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-audit-instance",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Create test audit logs
	logs := []models.DatabaseAuditLog{
		{
			InstanceID: &instance.ID,
			Operator:   "admin",
			Action:     "instance.create",
			TargetType: "instance",
			TargetName: "test-mysql",
			Success:    true,
			ClientIP:   "127.0.0.1",
		},
		{
			InstanceID: &instance.ID,
			Operator:   "admin",
			Action:     "database.create",
			TargetType: "database",
			TargetName: "testdb",
			Success:    true,
			ClientIP:   "127.0.0.1",
		},
	}
	for i := range logs {
		err = db.Create(&logs[i]).Error
		require.NoError(t, err)
	}

	// Make request with pagination
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/audit-logs?page=1&page_size=10", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool), "Expected success=true")
	data := response["data"].(map[string]interface{})

	// Verify pagination fields
	_, hasLogs := data["logs"]
	_, hasTotal := data["total"]
	_, hasPage := data["page"]
	_, hasPageSize := data["page_size"]

	assert.True(t, hasLogs, "Response should have 'logs' field")
	assert.True(t, hasTotal, "Response should have 'total' field")
	assert.True(t, hasPage, "Response should have 'page' field")
	assert.True(t, hasPageSize, "Response should have 'page_size' field")
}

// TestAuditContract_FilterByInstance tests filtering audit logs by instance_id
func TestAuditContract_FilterByInstance(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create two test instances
	instance1 := &models.DatabaseInstance{
		Name: "instance1",
		Type: "mysql",
		Host: "localhost",
		Port: 3306,
	}
	err := db.Create(instance1).Error
	require.NoError(t, err)

	instance2 := &models.DatabaseInstance{
		Name: "instance2",
		Type: "postgresql",
		Host: "localhost",
		Port: 5432,
	}
	err = db.Create(instance2).Error
	require.NoError(t, err)

	// Create audit logs for each instance
	log1 := models.DatabaseAuditLog{
		InstanceID: &instance1.ID,
		Operator:   "admin",
		Action:     "instance.test",
		TargetType: "instance",
		TargetName: "instance1",
		Success:    true,
	}
	err = db.Create(&log1).Error
	require.NoError(t, err)

	log2 := models.DatabaseAuditLog{
		InstanceID: &instance2.ID,
		Operator:   "admin",
		Action:     "instance.test",
		TargetType: "instance",
		TargetName: "instance2",
		Success:    true,
	}
	err = db.Create(&log2).Error
	require.NoError(t, err)

	// Filter by instance1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/audit-logs?instance_id=1", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool), "Expected success=true")
}

// TestAuditContract_FilterByOperator tests filtering by operator name
func TestAuditContract_FilterByOperator(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create audit logs with different operators
	logs := []models.DatabaseAuditLog{
		{
			Operator:   "admin",
			Action:     "database.create",
			TargetType: "database",
			Success:    true,
		},
		{
			Operator:   "user1",
			Action:     "query.execute",
			TargetType: "query",
			Success:    true,
		},
	}
	for i := range logs {
		err := db.Create(&logs[i]).Error
		require.NoError(t, err)
	}

	// Filter by operator
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/audit-logs?operator=admin", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool), "Expected success=true")
}

// TestAuditContract_FilterByAction tests filtering by action type
func TestAuditContract_FilterByAction(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create audit logs with different actions
	logs := []models.DatabaseAuditLog{
		{
			Operator:   "admin",
			Action:     "instance.create",
			TargetType: "instance",
			Success:    true,
		},
		{
			Operator:   "admin",
			Action:     "query.blocked",
			TargetType: "query",
			Success:    false,
		},
	}
	for i := range logs {
		err := db.Create(&logs[i]).Error
		require.NoError(t, err)
	}

	// Filter by action
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/audit-logs?action=query.blocked", nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool), "Expected success=true")
}

// TestAuditContract_FilterByDateRange tests filtering by date range
func TestAuditContract_FilterByDateRange(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create audit logs with different timestamps
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	log1 := models.DatabaseAuditLog{
		BaseModel: models.BaseModel{
			CreatedAt: yesterday,
		},
		Operator:   "admin",
		Action:     "old.action",
		TargetType: "instance",
		Success:    true,
	}
	err := db.Create(&log1).Error
	require.NoError(t, err)

	log2 := models.DatabaseAuditLog{
		Operator:   "admin",
		Action:     "new.action",
		TargetType: "instance",
		Success:    true,
	}
	err = db.Create(&log2).Error
	require.NoError(t, err)

	// Filter by date (today only)
	today := now.Format("2006-01-02")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/database/audit-logs?start_date="+today, nil)
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool), "Expected success=true")
}
