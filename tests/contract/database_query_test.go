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

// TestQueryContract_ExecuteSELECT tests POST /api/v1/database/instances/:id/query with SELECT
func TestQueryContract_ExecuteSELECT(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-query-select",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Request payload with SELECT query
	payload := map[string]interface{}{
		"database_name": "mysql",
		"query":         "SELECT 1 AS result",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/database/instances/1/query", bytes.NewReader(body))
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

// TestQueryContract_BlockDDL tests that DDL operations are forbidden
func TestQueryContract_BlockDDL(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-query-ddl",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Test cases for DDL operations that should be blocked
	ddlQueries := []string{
		"DROP TABLE users",
		"TRUNCATE TABLE logs",
		"ALTER TABLE users ADD COLUMN age INT",
		"CREATE TABLE test (id INT)",
	}

	for _, query := range ddlQueries {
		t.Run(query, func(t *testing.T) {
			payload := map[string]interface{}{
				"database_name": "test",
				"query":         query,
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/database/instances/1/query", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer mock-jwt-token")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should return 400 Bad Request
			assert.Equal(t, http.StatusBadRequest, w.Code, "DDL should be blocked with 400 status")

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.False(t, response["success"].(bool), "Expected success=false for DDL")
			if msg, ok := response["error"].(string); ok {
				assert.Contains(t, msg, "forbidden", "Error message should mention forbidden")
			}
		})
	}
}

// TestQueryContract_BlockUnsafeUpdate tests that UPDATE without WHERE is forbidden
func TestQueryContract_BlockUnsafeUpdate(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-query-unsafe",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Test UPDATE without WHERE
	payload := map[string]interface{}{
		"database_name": "test",
		"query":         "UPDATE users SET active = 0",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/database/instances/1/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code, "UPDATE without WHERE should be blocked")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool), "Expected success=false")
}

// TestQueryContract_ResultStructure tests the structure of QueryResult
func TestQueryContract_ResultStructure(t *testing.T) {
	router, db, cleanup := setupTestAPI(t)
	defer cleanup()

	// Create test instance
	instance := &models.DatabaseInstance{
		Name:     "test-query-result",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Status:   "online",
	}
	err := db.Create(instance).Error
	require.NoError(t, err)

	// Execute a simple query
	payload := map[string]interface{}{
		"database_name": "mysql",
		"query":         "SELECT 'test' AS col1, 123 AS col2",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/database/instances/1/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mock-jwt-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// If query succeeded, verify result structure
	if response["success"].(bool) {
		data := response["data"].(map[string]interface{})

		// Should have columns, rows, row_count, duration fields
		_, hasColumns := data["columns"]
		_, hasRows := data["rows"]
		_, hasRowCount := data["row_count"]
		_, hasDuration := data["duration"]

		assert.True(t, hasColumns, "Result should have 'columns' field")
		assert.True(t, hasRows, "Result should have 'rows' field")
		assert.True(t, hasRowCount, "Result should have 'row_count' field")
		assert.True(t, hasDuration, "Result should have 'duration' field")
	}
}
