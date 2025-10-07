package managers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ysicing/tiga/internal/models"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLManager manages MySQL instances
type MySQLManager struct {
	*BaseManager
	db *sql.DB
}

// NewMySQLManager creates a new MySQL manager
func NewMySQLManager() *MySQLManager {
	return &MySQLManager{
		BaseManager: NewBaseManager(),
	}
}

// Initialize initializes the MySQL manager
func (m *MySQLManager) Initialize(ctx context.Context, instance *models.Instance) error {
	if err := m.BaseManager.Initialize(ctx, instance); err != nil {
		return err
	}
	return nil
}

// Connect establishes connection to MySQL
func (m *MySQLManager) Connect(ctx context.Context) error {
	connection := m.instance.Connection
	host, _ := connection["host"].(string)
	port, _ := connection["port"].(float64)

	username := m.GetConfigValue("username", "root").(string)
	password := m.GetConfigValue("password", "").(string)
	database := m.GetConfigValue("database", "").(string)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=10s",
		username, password, host, int(port), database)

	db, err := sql.Open("mysql", dsn)
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

// Disconnect closes connection to MySQL
func (m *MySQLManager) Disconnect(ctx context.Context) error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// HealthCheck checks MySQL health
func (m *MySQLManager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
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

	// Get MySQL version
	var version string
	if err := m.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err == nil {
		status.Details["version"] = version
	}

	// Get connection stats
	stats := m.db.Stats()
	status.Details["open_connections"] = stats.OpenConnections
	status.Details["in_use"] = stats.InUse
	status.Details["idle"] = stats.Idle

	status.Healthy = true
	status.Message = "MySQL is healthy"
	status.ResponseTime = time.Since(start).Milliseconds()

	return status, nil
}

// CollectMetrics collects metrics from MySQL
func (m *MySQLManager) CollectMetrics(ctx context.Context) (*ServiceMetrics, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	metrics := &ServiceMetrics{
		InstanceID: m.instance.ID,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metrics:    make(map[string]interface{}),
	}

	// Get global status
	rows, err := m.db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricsCollectionFailed, err)
	}
	defer rows.Close()

	statusMap := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		statusMap[name] = value
	}

	// Extract key metrics
	if val, ok := statusMap["Threads_connected"]; ok {
		metrics.Metrics["threads_connected"] = val
	}
	if val, ok := statusMap["Threads_running"]; ok {
		metrics.Metrics["threads_running"] = val
	}
	if val, ok := statusMap["Questions"]; ok {
		metrics.Metrics["questions"] = val
	}
	if val, ok := statusMap["Slow_queries"]; ok {
		metrics.Metrics["slow_queries"] = val
	}
	if val, ok := statusMap["Bytes_sent"]; ok {
		metrics.Metrics["bytes_sent"] = val
	}
	if val, ok := statusMap["Bytes_received"]; ok {
		metrics.Metrics["bytes_received"] = val
	}

	// Get connection stats
	stats := m.db.Stats()
	metrics.Metrics["pool_open_connections"] = stats.OpenConnections
	metrics.Metrics["pool_in_use"] = stats.InUse
	metrics.Metrics["pool_idle"] = stats.Idle
	metrics.Metrics["pool_wait_count"] = stats.WaitCount
	metrics.Metrics["pool_wait_duration_ms"] = stats.WaitDuration.Milliseconds()

	return metrics, nil
}

// GetInfo returns MySQL service information
func (m *MySQLManager) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	info := make(map[string]interface{})

	// Get version
	var version string
	if err := m.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err == nil {
		info["version"] = version
	}

	// Get databases
	rows, err := m.db.QueryContext(ctx, "SHOW DATABASES")
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

	// Get variables
	var maxConnections string
	if err := m.db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'max_connections'").Scan(new(string), &maxConnections); err == nil {
		info["max_connections"] = maxConnections
	}

	return info, nil
}

// ValidateConfig validates MySQL configuration
func (m *MySQLManager) ValidateConfig(config map[string]interface{}) error {
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
func (m *MySQLManager) Type() string {
	return "mysql"
}
