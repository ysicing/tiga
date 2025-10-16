package contract

import (
	"testing"
)

// T009: 测试 GET /api/install/status 契约

type StatusResponse struct {
	Installed  bool   `json:"installed"`
	RedirectTo string `json:"redirect_to"`
}

func TestStatus_NotInstalled(t *testing.T) {
	t.Skip("待实现：需要实际的 handler")

	// TODO: 替换为实际的 handler
	// req := httptest.NewRequest(http.MethodGet, "/api/install/status", nil)
	// rec := httptest.NewRecorder()
	// handler.Status(rec, req)
	// var resp StatusResponse
	// err := json.NewDecoder(rec.Body).Decode(&resp)
	// require.NoError(t, err)
	// assert.Equal(t, http.StatusOK, rec.Code)
	// assert.False(t, resp.Installed)
	// assert.Equal(t, "/install", resp.RedirectTo)
}

func TestStatus_Installed(t *testing.T) {
	t.Skip("待实现：需要设置 install_lock = true")

	// TODO:
	// 1. 设置 install_lock = true
	// 2. 调用 /api/install/status
	// 3. 验证 installed = true, redirect_to = "/login"
}
