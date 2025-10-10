package services

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/install/models"

	mainmodels "github.com/ysicing/tiga/internal/models"
)

// T022: 安装服务编排

// InstallService 安装服务
type InstallService struct {
	configService     *ConfigService
	databaseService   *DatabaseService
	validationService *ValidationService
}

// NewInstallService 创建安装服务
func NewInstallService(configPath string) *InstallService {
	return &InstallService{
		configService:     NewConfigService(configPath),
		databaseService:   NewDatabaseService(),
		validationService: NewValidationService(),
	}
}

// IsInstalled 检查是否已初始化
func (s *InstallService) IsInstalled() bool {
	return s.configService.IsInstalled()
}

// CheckDatabase 检查数据库连接
func (s *InstallService) CheckDatabase(req models.CheckDatabaseRequest) (*models.CheckDBResponse, error) {
	// 验证数据库配置
	if errors := s.validationService.ValidateDatabase(req.DatabaseConfig); len(errors) > 0 {
		return nil, fmt.Errorf("invalid database configuration: %v", errors)
	}

	// 测试连接
	if err := s.databaseService.TestConnection(req.DatabaseConfig); err != nil {
		return &models.CheckDBResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to connect: %v", err),
		}, nil
	}

	// 检查现有数据
	hasData, version, err := s.databaseService.CheckExistingData(req.DatabaseConfig)
	if err != nil {
		return &models.CheckDBResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to check existing data: %v", err),
		}, nil
	}

	// 如果有数据但没有确认重新安装，返回成功但标记需要确认
	if hasData && !req.ConfirmReinstall {
		return &models.CheckDBResponse{
			Success:         true,
			HasExistingData: true,
			SchemaVersion:   version,
			CanUpgrade:      true,
		}, nil
	}

	// 如果有数据且已确认重新安装，或没有数据
	return &models.CheckDBResponse{
		Success:         true,
		HasExistingData: hasData,
		SchemaVersion:   version,
	}, nil
}

// ValidateAdmin 验证管理员账户
func (s *InstallService) ValidateAdmin(admin models.AdminAccount) *models.ValidateAdminResponse {
	errors := s.validationService.ValidateAdmin(admin)

	if len(errors) > 0 {
		return &models.ValidateAdminResponse{
			Valid:  false,
			Errors: errors,
		}
	}

	return &models.ValidateAdminResponse{
		Valid: true,
	}
}

// ValidateSettings 验证系统设置
func (s *InstallService) ValidateSettings(settings models.SystemSettings) *models.ValidateSettingsResponse {
	errors := s.validationService.ValidateSettings(settings)

	if len(errors) > 0 {
		return &models.ValidateSettingsResponse{
			Valid:  false,
			Errors: errors,
		}
	}

	return &models.ValidateSettingsResponse{
		Valid: true,
	}
}

// Finalize 完成初始化
func (s *InstallService) Finalize(req models.FinalizeRequest) (*models.FinalizeResponse, error) {
	// 1. 检查是否已初始化
	if s.IsInstalled() {
		return &models.FinalizeResponse{
			Success: false,
			Error:   "Installation already completed",
		}, fmt.Errorf("installation already completed")
	}

	// 2. 验证所有配置
	if errors := s.validationService.ValidateDatabase(req.Database); len(errors) > 0 {
		return nil, fmt.Errorf("invalid database configuration: %v", errors)
	}
	if errors := s.validationService.ValidateAdmin(req.Admin); len(errors) > 0 {
		return nil, fmt.Errorf("invalid admin account: %v", errors)
	}
	if errors := s.validationService.ValidateSettings(req.Settings); len(errors) > 0 {
		return nil, fmt.Errorf("invalid settings: %v", errors)
	}

	// 3. 检查现有数据
	hasData, _, err := s.databaseService.CheckExistingData(req.Database)
	if err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to check existing data: %v", err),
		}, err
	}

	if hasData && !req.ConfirmReinstall {
		return &models.FinalizeResponse{
			Success: false,
			Error:   "Existing data found. Set confirm_reinstall to true to proceed.",
		}, fmt.Errorf("existing data found")
	}

	// 4. 连接数据库
	db, err := s.databaseService.Connect(req.Database)
	if err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to connect to database: %v", err),
		}, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get database connection: %v", err),
		}, err
	}
	defer sqlDB.Close()

	// 5. 运行迁移（在事务外部执行，避免 SQLite DDL 事务问题）
	if err := s.databaseService.RunMigrations(db); err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to run migrations: %v", err),
		}, err
	}

	// 6. 执行数据初始化事务
	err = db.Transaction(func(tx *gorm.DB) error {
		// 6.1 创建管理员账户
		if err := s.validationService.CreateAdminAccount(tx, req.Admin); err != nil {
			return fmt.Errorf("failed to create admin account: %w", err)
		}

		// 6.2 保存系统配置到数据库 (key-value 格式)
		grpcPort := req.Settings.GRPCPort
		if grpcPort == 0 {
			grpcPort = 12307
		}

		configs := []struct {
			Key       string
			Value     interface{}
			ValueType string
		}{
			{"install_lock", true, "boolean"},
			{"app_name", req.Settings.AppName, "string"},
			{"app_subtitle", req.Settings.AppSubtitle, "string"},
			{"domain", req.Settings.Domain, "string"},
			{"http_port", req.Settings.HTTPPort, "number"},
			{"grpc_port", grpcPort, "number"},
			{"language", req.Settings.Language, "string"},
			{"enable_analytics", req.Settings.EnableAnalytics, "boolean"},
		}

		for _, cfg := range configs {
			systemConfig := mainmodels.SystemConfig{
				Key:       cfg.Key,
				Value:     mainmodels.JSONB{"value": cfg.Value},
				ValueType: cfg.ValueType,
			}
			if err := tx.Create(&systemConfig).Error; err != nil {
				return fmt.Errorf("failed to save config %s: %w", cfg.Key, err)
			}
		}

		return nil
	})

	if err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 6. 生成配置文件（会生成 JWT Secret 和加密密钥）
	installConfig := models.InstallConfig{
		Database: req.Database,
		Admin:    req.Admin,
		Settings: req.Settings,
	}
	if err := s.configService.GenerateConfigFile(installConfig); err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to generate config file: %v", err),
		}, err
	}

	// 7. 读取生成的加密密钥并保存到数据库
	generatedConfig, err := s.configService.LoadConfig()
	if err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to load generated config: %v", err),
		}, err
	}

	// 更新数据库中的加密密钥
	err = db.Transaction(func(tx *gorm.DB) error {
		encryptionConfig := mainmodels.SystemConfig{
			Key:       "encryption_key",
			Value:     mainmodels.JSONB{"value": generatedConfig.Security.EncryptionKey},
			ValueType: "string",
		}
		if err := tx.Create(&encryptionConfig).Error; err != nil {
			return fmt.Errorf("failed to save encryption key: %w", err)
		}
		return nil
	})
	if err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save encryption key: %v", err),
		}, err
	}

	// 8. 生成 session token
	token, err := s.generateSessionToken(req.Admin.Username)
	if err != nil {
		return &models.FinalizeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to generate session token: %v", err),
		}, err
	}

	// 9. 构建重定向 URL（使用配置的端口）
	protocol := "http"
	domain := req.Settings.Domain
	if domain == "" {
		domain = "localhost"
	}
	port := req.Settings.HTTPPort
	if port == 0 {
		port = 8080
	}

	redirectURL := fmt.Sprintf("%s://%s:%d/login", protocol, domain, port)

	return &models.FinalizeResponse{
		Success:        true,
		Message:        "Installation completed successfully",
		SessionToken:   token,
		RedirectURL:    redirectURL,
		NeedsRestart:   true,
		RestartMessage: "Please restart the application to apply changes",
	}, nil
}

// generateSessionToken 生成 JWT session token
func (s *InstallService) generateSessionToken(username string) (string, error) {
	// 从已生成的配置文件中读取 JWT secret
	config, err := s.configService.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	claims := jwt.MapClaims{
		"username": username,
		"is_admin": true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Security.JWTSecret))
}
