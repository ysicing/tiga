package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/models"
)

// Database represents the database connection
type Database struct {
	DB *gorm.DB
}

// NewDatabase creates a new database connection supporting PostgreSQL, MySQL, and SQLite
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	// Configure GORM logger to show SQL queries
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level: Silent, Error, Warn, Info
			IgnoreRecordNotFoundError: false,       // Don't ignore ErrRecordNotFound error
			Colorful:                  true,        // Enable color
		},
	)

	gormConfig := &gorm.Config{
		Logger: newLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	var db *gorm.DB
	var err error

	// Detect database type and create appropriate connection
	// Default to SQLite for ease of use
	dbType := cfg.Type
	if dbType == "" {
		dbType = "sqlite"
	}

	switch dbType {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host,
			cfg.Port,
			cfg.User,
			cfg.Password,
			cfg.Name,
			cfg.SSLMode,
		)
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)

	case "mysql":
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.Name,
		)
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)

	case "sqlite":
		// For SQLite, use Name as the database file path
		dsn := cfg.Name
		if dsn == "" {
			dsn = "tiga.db"
		}
		logrus.Infof("SQLite database path: %s", dsn)
		db, err = gorm.Open(sqlite.Open(dsn), gormConfig)

	default:
		return nil, fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, sqlite)", dbType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s database: %w", dbType, err)
	}

	// Get underlying SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings (not applicable for SQLite)
	if dbType != "sqlite" {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign key enforcement for SQLite
	if dbType == "sqlite" {
		if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			return nil, fmt.Errorf("failed to enable sqlite foreign keys: %w", err)
		}
	}

	logrus.Infof("Database connection established successfully (type: %s)", dbType)

	return &Database{DB: db}, nil
}

// AutoMigrate runs database migrations
func (d *Database) AutoMigrate() error {
	logrus.Info("Running database migrations...")

	// Run migrations for all models
	err := d.DB.AutoMigrate(
		// User and authentication
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.Session{},
		&models.OAuthProvider{},

		// Instances and monitoring
		&models.Instance{},
		&models.InstanceSnapshot{},
		&models.Metric{},
		&models.Alert{},
		&models.AlertEvent{},

		// Operations
		&models.Backup{},
		&models.BackgroundTask{},
		&models.Event{},

		// Audit and configuration
		&models.AuditLog{},
		&models.SystemConfig{},

		// Kubernetes management
		&models.Cluster{},
		&models.ResourceHistory{},

		// Host monitoring subsystem (Nezha-inspired)
		&models.HostNode{},
		&models.HostInfo{},
		&models.HostState{},
		&models.ServiceMonitor{},
		&models.ServiceProbeResult{},
		&models.ServiceAvailability{},
		&models.ServiceHistory{}, // 30-day aggregated service history
		&models.WebSSHSession{},
		&models.HostActivityLog{},
		&models.MonitorAlertRule{},
		&models.MonitorAlertEvent{},
		&models.AgentConnection{},

		// MinIO subsystem
		&models.MinIOInstance{},
		&models.MinIOUser{},
		&models.BucketPermission{},
		&models.MinIOShareLink{},
		&models.MinIOAuditLog{},

		// Database management
		&models.DatabaseInstance{},
		&models.Database{},
		&models.DatabaseUser{},
		&models.PermissionPolicy{},
		&models.QuerySession{},
		&models.DatabaseAuditLog{},
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Apply database-specific post-migration optimizations
	if err := (&models.ResourceHistory{}).AfterMigrate(d.DB); err != nil {
		logrus.Warnf("Failed to create resource history indexes: %v", err)
	}

	logrus.Info("Database migrations completed successfully")
	return nil
}

// SeedDefaultData creates default data if not exists
func (d *Database) SeedDefaultData() error {
	logrus.Info("Seeding default data...")

	// Note: Default host groups are no longer needed as we use simple string grouping
	// Hosts will default to "默认分组" via model BeforeCreate hook

	logrus.Info("Default data seeding completed")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
