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
	Kubernetes         KubernetesConfig
	Prometheus         PrometheusConfig
	Webhook            WebhookConfig
	Features           FeaturesConfig
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

// KubernetesConfig holds Kubernetes-related configuration (Phase 0 扩展)
type KubernetesConfig struct {
	NodeTerminalImage   string // Docker image for node terminal pods
	NodeTerminalPodName string // Name prefix for node terminal pods
	EnableKruise        bool   // Enable OpenKruise CRD support
	EnableTailscale     bool   // Enable Tailscale CRD support
	EnableTraefik       bool   // Enable Traefik CRD support
	EnableK3sUpgrade    bool   // Enable K3s Upgrade Controller CRD support
}

// PrometheusConfig holds Prometheus integration configuration (Phase 0 新增)
type PrometheusConfig struct {
	AutoDiscovery    bool              // Enable automatic Prometheus discovery
	DiscoveryTimeout int               // Discovery timeout in seconds (default: 30)
	ClusterURLs      map[string]string // Manual Prometheus URLs per cluster name
}

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	Enabled  bool
	Username string
	Password string
}

// FeaturesConfig holds feature flags (Phase 0 扩展)
type FeaturesConfig struct {
	AnonymousUserEnabled bool
	DisableGZIP          bool
	DisableVersionCheck  bool
	ReadonlyMode         bool // Enable K8s readonly mode (disables write operations)
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
			Secret:    getOrDefault(configFile.Security.JWTSecret, getEnv("JWT_SECRET", "")),
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
		Kubernetes: KubernetesConfig{
			NodeTerminalImage:   getOrDefault(configFile.Kubernetes.NodeTerminalImage, getEnv("NODE_TERMINAL_IMAGE", "busybox:latest")),
			NodeTerminalPodName: "tiga-node-terminal-agent",
			EnableKruise:        getBoolOrDefault(configFile.Kubernetes.EnableKruise, getEnvAsBool("K8S_ENABLE_KRUISE", true)),
			EnableTailscale:     getBoolOrDefault(configFile.Kubernetes.EnableTailscale, getEnvAsBool("K8S_ENABLE_TAILSCALE", false)),
			EnableTraefik:       getBoolOrDefault(configFile.Kubernetes.EnableTraefik, getEnvAsBool("K8S_ENABLE_TRAEFIK", true)),
			EnableK3sUpgrade:    getBoolOrDefault(configFile.Kubernetes.EnableK3sUpgrade, getEnvAsBool("K8S_ENABLE_K3S_UPGRADE", false)),
		},
		Prometheus: PrometheusConfig{
			AutoDiscovery:    getBoolOrDefault(configFile.Prometheus.AutoDiscovery, getEnvAsBool("PROMETHEUS_AUTO_DISCOVERY", true)),
			DiscoveryTimeout: getIntOrDefault(configFile.Prometheus.DiscoveryTimeout, getEnvAsInt("PROMETHEUS_DISCOVERY_TIMEOUT", 30)),
			ClusterURLs:      configFile.Prometheus.ClusterURLs,
		},
		Webhook: WebhookConfig{
			Username: getEnv("WEBHOOK_USERNAME", ""),
			Password: getEnv("WEBHOOK_PASSWORD", ""),
			Enabled:  getEnv("WEBHOOK_USERNAME", "") != "" && getEnv("WEBHOOK_PASSWORD", "") != "",
		},
		Features: FeaturesConfig{
			AnonymousUserEnabled: getEnvAsBool("ANONYMOUS_USER_ENABLED", false),
			DisableGZIP:          getEnvAsBool("DISABLE_GZIP", true),
			DisableVersionCheck:  getEnvAsBool("DISABLE_VERSION_CHECK", false),
			ReadonlyMode:         getBoolOrDefault(configFile.Features.ReadonlyMode, getEnvAsBool("K8S_READONLY_MODE", false)),
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

	Kubernetes struct {
		NodeTerminalImage string `yaml:"node_terminal_image"`
		EnableKruise      bool   `yaml:"enable_kruise"`
		EnableTailscale   bool   `yaml:"enable_tailscale"`
		EnableTraefik     bool   `yaml:"enable_traefik"`
		EnableK3sUpgrade  bool   `yaml:"enable_k3s_upgrade"`
	} `yaml:"kubernetes"`

	Prometheus struct {
		AutoDiscovery    bool              `yaml:"auto_discovery"`
		DiscoveryTimeout int               `yaml:"discovery_timeout"`
		ClusterURLs      map[string]string `yaml:"cluster_urls"`
	} `yaml:"prometheus"`

	Features struct {
		ReadonlyMode bool `yaml:"readonly_mode"`
	} `yaml:"features"`
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
			Secret:    getEnv("JWT_SECRET", ""),
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
			EncryptionKey: getEnv("ENCRYPTION_KEY", ""),
			BcryptCost:    getEnvAsInt("BCRYPT_COST", 10),
		},
		DatabaseManagement: DatabaseManagementConfig{
			CredentialKey:       getEnv("CREDENTIAL_KEY", ""),
			QueryTimeoutSeconds: getEnvAsInt("DB_MGMT_QUERY_TIMEOUT", 30),
			MaxResultBytes:      getEnvAsInt("DB_MGMT_MAX_RESULT_BYTES", 10*1024*1024),
			AuditRetentionDays:  getEnvAsInt("DB_MGMT_AUDIT_RETENTION_DAYS", 90),
		},
		Kubernetes: KubernetesConfig{
			NodeTerminalImage:   getEnv("NODE_TERMINAL_IMAGE", "busybox:latest"),
			NodeTerminalPodName: "tiga-node-terminal-agent",
			EnableKruise:        getEnvAsBool("K8S_ENABLE_KRUISE", true),
			EnableTailscale:     getEnvAsBool("K8S_ENABLE_TAILSCALE", false),
			EnableTraefik:       getEnvAsBool("K8S_ENABLE_TRAEFIK", true),
			EnableK3sUpgrade:    getEnvAsBool("K8S_ENABLE_K3S_UPGRADE", false),
		},
		Prometheus: PrometheusConfig{
			AutoDiscovery:    getEnvAsBool("PROMETHEUS_AUTO_DISCOVERY", true),
			DiscoveryTimeout: getEnvAsInt("PROMETHEUS_DISCOVERY_TIMEOUT", 30),
			ClusterURLs:      make(map[string]string), // Empty map from env
		},
		Webhook: WebhookConfig{
			Username: getEnv("WEBHOOK_USERNAME", ""),
			Password: getEnv("WEBHOOK_PASSWORD", ""),
			Enabled:  getEnv("WEBHOOK_USERNAME", "") != "" && getEnv("WEBHOOK_PASSWORD", "") != "",
		},
		Features: FeaturesConfig{
			AnonymousUserEnabled: getEnvAsBool("ANONYMOUS_USER_ENABLED", false),
			DisableGZIP:          getEnvAsBool("DISABLE_GZIP", true),
			DisableVersionCheck:  getEnvAsBool("DISABLE_VERSION_CHECK", false),
			ReadonlyMode:         getEnvAsBool("K8S_READONLY_MODE", false),
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

// getBoolOrDefault returns the value from YAML if explicitly set, otherwise returns the default
func getBoolOrDefault(yamlValue bool, defaultValue bool) bool {
	// In YAML, if field is not set, it defaults to false for bool
	// So we use the YAML value if it's true, otherwise use the default
	// This is a simplified approach; for proper handling, we'd need pointers in ConfigFile
	return yamlValue || defaultValue
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

// Validate validates the configuration and returns errors for critical missing values
func (c *Config) Validate() error {
	var errors []string

	// Validate JWT Secret
	if c.JWT.Secret == "" {
		errors = append(errors, "JWT_SECRET is not set (required for authentication)")
	} else if len(c.JWT.Secret) < 32 {
		errors = append(errors, "JWT_SECRET must be at least 32 characters")
	}

	// Validate Encryption Key for database management
	if c.Security.EncryptionKey != "" && len(c.Security.EncryptionKey) != 44 {
		// Base64-encoded 32-byte key is 44 characters
		errors = append(errors, "ENCRYPTION_KEY must be a base64-encoded 32-byte key (44 characters)")
	}

	if c.DatabaseManagement.CredentialKey != "" && len(c.DatabaseManagement.CredentialKey) != 44 {
		errors = append(errors, "CREDENTIAL_KEY must be a base64-encoded 32-byte key (44 characters)")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n- %s",
			fmt.Sprintf("%s", errors))
	}

	return nil
}

// ValidateOrExit validates the configuration and exits if validation fails
func (c *Config) ValidateOrExit() {
	if err := c.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Configuration Error:\n%v\n\n", err)
		fmt.Fprintf(os.Stderr, "🔑 To generate secure keys, run:\n")
		fmt.Fprintf(os.Stderr, "  JWT_SECRET:       openssl rand -base64 48\n")
		fmt.Fprintf(os.Stderr, "  ENCRYPTION_KEY:   openssl rand -base64 32\n")
		fmt.Fprintf(os.Stderr, "\nThen set them in config.yaml or as environment variables.\n")
		os.Exit(1)
	}
}
