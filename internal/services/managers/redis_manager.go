package managers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ysicing/tiga/internal/models"
)

// RedisManager manages Redis instances
type RedisManager struct {
	*BaseManager
	client *redis.Client
}

// NewRedisManager creates a new Redis manager
func NewRedisManager() *RedisManager {
	return &RedisManager{
		BaseManager: NewBaseManager(),
	}
}

// Initialize initializes the Redis manager
func (m *RedisManager) Initialize(ctx context.Context, instance *models.Instance) error {
	if err := m.BaseManager.Initialize(ctx, instance); err != nil {
		return err
	}
	return nil
}

// Connect establishes connection to Redis
func (m *RedisManager) Connect(ctx context.Context) error {
	connection := m.instance.Connection
	host, _ := connection["host"].(string)
	port, _ := connection["port"].(float64)

	addr := fmt.Sprintf("%s:%d", host, int(port))

	password := m.GetConfigValue("password", "").(string)
	db := 0
	if dbVal := m.GetConfigValue("database", 0); dbVal != nil {
		switch v := dbVal.(type) {
		case int:
			db = v
		case float64:
			db = int(v)
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	m.client = client
	return nil
}

// Disconnect closes connection to Redis
func (m *RedisManager) Disconnect(ctx context.Context) error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// HealthCheck checks Redis health
func (m *RedisManager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	start := time.Now()
	status := &HealthStatus{
		Healthy:       false,
		LastCheckTime: time.Now().Format(time.RFC3339),
		Details:       make(map[string]interface{}),
	}

	// Ping Redis
	if err := m.client.Ping(ctx).Err(); err != nil {
		status.Message = fmt.Sprintf("Health check failed: %v", err)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status, nil
	}

	// Get Redis info
	info, err := m.client.Info(ctx, "server").Result()
	if err == nil {
		status.Details["server_info"] = info
	}

	// Get connected clients
	info, err = m.client.Info(ctx, "clients").Result()
	if err == nil {
		status.Details["clients_info"] = info
	}

	status.Healthy = true
	status.Message = "Redis is healthy"
	status.ResponseTime = time.Since(start).Milliseconds()

	return status, nil
}

// CollectMetrics collects metrics from Redis
func (m *RedisManager) CollectMetrics(ctx context.Context) (*ServiceMetrics, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	metrics := &ServiceMetrics{
		InstanceID: m.instance.ID,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metrics:    make(map[string]interface{}),
	}

	// Get server info
	info, err := m.client.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricsCollectionFailed, err)
	}

	// Parse info string and extract key metrics
	infoMap := parseRedisInfo(info)

	// Server metrics
	if val, ok := infoMap["redis_version"]; ok {
		metrics.Metrics["redis_version"] = val
	}
	if val, ok := infoMap["uptime_in_seconds"]; ok {
		metrics.Metrics["uptime_seconds"] = val
	}

	// Memory metrics
	if val, ok := infoMap["used_memory"]; ok {
		metrics.Metrics["used_memory_bytes"] = val
	}
	if val, ok := infoMap["used_memory_rss"]; ok {
		metrics.Metrics["used_memory_rss_bytes"] = val
	}
	if val, ok := infoMap["used_memory_peak"]; ok {
		metrics.Metrics["used_memory_peak_bytes"] = val
	}
	if val, ok := infoMap["mem_fragmentation_ratio"]; ok {
		metrics.Metrics["mem_fragmentation_ratio"] = val
	}

	// Client metrics
	if val, ok := infoMap["connected_clients"]; ok {
		metrics.Metrics["connected_clients"] = val
	}
	if val, ok := infoMap["blocked_clients"]; ok {
		metrics.Metrics["blocked_clients"] = val
	}

	// Stats metrics
	if val, ok := infoMap["total_commands_processed"]; ok {
		metrics.Metrics["total_commands_processed"] = val
	}
	if val, ok := infoMap["total_connections_received"]; ok {
		metrics.Metrics["total_connections_received"] = val
	}
	if val, ok := infoMap["keyspace_hits"]; ok {
		metrics.Metrics["keyspace_hits"] = val
	}
	if val, ok := infoMap["keyspace_misses"]; ok {
		metrics.Metrics["keyspace_misses"] = val
	}

	// Calculate hit ratio
	if hits, hok := infoMap["keyspace_hits"]; hok {
		if misses, mok := infoMap["keyspace_misses"]; mok {
			h, _ := strconv.ParseFloat(hits, 64)
			m, _ := strconv.ParseFloat(misses, 64)
			if h+m > 0 {
				metrics.Metrics["hit_ratio"] = h / (h + m)
			}
		}
	}

	// Database keys
	dbSize, err := m.client.DBSize(ctx).Result()
	if err == nil {
		metrics.Metrics["total_keys"] = dbSize
	}

	// Pool stats
	stats := m.client.PoolStats()
	metrics.Metrics["pool_hits"] = stats.Hits
	metrics.Metrics["pool_misses"] = stats.Misses
	metrics.Metrics["pool_timeouts"] = stats.Timeouts
	metrics.Metrics["pool_total_conns"] = stats.TotalConns
	metrics.Metrics["pool_idle_conns"] = stats.IdleConns
	metrics.Metrics["pool_stale_conns"] = stats.StaleConns

	return metrics, nil
}

// GetInfo returns Redis service information
func (m *RedisManager) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	info := make(map[string]interface{})

	// Get full info
	infoStr, err := m.client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	infoMap := parseRedisInfo(infoStr)

	// Server info
	if val, ok := infoMap["redis_version"]; ok {
		info["version"] = val
	}
	if val, ok := infoMap["redis_mode"]; ok {
		info["mode"] = val
	}
	if val, ok := infoMap["os"]; ok {
		info["os"] = val
	}

	// Database info
	dbSize, err := m.client.DBSize(ctx).Result()
	if err == nil {
		info["total_keys"] = dbSize
	}

	// Get keyspace info
	keyspace, err := m.client.Info(ctx, "keyspace").Result()
	if err == nil {
		info["keyspace"] = parseRedisInfo(keyspace)
	}

	return info, nil
}

// ValidateConfig validates Redis configuration
func (m *RedisManager) ValidateConfig(config map[string]interface{}) error {
	// Password is optional
	// Database is optional (defaults to 0)
	return nil
}

// Type returns the service type
func (m *RedisManager) Type() string {
	return "redis"
}

// parseRedisInfo parses Redis INFO output into a map
func parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := []rune(info)

	var currentLine []rune
	for _, r := range lines {
		if r == '\n' || r == '\r' {
			if len(currentLine) > 0 {
				line := string(currentLine)
				if len(line) > 0 && line[0] != '#' {
					// Split by ':'
					colonIdx := -1
					for i, c := range line {
						if c == ':' {
							colonIdx = i
							break
						}
					}
					if colonIdx > 0 {
						key := line[:colonIdx]
						value := line[colonIdx+1:]
						result[key] = value
					}
				}
				currentLine = currentLine[:0]
			}
		} else if r != '\r' {
			currentLine = append(currentLine, r)
		}
	}

	return result
}
