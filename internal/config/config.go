package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	Server             ServerConfig
	Database           DatabaseConfig
	Redis              RedisConfig
	JWT                JWTConfig
	OAuth              OAuthConfig
	Security           SecurityConfig
	DatabaseManagement DatabaseManagementConfig
	Log                LogConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Debug        bool
	Port         int
	GRPCDomain   string
	GRPCPort     int
	ReadTimeout  int
	WriteTimeout int
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Type            string // Database type: postgres, mysql, sqlite
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// DatabaseManagementConfig holds configuration for the database management subsystem.
type DatabaseManagementConfig struct {
	CredentialKey       string
	QueryTimeoutSeconds int
	MaxResultBytes      int
	AuditRetentionDays  int
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
}

// OAuthConfig holds OAuth provider configuration
type OAuthConfig struct {
	Google OAuthProvider
	GitHub OAuthProvider
}

// OAuthProvider holds OAuth provider credentials
type OAuthProvider struct {
	ClientID     string
	ClientSecret string
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	EncryptionKey string
	BcryptCost    int
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string
	Format string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return LoadFromEnv(), nil
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(filename string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// File doesn't exist, fall back to environment variables
		return LoadFromEnv(), nil
	}

	// Read YAML file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	// Parse YAML into ConfigFile structure
	var configFile ConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
	}

	// Convert ConfigFile to Config
	config := &Config{
		Server: ServerConfig{
			Debug:        getEnvAsBool("DEBUG", false),
			Port:         configFile.Server.HTTPPort,
			GRPCDomain:   getOrDefault(configFile.Server.GRPCDomain, configFile.Server.Domain),
			GRPCPort:     getIntOrDefault(configFile.Server.GRPCPort, 12307),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 60),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 60),
		},
		Database: DatabaseConfig{
			Type:            getOrDefault(configFile.Database.Type, "sqlite"),
			Host:            getOrDefault(configFile.Database.Host, "localhost"),
			Port:            getIntOrDefault(configFile.Database.Port, 5432),
			User:            getOrDefault(configFile.Database.Username, ""),
			Password:        getOrDefault(configFile.Database.Password, ""),
			Name:            getOrDefault(configFile.Database.Database, "tiga.db"),
			SSLMode:         getOrDefault(configFile.Database.SSLMode, "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 3600),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:    getOrDefault(configFile.Security.JWTSecret, getEnv("JWT_SECRET", "your-secret-key-change-in-production")),
			ExpiresIn: getEnvAsDuration("JWT_EXPIRES_IN", 24*time.Hour),
		},
		OAuth: OAuthConfig{
			Google: OAuthProvider{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			},
			GitHub: OAuthProvider{
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
			},
		},
		Security: SecurityConfig{
			EncryptionKey: configFile.Security.EncryptionKey,
			BcryptCost:    getEnvAsInt("BCRYPT_COST", 10),
		},
		DatabaseManagement: DatabaseManagementConfig{
			CredentialKey:       configFile.DatabaseManagement.CredentialKey,
			QueryTimeoutSeconds: getIntOrDefault(configFile.DatabaseManagement.QueryTimeoutSeconds, 30),
			MaxResultBytes:      getIntOrDefault(configFile.DatabaseManagement.MaxResultBytes, 10*1024*1024),
			AuditRetentionDays:  getIntOrDefault(configFile.DatabaseManagement.AuditRetentionDays, 90),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	return config, nil
}

// ConfigFile represents the YAML configuration file structure
type ConfigFile struct {
	Server struct {
		InstallLock bool   `yaml:"install_lock"`
		AppName     string `yaml:"app_name"`
		AppSubtitle string `yaml:"app_subtitle"`
		Domain      string `yaml:"domain"`
		HTTPPort    int    `yaml:"http_port"`
		GRPCDomain  string `yaml:"grpc_domain"`
		GRPCPort    int    `yaml:"grpc_port"`
		TLSEnabled  bool   `yaml:"tls_enabled"`
	} `yaml:"server"`

	Database struct {
		Type     string `yaml:"type"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Database string `yaml:"database"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Charset  string `yaml:"charset"`
		SSLMode  string `yaml:"ssl_mode"`
	} `yaml:"database"`

	Security struct {
		JWTSecret     string `yaml:"jwt_secret"`
		EncryptionKey string `yaml:"encryption_key"`
	} `yaml:"security"`

	DatabaseManagement struct {
		CredentialKey       string `yaml:"credential_key"`
		QueryTimeoutSeconds int    `yaml:"query_timeout_seconds"`
		MaxResultBytes      int    `yaml:"max_result_bytes"`
		AuditRetentionDays  int    `yaml:"audit_retention_days"`
	} `yaml:"database_management"`
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := &Config{
		Server: ServerConfig{
			Debug:        getEnvAsBool("DEBUG", false),
			Port:         getEnvAsInt("SERVER_PORT", 12306),
			GRPCDomain:   getEnv("SERVER_GRPC_DOMAIN", ""),
			GRPCPort:     getEnvAsInt("SERVER_GRPC_PORT", 12307),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 60),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 60),
		},
		Database: DatabaseConfig{
			Type:            getEnv("DB_TYPE", "sqlite"),
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "devops"),
			Password:        getEnv("DB_PASSWORD", "devops123"),
			Name:            getEnv("DB_NAME", "tiga.db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 3600),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:    getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			ExpiresIn: getEnvAsDuration("JWT_EXPIRES_IN", 24*time.Hour),
		},
		OAuth: OAuthConfig{
			Google: OAuthProvider{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			},
			GitHub: OAuthProvider{
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
			},
		},
		Security: SecurityConfig{
			EncryptionKey: "",
			BcryptCost:    getEnvAsInt("BCRYPT_COST", 10),
		},
		DatabaseManagement: DatabaseManagementConfig{
			CredentialKey:       "",
			QueryTimeoutSeconds: getEnvAsInt("DB_MGMT_QUERY_TIMEOUT", 30),
			MaxResultBytes:      getEnvAsInt("DB_MGMT_MAX_RESULT_BYTES", 10*1024*1024),
			AuditRetentionDays:  getEnvAsInt("DB_MGMT_AUDIT_RETENTION_DAYS", 90),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "debug"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	return config
}

// DSN returns the database connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

// RedisAddr returns the Redis connection address
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// QueryTimeout returns the configured query timeout or default (30s).
func (c DatabaseManagementConfig) QueryTimeout() time.Duration {
	if c.QueryTimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.QueryTimeoutSeconds) * time.Second
}

// ResultSizeLimit returns the query result size cap or default (10MB).
func (c DatabaseManagementConfig) ResultSizeLimit() int64 {
	if c.MaxResultBytes <= 0 {
		return 10 * 1024 * 1024
	}
	return int64(c.MaxResultBytes)
}

// AuditRetention returns the audit retention duration or default (90 days).
func (c DatabaseManagementConfig) AuditRetention() time.Duration {
	days := c.AuditRetentionDays
	if days <= 0 {
		days = 90
	}
	return time.Duration(days) * 24 * time.Hour
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getOrDefault returns the value if not empty, otherwise returns the default
func getOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

// getIntOrDefault returns the value if not zero, otherwise returns the default
func getIntOrDefault(value, defaultValue int) int {
	if value != 0 {
		return value
	}
	return defaultValue
}

// InstallChecker provides interface to check installation status
type InstallChecker interface {
	IsInstalled() bool
}

// NewInstallConfigService creates a service to check if system is installed
func NewInstallConfigService(configPath string) InstallChecker {
	return &installChecker{configPath: configPath}
}

type installChecker struct {
	configPath string
}

func (c *installChecker) IsInstalled() bool {
	// Check if config file exists
	if _, err := os.Stat(c.configPath); os.IsNotExist(err) {
		return false
	}

	// Read config file
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return false
	}

	// Parse YAML to check install_lock
	var config struct {
		Server struct {
			InstallLock bool `yaml:"install_lock"`
		} `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return false
	}

	return config.Server.InstallLock
}
