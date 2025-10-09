package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ysicing/tiga/internal/install/models"
	"github.com/ysicing/tiga/internal/install/services"
)

// T012: 集成测试 - 配置文件生成

type AppConfig struct {
	Server struct {
		InstallLock bool   `yaml:"install_lock"`
		AppName     string `yaml:"app_name"`
		Domain      string `yaml:"domain"`
		HTTPPort    int    `yaml:"http_port"`
		GRPCPort    int    `yaml:"grpc_port"`
		Language    string `yaml:"language"`
	} `yaml:"server"`
	Database struct {
		Type     string `yaml:"type"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Database string `yaml:"database"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"database"`
	Security struct {
		JWTSecret     string `yaml:"jwt_secret"`
		EncryptionKey string `yaml:"encryption_key"`
	} `yaml:"security"`
}

func TestGenerateConfigFile_Success(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 创建配置服务
	configService := services.NewConfigService(configPath)

	// 创建测试数据
	installConfig := models.InstallConfig{
		Settings: models.SystemSettings{
			AppName:     "Tiga Dashboard",
			AppSubtitle: "DevOps Platform",
			Domain:      "localhost",
			HTTPPort:    12306,
			Language:    "zh-CN",
		},
		Database: models.DatabaseConfig{
			Type:     "postgresql",
			Host:     "127.0.0.1",
			Port:     5432,
			Database: "tiga_test",
			Username: "tiga",
			Password: "tiga_password",
			SSLMode:  "disable",
			Charset:  "utf8mb4",
		},
	}

	// 调用配置生成函数
	err := configService.GenerateConfigFile(installConfig)
	require.NoError(t, err)

	// 验证文件已创建
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// 验证文件内容
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var config AppConfig
	err = yaml.Unmarshal(data, &config)
	require.NoError(t, err)

	assert.True(t, config.Server.InstallLock)
	assert.Equal(t, "Tiga Dashboard", config.Server.AppName)
	assert.Equal(t, "localhost", config.Server.Domain)
	assert.Equal(t, 12306, config.Server.HTTPPort)
	assert.Equal(t, 12307, config.Server.GRPCPort)
	assert.Equal(t, "zh-CN", config.Server.Language)

	assert.Equal(t, "postgresql", config.Database.Type)
	assert.Equal(t, "127.0.0.1", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "tiga_test", config.Database.Database)

	// 验证 JWT Secret 已自动生成
	assert.NotEmpty(t, config.Security.JWTSecret, "JWT Secret should be auto-generated")
	assert.Greater(t, len(config.Security.JWTSecret), 32, "JWT Secret should be at least 32 characters")

	// 验证加密密钥已自动生成
	assert.NotEmpty(t, config.Security.EncryptionKey, "Encryption key should be auto-generated")
	assert.Greater(t, len(config.Security.EncryptionKey), 32, "Encryption key should be at least 32 characters")
}

func TestGenerateConfigFile_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// 创建配置服务
	configService := services.NewConfigService(configPath)

	// 创建测试数据
	installConfig := models.InstallConfig{
		Settings: models.SystemSettings{
			AppName:  "Test App",
			Domain:   "test.local",
			HTTPPort: 12306,
			Language: "en-US",
		},
		Database: models.DatabaseConfig{
			Type:     "sqlite",
			Database: "test.db",
		},
	}

	// 调用配置生成函数
	err := configService.GenerateConfigFile(installConfig)
	require.NoError(t, err)

	// 验证文件权限 (应为 600)
	fileInfo, err := os.Stat(configPath)
	require.NoError(t, err)

	mode := fileInfo.Mode()
	assert.Equal(t, os.FileMode(0600), mode.Perm(), "Config file should have 600 permissions")
}

func TestGenerateConfigFile_AlreadyExists(t *testing.T) {
	t.Skip("待实现：验证文件已存在时的行为")

	// TODO:
	// 1. 创建 config.yaml
	// 2. 再次尝试生成
	// 3. 验证返回错误或覆盖行为
}
