package managers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ysicing/tiga/internal/models"

	_ "github.com/lib/pq"
)

// PostgreSQLManager manages PostgreSQL instances
type PostgreSQLManager struct {
	*BaseManager
	db *sql.DB
}

// NewPostgreSQLManager creates a new PostgreSQL manager
func NewPostgreSQLManager() *PostgreSQLManager {
	return &PostgreSQLManager{
		BaseManager: NewBaseManager(),
	}
}

// Initialize initializes the PostgreSQL manager
func (m *PostgreSQLManager) Initialize(ctx context.Context, instance *models.Instance) error {
	if err := m.BaseManager.Initialize(ctx, instance); err != nil {
		return err
	}
	return nil
}

// Connect establishes connection to PostgreSQL
func (m *PostgreSQLManager) Connect(ctx context.Context) error {
	connection := m.instance.Connection
	host, _ := connection["host"].(string)
	port, _ := connection["port"].(float64)

	username := m.GetConfigValue("username", "postgres").(string)
	password := m.GetConfigValue("password", "").(string)
	database := m.GetConfigValue("database", "postgres").(string)
	sslMode := m.GetConfigValue("ssl_mode", "disable").(string)

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
		host, int(port), username, password, database, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Set connection pool settings
	maxOpenConns := m.GetConfigValue("max_open_conns", 10).(int)
	maxIdleConns := m.GetConfigValue("max_idle_conns", 5).(int)
	connMaxLifetime := m.GetConfigValue("conn_max_lifetime", 3600).(int)

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	m.db = db
	return nil
}

// Disconnect closes connection to PostgreSQL
func (m *PostgreSQLManager) Disconnect(ctx context.Context) error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// HealthCheck checks PostgreSQL health
func (m *PostgreSQLManager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	start := time.Now()
	status := &HealthStatus{
		Healthy:       false,
		LastCheckTime: time.Now().Format(time.RFC3339),
		Details:       make(map[string]interface{}),
	}

	// Ping database
	if err := m.db.PingContext(ctx); err != nil {
		status.Message = fmt.Sprintf("Health check failed: %v", err)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status, nil
	}

	// Get PostgreSQL version
	var version string
	if err := m.db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err == nil {
		status.Details["version"] = version
	}

	// Get active connections
	var activeConnections int
	query := "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'"
	if err := m.db.QueryRowContext(ctx, query).Scan(&activeConnections); err == nil {
		status.Details["active_connections"] = activeConnections
	}

	// Get connection stats
	stats := m.db.Stats()
	status.Details["open_connections"] = stats.OpenConnections
	status.Details["in_use"] = stats.InUse
	status.Details["idle"] = stats.Idle

	status.Healthy = true
	status.Message = "PostgreSQL is healthy"
	status.ResponseTime = time.Since(start).Milliseconds()

	return status, nil
}

// CollectMetrics collects metrics from PostgreSQL
func (m *PostgreSQLManager) CollectMetrics(ctx context.Context) (*ServiceMetrics, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	metrics := &ServiceMetrics{
		InstanceID: m.instance.ID,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metrics:    make(map[string]interface{}),
	}

	// Get connection stats
	var totalConnections, activeConnections, idleConnections int
	m.db.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity").Scan(&totalConnections)
	m.db.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'").Scan(&activeConnections)
	m.db.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity WHERE state = 'idle'").Scan(&idleConnections)

	metrics.Metrics["total_connections"] = totalConnections
	metrics.Metrics["active_connections"] = activeConnections
	metrics.Metrics["idle_connections"] = idleConnections

	// Get database size
	database := m.GetConfigValue("database", "postgres").(string)
	var dbSize int64
	query := "SELECT pg_database_size($1)"
	if err := m.db.QueryRowContext(ctx, query, database).Scan(&dbSize); err == nil {
		metrics.Metrics["database_size_bytes"] = dbSize
	}

	// Get cache hit ratio
	var cacheHitRatio float64
	query = `SELECT
		sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as cache_hit_ratio
		FROM pg_statio_user_tables
		WHERE heap_blks_hit + heap_blks_read > 0`
	if err := m.db.QueryRowContext(ctx, query).Scan(&cacheHitRatio); err == nil {
		metrics.Metrics["cache_hit_ratio"] = cacheHitRatio
	}

	// Get transaction stats
	var commits, rollbacks int64
	query = "SELECT sum(xact_commit), sum(xact_rollback) FROM pg_stat_database"
	if err := m.db.QueryRowContext(ctx, query).Scan(&commits, &rollbacks); err == nil {
		metrics.Metrics["transactions_committed"] = commits
		metrics.Metrics["transactions_rolled_back"] = rollbacks
	}

	// Get connection pool stats
	stats := m.db.Stats()
	metrics.Metrics["pool_open_connections"] = stats.OpenConnections
	metrics.Metrics["pool_in_use"] = stats.InUse
	metrics.Metrics["pool_idle"] = stats.Idle
	metrics.Metrics["pool_wait_count"] = stats.WaitCount
	metrics.Metrics["pool_wait_duration_ms"] = stats.WaitDuration.Milliseconds()

	return metrics, nil
}

// GetInfo returns PostgreSQL service information
func (m *PostgreSQLManager) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	info := make(map[string]interface{})

	// Get version
	var version string
	if err := m.db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err == nil {
		info["version"] = version
	}

	// Get databases
	rows, err := m.db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	databases := make([]string, 0)
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}
		databases = append(databases, dbName)
	}
	info["databases"] = databases

	// Get max connections
	var maxConnections int
	if err := m.db.QueryRowContext(ctx, "SHOW max_connections").Scan(&maxConnections); err == nil {
		info["max_connections"] = maxConnections
	}

	// Get current database
	var currentDB string
	if err := m.db.QueryRowContext(ctx, "SELECT current_database()").Scan(&currentDB); err == nil {
		info["current_database"] = currentDB
	}

	// Get extensions
	rows, err = m.db.QueryContext(ctx, "SELECT extname, extversion FROM pg_extension")
	if err == nil {
		defer rows.Close()
		extensions := make([]map[string]string, 0)
		for rows.Next() {
			var name, version string
			if err := rows.Scan(&name, &version); err != nil {
				continue
			}
			extensions = append(extensions, map[string]string{
				"name":    name,
				"version": version,
			})
		}
		info["extensions"] = extensions
	}

	return info, nil
}

// ValidateConfig validates PostgreSQL configuration
func (m *PostgreSQLManager) ValidateConfig(config map[string]interface{}) error {
	username, ok := config["username"].(string)
	if !ok || username == "" {
		return fmt.Errorf("%w: username is required", ErrInvalidConfig)
	}

	password, ok := config["password"].(string)
	if !ok || password == "" {
		return fmt.Errorf("%w: password is required", ErrInvalidConfig)
	}

	return nil
}

// Type returns the service type
func (m *PostgreSQLManager) Type() string {
	return "postgres"
}
