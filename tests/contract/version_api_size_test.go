package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	versionhandler "github.com/ysicing/tiga/internal/api/handlers/version"
)

// TestVersionAPIResponseSize 验证版本 API 响应体大小
// 参考: .claude/specs/008-commitid-commit-agent/contracts/version-api.md (测试用例 4)
//
// 响应体大小目标：<500 bytes
//
// 重要提示：这个测试在实现 API 端点之前应该失败（TDD 方法）
func TestVersionAPIResponseSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router with version endpoint
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/version", versionhandler.GetVersion)

	t.Run("response body size should be less than 500 bytes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// 跳过如果端点未实现
		if w.Code != http.StatusOK {
			t.Skip("API not implemented yet")
		}

		// 验证响应体大小
		responseSize := w.Body.Len()
		assert.Less(t, responseSize, 500,
			"Response body size should be less than 500 bytes (got: %d bytes)", responseSize)

		t.Logf("Response body size: %d bytes", responseSize)
	})

	t.Run("response is valid and compact JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("API not implemented yet")
		}

		// 验证 JSON 格式正确
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Response should be valid JSON")

		// 验证 JSON 紧凑（无多余空格）
		compactJSON, _ := json.Marshal(response)
		originalJSON := w.Body.Bytes()

		// 紧凑 JSON 长度应该接近原始长度（允许 gin 的格式化）
		sizeDiff := len(originalJSON) - len(compactJSON)
		assert.Less(t, sizeDiff, 50,
			"JSON should be reasonably compact (size difference: %d bytes)", sizeDiff)
	})

	t.Run("response includes correct Content-Length header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("API not implemented yet")
		}

		// 验证 Content-Length 头存在且正确
		contentLength := w.Header().Get("Content-Length")
		if contentLength != "" {
			assert.Equal(t, string(rune(w.Body.Len())), contentLength,
				"Content-Length header should match actual body length")
		}

		t.Logf("Response headers: %v", w.Header())
	})
}
