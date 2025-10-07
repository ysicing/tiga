package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// T011: 集成测试 - 创建管理员账户

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex;not null"`
	Email    string `gorm:"uniqueIndex;not null"`
	Password string `gorm:"not null"`
	IsAdmin  bool   `gorm:"default:false"`
}

func TestCreateAdminAccount_Success(t *testing.T) {
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

	// 创建 users 表
	err = db.AutoMigrate(&User{})
	require.NoError(t, err)

	// TODO: 调用服务层创建管理员
	// err = service.CreateAdminAccount(db, "admin", "Admin123!", "admin@example.com")
	// require.NoError(t, err)

	// 验证管理员已创建
	var admin User
	err = db.Where("username = ?", "admin").First(&admin).Error
	require.NoError(t, err)

	assert.Equal(t, "admin", admin.Username)
	assert.Equal(t, "admin@example.com", admin.Email)
	assert.True(t, admin.IsAdmin)

	// 验证密码已哈希
	assert.NotEqual(t, "Admin123!", admin.Password)
	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("Admin123!"))
	assert.NoError(t, err, "Password should be bcrypt hashed")
}

func TestCreateAdminAccount_DuplicateUsername(t *testing.T) {
	t.Skip("待实现：验证用户名重复时返回错误")

	// TODO:
	// 1. 创建管理员 'admin'
	// 2. 再次创建 'admin'
	// 3. 验证返回 "username already exists" 错误
}

func TestCreateAdminAccount_WeakPassword(t *testing.T) {
	t.Skip("待实现：验证弱密码被拒绝")

	// TODO:
	// 1. 尝试使用弱密码创建管理员
	// 2. 验证返回验证错误
}
