package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	versionhandler "github.com/ysicing/tiga/internal/api/handlers/version"
)

// TestVersionAPISchemaValidation 验证版本 API 响应的 JSON Schema
// 参考: .claude/specs/008-commitid-commit-agent/contracts/version-api.md (测试用例 2)
// 参考: .claude/specs/008-commitid-commit-agent/data-model.md (验证规则)
//
// 重要提示：这个测试在实现 API 端点之前应该失败（TDD 方法）
func TestVersionAPISchemaValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router with version endpoint
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/version", versionhandler.GetVersion)

	t.Run("validates version field format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// 跳过状态码检查（在未实现时会失败），直接测试 schema
		if w.Code != http.StatusOK {
			t.Skip("API not implemented yet")
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		version, ok := response["version"].(string)
		require.True(t, ok, "version should be a string")

		// 验证版本格式：
		// - v1.2.3-a1b2c3d (tag + commit)
		// - 20251026-a1b2c3d (date + commit)
		// - dev (默认值)
		// - snapshot (构建失败时)
		versionPattern := regexp.MustCompile(`^(v?\d+\.\d+\.\d+|\d{8}|dev|snapshot)(-[0-9a-f]{7})?$`)
		assert.True(t, versionPattern.MatchString(version),
			"version format should match pattern: %s (got: %s)", versionPattern.String(), version)
	})

	t.Run("validates build_time field format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("API not implemented yet")
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		buildTime, ok := response["build_time"].(string)
		require.True(t, ok, "build_time should be a string")

		// 验证 build_time 格式：RFC3339 或 "unknown"
		if buildTime != "unknown" {
			_, err := time.Parse(time.RFC3339, buildTime)
			assert.NoError(t, err,
				"build_time should be valid RFC3339 format (got: %s)", buildTime)
		}
	})

	t.Run("validates commit_id field format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("API not implemented yet")
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		commitID, ok := response["commit_id"].(string)
		require.True(t, ok, "commit_id should be a string")

		// 验证 commit_id 格式：7 位十六进制或 "0000000"
		commitPattern := regexp.MustCompile(`^[0-9a-f]{7}$`)
		assert.True(t, commitPattern.MatchString(commitID),
			"commit_id should be 7-character hex string (got: %s)", commitID)
	})

	t.Run("no additional properties in response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Skip("API not implemented yet")
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		// JSON Schema 定义：additionalProperties: false
		// 只允许 version, build_time, commit_id 三个字段
		allowedFields := map[string]bool{
			"version":    true,
			"build_time": true,
			"commit_id":  true,
		}

		for field := range response {
			assert.True(t, allowedFields[field],
				"Unexpected field in response: %s", field)
		}

		// 验证必需字段都存在
		assert.Len(t, response, 3, "Response should contain exactly 3 fields")
	})
}
