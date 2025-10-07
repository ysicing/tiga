package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T006: 测试 POST /api/install/validate-admin 契约

type ValidateAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type ValidateAdminResponse struct {
	Valid  bool              `json:"valid"`
	Errors map[string]string `json:"errors,omitempty"`
}

func TestValidateAdmin_Success(t *testing.T) {
	reqBody := ValidateAdminRequest{
		Username: "admin",
		Password: "Admin123!",
		Email:    "admin@example.com",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/validate-admin", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.ValidateAdmin(rec, req)

	var resp ValidateAdminResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, resp.Valid)
	assert.Nil(t, resp.Errors)
}

func TestValidateAdmin_WeakPassword(t *testing.T) {
	reqBody := ValidateAdminRequest{
		Username: "admin",
		Password: "admin123", // 缺少大写字母
		Email:    "admin@example.com",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/validate-admin", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.ValidateAdmin(rec, req)

	var resp ValidateAdminResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.Errors["password"], "uppercase letter")
}

func TestValidateAdmin_InvalidEmail(t *testing.T) {
	reqBody := ValidateAdminRequest{
		Username: "admin",
		Password: "Admin123!",
		Email:    "admin@", // 无效邮箱
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/validate-admin", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.ValidateAdmin(rec, req)

	var resp ValidateAdminResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.Errors["email"], "Invalid email format")
}

func TestValidateAdmin_ShortUsername(t *testing.T) {
	reqBody := ValidateAdminRequest{
		Username: "ad", // 少于 3 个字符
		Password: "Admin123!",
		Email:    "admin@example.com",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/validate-admin", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.ValidateAdmin(rec, req)

	var resp ValidateAdminResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.Errors["username"], "at least 3 characters")
}
