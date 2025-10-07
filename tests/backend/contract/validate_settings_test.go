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

// T007: 测试 POST /api/install/validate-settings 契约

type ValidateSettingsRequest struct {
	AppName  string `json:"app_name"`
	Domain   string `json:"domain"`
	HTTPPort int    `json:"http_port"`
	Language string `json:"language"`
}

type ValidateSettingsResponse struct {
	Valid  bool              `json:"valid"`
	Errors map[string]string `json:"errors,omitempty"`
}

func TestValidateSettings_Success(t *testing.T) {
	reqBody := ValidateSettingsRequest{
		AppName:  "Tiga Dashboard",
		Domain:   "tiga.example.com",
		HTTPPort: 12306,
		Language: "zh-CN",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/validate-settings", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.ValidateSettings(rec, req)

	var resp ValidateSettingsResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, resp.Valid)
	assert.Nil(t, resp.Errors)
}

func TestValidateSettings_InvalidPort(t *testing.T) {
	reqBody := ValidateSettingsRequest{
		AppName:  "Tiga Dashboard",
		Domain:   "tiga.example.com",
		HTTPPort: 70000, // 超出范围
		Language: "zh-CN",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/validate-settings", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.ValidateSettings(rec, req)

	var resp ValidateSettingsResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, resp.Valid)
	assert.Contains(t, resp.Errors["http_port"], "between 1 and 65535")
}

func TestValidateSettings_EmptyAppName(t *testing.T) {
	reqBody := ValidateSettingsRequest{
		AppName:  "",
		Domain:   "tiga.example.com",
		HTTPPort: 12306,
		Language: "zh-CN",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/validate-settings", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.ValidateSettings(rec, req)

	var resp ValidateSettingsResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, resp.Valid)
	assert.NotNil(t, resp.Errors["app_name"])
}
