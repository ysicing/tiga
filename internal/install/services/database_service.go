package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/install/models"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/glebarez/go-sqlite"

	mainmodels "github.com/ysicing/tiga/internal/models"
)

// T020: 数据库连接验证服务

// DatabaseService 数据库服务
type DatabaseService struct{}

// NewDatabaseService 创建数据库服务
func NewDatabaseService() *DatabaseService {
	return &DatabaseService{}
}

// TestConnection 测试数据库连接
func (s *DatabaseService) TestConnection(config models.DatabaseConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dsn := s.buildDSN(config)
	var sqlDB *sql.DB
	var err error

	switch config.Type {
	case "mysql":
		sqlDB, err = sql.Open("mysql", dsn)
	case "postgresql":
		sqlDB, err = sql.Open("postgres", dsn)
	case "sqlite":
		sqlDB, err = sql.Open("sqlite3", dsn)
	default:
		return fmt.Errorf("unsupported database type: %s", config.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("connection refused: %w", err)
	}

	return nil
}

// CheckExistingData 检查数据库是否已有 Tiga 数据
func (s *DatabaseService) CheckExistingData(config models.DatabaseConfig) (hasData bool, version string, err error) {
	db, err := s.Connect(config)
	if err != nil {
		return false, "", err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return false, "", err
	}
	defer sqlDB.Close()

	// 检查 users 表是否存在
	var tableExists bool
	switch config.Type {
	case "postgresql":
		err = db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists).Error
	case "mysql":
		err = db.Raw("SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_name = 'users' AND table_schema = DATABASE()").Scan(&tableExists).Error
	case "sqlite":
		err = db.Raw("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableExists).Error
	}

	if err != nil {
		return false, "", err
	}

	if !tableExists {
		return false, "", nil
	}

	// 检查是否有数据
	var count int64
	if err := db.Table("users").Count(&count).Error; err != nil {
		return false, "", err
	}

	if count > 0 {
		// TODO: 从 system_config 表读取版本号
		return true, "1.0.0", nil
	}

	return false, "", nil
}

// Connect 连接到数据库
func (s *DatabaseService) Connect(config models.DatabaseConfig) (*gorm.DB, error) {
	dsn := s.buildDSN(config)
	var db *gorm.DB
	var err error

	switch config.Type {
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	case "postgresql":
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// buildDSN 构建数据库连接字符串
func (s *DatabaseService) buildDSN(config models.DatabaseConfig) string {
	switch config.Type {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			config.Username,
			config.Password,
			config.Host,
			config.Port,
			config.Database,
		)
		if config.Charset != "" {
			dsn += "?charset=" + config.Charset
		} else {
			dsn += "?charset=utf8mb4&parseTime=True&loc=Local"
		}
		return dsn

	case "postgresql":
		sslMode := config.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host,
			config.Port,
			config.Username,
			config.Password,
			config.Database,
			sslMode,
		)

	case "sqlite":
		return config.Database

	default:
		return ""
	}
}

// RunMigrations 运行数据库迁移
func (s *DatabaseService) RunMigrations(db *gorm.DB) error {
	// Use main application models for complete schema
	return db.AutoMigrate(
		// User and authentication
		&mainmodels.User{},
		&mainmodels.Role{},
		&mainmodels.UserRole{},
		&mainmodels.Session{},
		&mainmodels.OAuthProvider{},

		// Instances and monitoring
		&mainmodels.Instance{},
		&mainmodels.InstanceSnapshot{},
		&mainmodels.Metric{},
		&mainmodels.Alert{},
		&mainmodels.AlertEvent{},

		// Operations
		&mainmodels.Backup{},
		&mainmodels.BackgroundTask{},
		&mainmodels.Event{},

		// Audit and configuration
		&mainmodels.AuditLog{},
		&mainmodels.SystemConfig{},

		// Kubernetes management
		&mainmodels.Cluster{},
		&mainmodels.ResourceHistory{},
	)
}
