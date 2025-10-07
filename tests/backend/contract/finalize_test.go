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

// T008: 测试 POST /api/install/finalize 契约

type FinalizeRequest struct {
	Database         CheckDBRequest          `json:"database"`
	Admin            ValidateAdminRequest    `json:"admin"`
	Settings         ValidateSettingsRequest `json:"settings"`
	ConfirmReinstall bool                    `json:"confirm_reinstall"`
}

type FinalizeResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
	Error        string `json:"error,omitempty"`
}

func TestFinalize_Success(t *testing.T) {
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

	reqBody := FinalizeRequest{
		Database: CheckDBRequest{
			Type:     "postgresql",
			Host:     host,
			Port:     port.Int(),
			Database: "tiga_test",
			Username: "tiga",
			Password: "test_password",
			SSLMode:  "disable",
		},
		Admin: ValidateAdminRequest{
			Username: "admin",
			Password: "Admin123!",
			Email:    "admin@example.com",
		},
		Settings: ValidateSettingsRequest{
			AppName:  "Tiga Dashboard",
			Domain:   "localhost",
			HTTPPort: 12306,
			Language: "zh-CN",
		},
		ConfirmReinstall: false,
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/install/finalize", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// TODO: 替换为实际的 handler
	// handler.Finalize(rec, req)

	var resp FinalizeResponse
	err = json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, resp.Success)
	assert.Equal(t, "Installation completed successfully", resp.Message)
	assert.NotEmpty(t, resp.SessionToken)
}

func TestFinalize_AlreadyInstalled(t *testing.T) {
	t.Skip("待实现：需要设置 install_lock = true")

	// TODO:
	// 1. 设置 install_lock = true
	// 2. 调用 /api/install/finalize
	// 3. 验证返回 403 Forbidden
}

func TestFinalize_ExistingDataWithoutConfirmation(t *testing.T) {
	t.Skip("待实现：需要预先插入数据到数据库")

	// TODO:
	// 1. 启动 PostgreSQL 容器
	// 2. 插入现有数据
	// 3. confirm_reinstall = false
	// 4. 验证返回 409 Conflict
}
