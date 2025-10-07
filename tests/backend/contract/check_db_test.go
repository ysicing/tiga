package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// T005: 测试 POST /api/install/check-db 契约

type CheckDBRequest struct {
	Type     string `json:"type"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	SSLMode  string `json:"ssl_mode,omitempty"`
	Charset  string `json:"charset,omitempty"`
}

type CheckDBResponse struct {
	Success         bool   `json:"success"`
	HasExistingData bool   `json:"has_existing_data,omitempty"`
	SchemaVersion   string `json:"schema_version,omitempty"`
	CanUpgrade      bool   `json:"can_upgrade,omitempty"`
	Error           string `json:"error,omitempty"`
}

func TestCheckDB_EmptyDatabase_Success(t *testing.T) {
	ctx := context.Background()

	// 启动 PostgreSQL 容器
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:17-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "tiga",
				"POSTGRES_PASSWORD": "test_password",
				"POSTGRES_DB":       "tiga_test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		},
		Started: true,
	})
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)
	port, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// 准备请求
	reqBody := CheckDBRequest{
		Type:     "postgresql",
		Host:     host,
		Port:     port.Int(),
		Database: "tiga_test",
		Username: "tiga",
		Password: "test_password",
		SSLMode:  "disable",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/check-db", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.CheckDB(rec, req)

	// 预期结果验证
	var resp CheckDBResponse
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, resp.Success)
	assert.False(t, resp.HasExistingData)
}

func TestCheckDB_ExistingData_Detected(t *testing.T) {
	t.Skip("待实现：需要预先插入数据到数据库")

	// TODO:
	// 1. 启动 PostgreSQL 容器
	// 2. 运行迁移创建表
	// 3. 插入测试数据
	// 4. 调用 /api/install/check-db
	// 5. 验证 has_existing_data = true, schema_version 存在
}

func TestCheckDB_ConnectionFailed(t *testing.T) {
	reqBody := CheckDBRequest{
		Type:     "postgresql",
		Host:     "127.0.0.1",
		Port:     9999, // 错误端口
		Database: "tiga_test",
		Username: "tiga",
		Password: "wrong_password",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/check-db", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.CheckDB(rec, req)

	var resp CheckDBResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "connection refused")
}

func TestCheckDB_ValidationFailed_MissingHost(t *testing.T) {
	reqBody := CheckDBRequest{
		Type: "postgresql",
		// Host 缺失
		Database: "tiga_test",
		Username: "tiga",
		Password: "password",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/check-db", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.CheckDB(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var respBody map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	require.NoError(t, err)

	assert.Contains(t, respBody, "error")
	assert.Equal(t, "Invalid database configuration", respBody["error"])
}
