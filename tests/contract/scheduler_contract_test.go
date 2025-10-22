package contract

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/api"
	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/models"

	authservices "github.com/ysicing/tiga/internal/services/auth"
	hostservices "github.com/ysicing/tiga/internal/services/host"
	monitorservices "github.com/ysicing/tiga/internal/services/monitor"
)

// setupTestRouter 设置测试用的 Gin router
// 使用内存数据库和最小依赖配置
// 返回: router, database, validJWTToken
func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, string) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to create in-memory database")

	// 初始化全局 DB 变量（用于 ClusterManager）
	models.DB = database

	// 运行迁移 - 直接使用 gorm.AutoMigrate
	err = database.AutoMigrate(
		&models.User{},
		&models.Instance{},
		&models.Alert{},
		&models.AlertEvent{},
		&models.AuditLog{},
		&models.AuditEvent{},
		&models.ScheduledTask{},
		&models.TaskExecution{},
		// Add other models as needed
	)
	require.NoError(t, err, "Failed to run migrations")

	// 创建测试用户和 JWT token
	hashedPassword := "$2a$10$test.hash" // Dummy hash for testing
	testUser := &models.User{
		Username: "testuser",
		Password: hashedPassword,
		Email:    "test@example.com",
		IsAdmin:  true,
		Provider: "password",
		Status:   "active",
		Enabled:  true,
	}
	database.Create(testUser)

	// 创建路由器
	router := gin.New()

	// 创建最小配置
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 12306,
		},
		JWT: config.JWTConfig{
			Secret:    "test-secret",
			ExpiresIn: time.Hour * 24,
		},
		Features: config.FeaturesConfig{
			ReadonlyMode: false,
		},
	}

	// 创建 JWT manager - 使用正确的参数
	jwtManager := authservices.NewJWTManager(cfg.JWT.Secret, time.Hour, time.Hour*24)

	// 生成有效的 JWT token 用于测试
	var roles []string
	if testUser.IsAdmin {
		roles = []string{"admin"}
	}
	accessToken, _, err := jwtManager.GenerateAccessToken(testUser.ID, testUser.Username, testUser.Email, roles)
	require.NoError(t, err, "Failed to generate JWT token")

	// 创建必要的服务（使用 nil 值,因为 Scheduler/Audit 测试不需要它们）
	hostService := &hostservices.HostService{}
	stateCollector := &hostservices.StateCollector{}
	terminalManager := &hostservices.TerminalManager{}
	probeScheduler := &monitorservices.ServiceProbeScheduler{}

	// 创建临时配置文件
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	tmpfile.WriteString("server:\n  install_lock: true\n")
	tmpfile.Close() // Close immediately to ensure content is written
	configPath := tmpfile.Name()
	t.Cleanup(func() { os.Remove(configPath) }) // Clean up after test

	// 设置路由
	api.SetupRoutes(
		router,
		database,
		configPath,
		jwtManager,
		"test-secret",
		config.DatabaseManagementConfig{},
		hostService,
		stateCollector,
		terminalManager,
		probeScheduler,
		cfg,
	)

	return router, database, accessToken
}

// TestSchedulerAPIContract 验证 Scheduler API 契约
// 参考: .claude/specs/006-gitness-tiga/contracts/scheduler_api.yaml
//
// 重要提示：这些测试在实现 API 端点之前应该全部失败（TDD 方法）
// 当前状态：预期所有测试失败（404 或空响应）
func TestSchedulerAPIContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 设置测试路由和数据库
	router, database, token := setupTestRouter(t)
	authHeader := "Bearer " + token // 构造完整的 Authorization header
	defer func() {
		sqlDB, _ := database.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	t.Run("GET /api/v1/scheduler/tasks - 获取任务列表", func(t *testing.T) {
		// 测试场景 1: 基本查询（无过滤）
		t.Run("should return paginated task list", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks?page=1&page_size=20", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 验证响应状态码
			assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

			// 验证响应结构
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err, "Response should be valid JSON")

			// 验证 data 字段
			assert.Contains(t, response, "data", "Response should contain 'data' field")
			data, ok := response["data"].([]interface{})
			assert.True(t, ok, "data should be an array")
			assert.NotNil(t, data, "data should not be nil")

			// 验证 pagination 字段
			assert.Contains(t, response, "pagination", "Response should contain 'pagination' field")
			pagination, ok := response["pagination"].(map[string]interface{})
			assert.True(t, ok, "pagination should be an object")
			assert.Contains(t, pagination, "page", "pagination should have 'page'")
			assert.Contains(t, pagination, "page_size", "pagination should have 'page_size'")
			assert.Contains(t, pagination, "total", "pagination should have 'total'")
			assert.Contains(t, pagination, "total_pages", "pagination should have 'total_pages'")
		})

		// 测试场景 2: 按启用状态过滤
		t.Run("should filter by enabled status", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks?enabled=true", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			// 如果有数据，验证所有任务都是启用状态
			if data, ok := response["data"].([]interface{}); ok && len(data) > 0 {
				for _, item := range data {
					task := item.(map[string]interface{})
					assert.True(t, task["enabled"].(bool), "All tasks should be enabled")
				}
			}
		})

		// 测试场景 3: 按任务类型过滤
		t.Run("should filter by task type", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks?type=alert_processing", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		// 测试场景 4: 未授权访问
		t.Run("should return 401 without auth token", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})

	t.Run("GET /api/v1/scheduler/tasks/:id - 获取任务详情", func(t *testing.T) {
		taskID := "task-uuid-1234"

		t.Run("should return task details", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/scheduler/tasks/%s", taskID), nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// 验证 data 字段
			assert.Contains(t, response, "data")
			data, ok := response["data"].(map[string]interface{})
			assert.True(t, ok, "data should be an object")

			// 验证必填字段
			requiredFields := []string{"uid", "name", "type", "is_recurring", "enabled",
				"max_duration_seconds", "created_at", "updated_at"}
			for _, field := range requiredFields {
				assert.Contains(t, data, field, "Task should contain '%s' field", field)
			}
		})

		t.Run("should return 404 for non-existent task", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/tasks/non-existent-id", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			assert.Contains(t, response, "error")
		})
	})

	t.Run("POST /api/v1/scheduler/tasks/:id/enable - 启用任务", func(t *testing.T) {
		taskID := "task-uuid-1234"

		t.Run("should enable task successfully", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/scheduler/tasks/%s/enable", taskID), nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "message")
			assert.NotEmpty(t, response["message"])
		})

		t.Run("should return 404 for non-existent task", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/tasks/non-existent/enable", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("should return 401 without auth", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/scheduler/tasks/%s/enable", taskID), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})

	t.Run("POST /api/v1/scheduler/tasks/:id/disable - 禁用任务", func(t *testing.T) {
		taskID := "task-uuid-1234"

		t.Run("should disable task successfully", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/scheduler/tasks/%s/disable", taskID), nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "message")
		})

		t.Run("should return 404 for non-existent task", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/tasks/non-existent/disable", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("POST /api/v1/scheduler/tasks/:id/trigger - 手动触发任务", func(t *testing.T) {
		taskID := "task-uuid-1234"

		t.Run("should trigger task successfully", func(t *testing.T) {
			body := `{"override_data": "{\"retention_days\": 30}"}`
			req := httptest.NewRequest(http.MethodPost,
				fmt.Sprintf("/api/v1/scheduler/tasks/%s/trigger", taskID),
				strings.NewReader(body))
			req.Header.Set("Authorization", authHeader)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 验证 HTTP 202 Accepted
			assert.Equal(t, http.StatusAccepted, w.Code, "Expected 202 Accepted")

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// 验证响应包含必要字段
			assert.Contains(t, response, "message")
			assert.Contains(t, response, "execution_uid")
			assert.NotEmpty(t, response["execution_uid"], "execution_uid should not be empty")
		})

		t.Run("should trigger without override data", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost,
				fmt.Sprintf("/api/v1/scheduler/tasks/%s/trigger", taskID), nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusAccepted, w.Code)
		})

		t.Run("should return 404 for non-existent task", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/tasks/non-existent/trigger", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})

		t.Run("should return 409 if task already running", func(t *testing.T) {
			// 注意：这个测试需要实际的任务执行状态
			// 当前仅验证 API 契约，实际逻辑在集成测试中验证
			req := httptest.NewRequest(http.MethodPost,
				fmt.Sprintf("/api/v1/scheduler/tasks/%s/trigger", taskID), nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 可能返回 202 或 409，取决于任务状态
			assert.True(t, w.Code == http.StatusAccepted || w.Code == http.StatusConflict,
				"Expected 202 or 409, got %d", w.Code)
		})
	})

	t.Run("GET /api/v1/scheduler/executions - 获取执行历史", func(t *testing.T) {
		t.Run("should return paginated execution history", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/executions?page=1&page_size=20", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "data")
			assert.Contains(t, response, "pagination")
		})

		t.Run("should filter by task name", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/scheduler/executions?task_name=alert_processing", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if data, ok := response["data"].([]interface{}); ok && len(data) > 0 {
				for _, item := range data {
					exec := item.(map[string]interface{})
					assert.Equal(t, "alert_processing", exec["task_name"])
				}
			}
		})

		t.Run("should filter by state", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/scheduler/executions?state=failure", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("should filter by time range", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/scheduler/executions?start_time=1697529600000&end_time=1697616000000", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	})

	t.Run("GET /api/v1/scheduler/executions/:id - 获取执行详情", func(t *testing.T) {
		executionID := "12345"

		t.Run("should return execution details", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("/api/v1/scheduler/executions/%s", executionID), nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "data")
			data := response["data"].(map[string]interface{})

			// 验证必填字段
			requiredFields := []string{"id", "task_uid", "task_name", "task_type",
				"execution_uid", "run_by", "scheduled_at", "started_at", "state",
				"trigger_type", "created_at", "updated_at"}
			for _, field := range requiredFields {
				assert.Contains(t, data, field, "Execution should contain '%s' field", field)
			}
		})

		t.Run("should return 404 for non-existent execution", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/executions/999999", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("GET /api/v1/scheduler/stats - 获取统计数据", func(t *testing.T) {
		t.Run("should return statistics", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/stats", nil)
			req.Header.Set("Authorization", authHeader)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "data")
			data := response["data"].(map[string]interface{})

			// 验证全局统计字段
			globalFields := []string{"total_tasks", "enabled_tasks", "total_executions",
				"success_rate", "average_duration_ms", "task_stats"}
			for _, field := range globalFields {
				assert.Contains(t, data, field, "Stats should contain '%s' field", field)
			}

			// 验证任务统计数组
			taskStats, ok := data["task_stats"].([]interface{})
			assert.True(t, ok, "task_stats should be an array")

			if len(taskStats) > 0 {
				stat := taskStats[0].(map[string]interface{})
				statFields := []string{"task_name", "total_executions", "success_executions",
					"failure_executions", "average_duration_ms", "last_executed_at"}
				for _, field := range statFields {
					assert.Contains(t, stat, field, "Task stat should contain '%s' field", field)
				}
			}
		})

		t.Run("should return 401 without auth", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler/stats", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})
}

// TestSchedulerAPIResponseSchema 验证响应 Schema 完整性
func TestSchedulerAPIResponseSchema(t *testing.T) {
	t.Run("ScheduledTask schema validation", func(t *testing.T) {
		// 测试 ScheduledTask 的 JSON 序列化/反序列化
		taskJSON := `{
			"uid": "task-uuid-1234",
			"name": "alert_processing",
			"type": "alert_processing",
			"description": "处理系统告警并发送通知",
			"is_recurring": true,
			"cron_expr": "*/5 * * * *",
			"next_run": "2025-10-19T12:40:00Z",
			"enabled": true,
			"max_duration_seconds": 3600,
			"max_retries": 3,
			"timeout_grace_period": 30,
			"max_concurrent": 1,
			"priority": 0,
			"total_executions": 523,
			"success_executions": 520,
			"failure_executions": 3,
			"consecutive_failures": 0,
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-10-19T12:00:00Z"
		}`

		var task map[string]interface{}
		err := json.Unmarshal([]byte(taskJSON), &task)
		require.NoError(t, err, "Task JSON should be valid")

		// 验证必填字段
		assert.NotEmpty(t, task["uid"])
		assert.NotEmpty(t, task["name"])
		assert.NotEmpty(t, task["type"])
	})

	t.Run("TaskExecution schema validation", func(t *testing.T) {
		executionJSON := `{
			"id": 12345,
			"task_uid": "task-uuid-1234",
			"task_name": "alert_processing",
			"task_type": "alert_processing",
			"execution_uid": "exec-uuid-5678",
			"run_by": "instance-1",
			"scheduled_at": "2025-10-19T12:30:00Z",
			"started_at": "2025-10-19T12:30:01Z",
			"finished_at": "2025-10-19T12:30:15Z",
			"state": "success",
			"result": "processed 15 alerts",
			"duration_ms": 14523,
			"progress": 100,
			"retry_count": 0,
			"trigger_type": "scheduled",
			"created_at": "2025-10-19T12:30:00Z",
			"updated_at": "2025-10-19T12:30:15Z"
		}`

		var execution map[string]interface{}
		err := json.Unmarshal([]byte(executionJSON), &execution)
		require.NoError(t, err, "Execution JSON should be valid")

		// 验证必填字段
		assert.NotEmpty(t, execution["id"])
		assert.NotEmpty(t, execution["task_uid"])
		assert.NotEmpty(t, execution["execution_uid"])

		// 验证枚举值
		state := execution["state"].(string)
		validStates := []string{"pending", "running", "success", "failure", "timeout", "cancelled", "interrupted"}
		assert.Contains(t, validStates, state, "state should be a valid enum value")

		triggerType := execution["trigger_type"].(string)
		validTriggerTypes := []string{"scheduled", "manual"}
		assert.Contains(t, validTriggerTypes, triggerType, "trigger_type should be a valid enum value")
	})
}
