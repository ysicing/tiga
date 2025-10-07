package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T009: 测试 GET /api/install/status 契约

type StatusResponse struct {
	Installed  bool   `json:"installed"`
	RedirectTo string `json:"redirect_to"`
}

func TestStatus_NotInstalled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/install/status", nil)
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.Status(rec, req)

	var resp StatusResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, resp.Installed)
	assert.Equal(t, "/install", resp.RedirectTo)
}

func TestStatus_Installed(t *testing.T) {
	t.Skip("待实现：需要设置 install_lock = true")

	// TODO:
	// 1. 设置 install_lock = true
	// 2. 调用 /api/install/status
	// 3. 验证 installed = true, redirect_to = "/login"
}
