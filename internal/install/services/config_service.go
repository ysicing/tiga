package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/ysicing/tiga/internal/install/models"
)

// T019: Viper 配置管理器

// ConfigService 配置服务
type ConfigService struct {
	configPath string
}

// NewConfigService 创建配置服务
func NewConfigService(configPath string) *ConfigService {
	return &ConfigService{
		configPath: configPath,
	}
}

// AppConfig 完整应用配置（用于 YAML 文件）
type AppConfig struct {
	Server struct {
		InstallLock bool   `yaml:"install_lock"`
		AppName     string `yaml:"app_name"`
		AppSubtitle string `yaml:"app_subtitle,omitempty"`
		Domain      string `yaml:"domain"`
		HTTPPort    int    `yaml:"http_port"`
		Language    string `yaml:"language"`
	} `yaml:"server"`
	Database struct {
		Type     string `yaml:"type"`
		Host     string `yaml:"host,omitempty"`
		Port     int    `yaml:"port,omitempty"`
		Database string `yaml:"database"`
		Username string `yaml:"username,omitempty"`
		Password string `yaml:"password,omitempty"`
		SSLMode  string `yaml:"ssl_mode,omitempty"`
		Charset  string `yaml:"charset,omitempty"`
	} `yaml:"database"`
	Security struct {
		JWTSecret     string `yaml:"jwt_secret"`
		EncryptionKey string `yaml:"encryption_key"`
	} `yaml:"security"`
}

// IsInstalled 检查是否已初始化
func (s *ConfigService) IsInstalled() bool {
	// 检查配置文件是否存在
	if _, err := os.Stat(s.configPath); os.IsNotExist(err) {
		return false
	}

	// 读取配置文件
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return false
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return false
	}

	return config.Server.InstallLock
}

// generateRandomSecret 生成随机密钥
func generateRandomSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateConfigFile 生成配置文件
func (s *ConfigService) GenerateConfigFile(installConfig models.InstallConfig) error {
	// 构建配置结构
	config := AppConfig{}
	config.Server.InstallLock = true
	config.Server.AppName = installConfig.Settings.AppName
	config.Server.AppSubtitle = installConfig.Settings.AppSubtitle
	config.Server.Domain = installConfig.Settings.Domain
	config.Server.HTTPPort = installConfig.Settings.HTTPPort
	config.Server.Language = installConfig.Settings.Language

	config.Database.Type = installConfig.Database.Type
	config.Database.Database = installConfig.Database.Database

	// 只在非 SQLite 数据库时设置连接参数
	if installConfig.Database.Type != "sqlite" {
		config.Database.Host = installConfig.Database.Host
		config.Database.Port = installConfig.Database.Port
		config.Database.Username = installConfig.Database.Username
		config.Database.Password = installConfig.Database.Password
		config.Database.SSLMode = installConfig.Database.SSLMode
		config.Database.Charset = installConfig.Database.Charset
	}

	// 生成随机 JWT Secret（32 字节 = 256 位）
	jwtSecret, err := generateRandomSecret(32)
	if err != nil {
		return fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	config.Security.JWTSecret = jwtSecret

	// 生成随机加密密钥（32 字节 = 256 位，AES-256）
	encryptionKey, err := generateRandomSecret(32)
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}
	config.Security.EncryptionKey = encryptionKey

	// 序列化为 YAML
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 原子写入文件
	tmpFile := s.configPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	// 重命名为最终文件（原子操作）
	if err := os.Rename(tmpFile, s.configPath); err != nil {
		os.Remove(tmpFile) // 清理临时文件
		return fmt.Errorf("failed to finalize config file: %w", err)
	}

	return nil
}

// LoadConfig 加载配置文件
func (s *ConfigService) LoadConfig() (*AppConfig, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SetInstallLock 设置 install_lock（用于并发控制）
func (s *ConfigService) SetInstallLock() error {
	// 检查文件是否已存在
	if _, err := os.Stat(s.configPath); err == nil {
		// 文件已存在，检查 install_lock
		config, err := s.LoadConfig()
		if err != nil {
			return err
		}
		if config.Server.InstallLock {
			return fmt.Errorf("installation already completed")
		}
	}

	// 创建最小配置文件（仅设置 install_lock）
	minimalConfig := AppConfig{}
	minimalConfig.Server.InstallLock = true

	data, err := yaml.Marshal(&minimalConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 使用 O_EXCL 确保原子创建
	file, err := os.OpenFile(s.configPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("installation already completed")
		}
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
