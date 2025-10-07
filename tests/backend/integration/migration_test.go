package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// T010: 集成测试 - 数据库迁移

func TestDatabaseMigration_Success(t *testing.T) {
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

	// 连接数据库
	dsn := "host=" + host + " user=tiga password=test_password dbname=tiga_test port=" + port.Port() + " sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// TODO: 运行迁移
	// err = migration.RunMigrations(db)
	// require.NoError(t, err)

	// 验证表已创建
	var tableNames []string
	err = db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tableNames).Error
	require.NoError(t, err)

	// 预期的表
	expectedTables := []string{"users", "system_config"}
	for _, table := range expectedTables {
		assert.Contains(t, tableNames, table, "Table %s should exist", table)
	}
}

func TestDatabaseMigration_Idempotent(t *testing.T) {
	t.Skip("待实现：验证迁移可以重复运行")

	// TODO:
	// 1. 运行迁移两次
	// 2. 验证第二次运行不会出错
	// 3. 验证数据没有重复
}
