package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/models"
)

// TestHelper provides common test utilities for contract tests
type TestHelper struct {
	T      *testing.T
	DB     *gorm.DB
	Router *gin.Engine
	Server *httptest.Server
}

// NewTestHelper creates a new test helper instance
func NewTestHelper(t *testing.T) *TestHelper {
	// TODO: Initialize test database and router
	// This will be implemented in T024 when the core implementation is ready
	return &TestHelper{
		T: t,
	}
}

// SetupTestDB initializes a test database
func (h *TestHelper) SetupTestDB() error {
	// TODO: Implement in T024
	return fmt.Errorf("not implemented: database setup required")
}

// SetupRouter initializes the API router
func (h *TestHelper) SetupRouter(cfg *config.Config) error {
	// TODO: Implement when API handlers are ready (T036-T041)
	return fmt.Errorf("not implemented: router setup required")
}

// Cleanup cleans up test resources
func (h *TestHelper) Cleanup() {
	if h.Server != nil {
		h.Server.Close()
	}
	// TODO: Close DB connection
}

// CreateTestRecording creates a test recording in the database
func (h *TestHelper) CreateTestRecording(rec *models.TerminalRecording) error {
	// TODO: Implement in T024
	return fmt.Errorf("not implemented: CreateTestRecording requires TerminalRecording model extension")
}

// MakeRequest performs an HTTP request and returns the response
func (h *TestHelper) MakeRequest(method, path string, body interface{}) (*httptest.ResponseRecorder, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	// TODO: Add authentication token when auth is implemented
	// req.Header.Set("Authorization", "Bearer "+h.GetTestToken())

	w := httptest.NewRecorder()

	if h.Router == nil {
		return nil, fmt.Errorf("router not initialized")
	}

	h.Router.ServeHTTP(w, req)
	return w, nil
}

// AssertJSONResponse validates a JSON response
func (h *TestHelper) AssertJSONResponse(t *testing.T, resp *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	assert.Equal(t, expectedStatus, resp.Code, "unexpected status code")
	assert.Equal(t, "application/json; charset=utf-8", resp.Header().Get("Content-Type"))

	var data map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &data)
	require.NoError(t, err, "response should be valid JSON")

	return data
}

// AssertSuccessResponse validates a successful API response
func (h *TestHelper) AssertSuccessResponse(t *testing.T, resp *httptest.ResponseRecorder) map[string]interface{} {
	data := h.AssertJSONResponse(t, resp, http.StatusOK)

	success, ok := data["success"].(bool)
	require.True(t, ok, "response should have 'success' field")
	assert.True(t, success, "'success' should be true")

	return data
}

// AssertErrorResponse validates an error API response
func (h *TestHelper) AssertErrorResponse(t *testing.T, resp *httptest.ResponseRecorder, expectedStatus int, expectedCode string) {
	data := h.AssertJSONResponse(t, resp, expectedStatus)

	success, ok := data["success"].(bool)
	require.True(t, ok, "response should have 'success' field")
	assert.False(t, success, "'success' should be false")

	errorData, ok := data["error"].(map[string]interface{})
	require.True(t, ok, "response should have 'error' object")

	code, ok := errorData["code"].(string)
	require.True(t, ok, "error should have 'code' field")
	assert.Equal(t, expectedCode, code, "unexpected error code")
}

// AssertPaginationStructure validates pagination structure
func (h *TestHelper) AssertPaginationStructure(t *testing.T, pagination interface{}) {
	pag, ok := pagination.(map[string]interface{})
	require.True(t, ok, "pagination should be an object")

	assert.Contains(t, pag, "page", "pagination should have 'page'")
	assert.Contains(t, pag, "limit", "pagination should have 'limit'")
	assert.Contains(t, pag, "total_pages", "pagination should have 'total_pages'")
	assert.Contains(t, pag, "total_count", "pagination should have 'total_count'")
}
