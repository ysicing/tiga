package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	versionhandler "github.com/ysicing/tiga/internal/api/handlers/version"
)

// TestVersionAPISuccess 验证版本 API 成功响应契约
// 参考: .claude/specs/008-commitid-commit-agent/contracts/version-api.md (测试用例 1)
//
// 重要提示：这个测试在实现 API 端点之前应该失败（TDD 方法）
// 当前状态：预期测试失败（404 或空响应）
func TestVersionAPISuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router with version endpoint
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/version", versionhandler.GetVersion)

	t.Run("GET /api/v1/version returns 200 OK with version information", func(t *testing.T) {
		// 创建 GET 请求
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, req)

		// 验证响应状态码
		assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

		// 验证响应头
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"),
			"Content-Type should be application/json")

		// 验证响应体格式
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// 验证必需字段存在
		assert.Contains(t, response, "version", "Response should contain 'version' field")
		assert.Contains(t, response, "build_time", "Response should contain 'build_time' field")
		assert.Contains(t, response, "commit_id", "Response should contain 'commit_id' field")

		// 验证字段非空
		version, ok := response["version"].(string)
		assert.True(t, ok, "version should be a string")
		assert.NotEmpty(t, version, "version should not be empty")

		buildTime, ok := response["build_time"].(string)
		assert.True(t, ok, "build_time should be a string")
		assert.NotEmpty(t, buildTime, "build_time should not be empty")

		commitID, ok := response["commit_id"].(string)
		assert.True(t, ok, "commit_id should be a string")
		assert.NotEmpty(t, commitID, "commit_id should not be empty")
	})

	t.Run("GET /api/v1/version does not require authentication", func(t *testing.T) {
		// 测试无认证头的请求
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// 应该返回 200 OK（无需认证）
		assert.Equal(t, http.StatusOK, w.Code,
			"Version endpoint should be accessible without authentication")
	})
}
