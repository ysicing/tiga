package dbdriver

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisDriver implements DatabaseDriver for Redis instances.
type RedisDriver struct {
	client *redis.Client
	config ConnectionConfig
}

// NewRedisDriver constructs a Redis driver.
func NewRedisDriver() *RedisDriver {
	return &RedisDriver{}
}

// Connect establishes a Redis connection using go-redis.
func (d *RedisDriver) Connect(ctx context.Context, cfg ConnectionConfig) error {
	if cfg.Host == "" || cfg.Port == 0 {
		return fmt.Errorf("invalid redis connection config: host and port required")
	}

	dbIndex := 0
	if cfg.Database != "" {
		if parsed, err := strconv.Atoi(cfg.Database); err == nil {
			dbIndex = parsed
		}
	}
	if val, ok := cfg.Params["db"]; ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			dbIndex = parsed
		}
	}

	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           dbIndex,
		PoolSize:     defaultOr(cfg.MaxOpenConns, 50),
		MinIdleConns: defaultOr(cfg.MaxIdleConns, 5),
		DialTimeout:  5 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	if cfg.ConnMaxIdleTime > 0 {
		opts.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	}

	if cfg.SSLMode != "" && !strings.EqualFold(cfg.SSLMode, "disable") {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: strings.EqualFold(cfg.SSLMode, "insecure"),
		}
	}

	d.client = redis.NewClient(opts)
	d.config = cfg

	if err := d.client.Ping(ctx).Err(); err != nil {
		_ = d.client.Close()
		d.client = nil
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	return nil
}

// Disconnect closes the Redis connection.
func (d *RedisDriver) Disconnect(_ context.Context) error {
	if d.client == nil {
		return nil
	}
	err := d.client.Close()
	d.client = nil
	return err
}

// Ping verifies connection health.
func (d *RedisDriver) Ping(ctx context.Context) error {
	if d.client == nil {
		return ErrNotConnected
	}
	return d.client.Ping(ctx).Err()
}

// ListDatabases returns keyspace statistics.
func (d *RedisDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	if d.client == nil {
		return nil, ErrNotConnected
	}

	info, err := d.client.Info(ctx, "keyspace").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to query redis keyspace info: %w", err)
	}

	return parseRedisKeyspace(info), nil
}

// CreateDatabase is not supported in Redis.
func (d *RedisDriver) CreateDatabase(context.Context, CreateDatabaseOptions) error {
	return ErrUnsupportedOperation
}

// DeleteDatabase is not supported in Redis.
func (d *RedisDriver) DeleteDatabase(context.Context, string, map[string]interface{}) error {
	return ErrUnsupportedOperation
}

// ListUsers enumerates ACL users.
func (d *RedisDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	if d.client == nil {
		return nil, ErrNotConnected
	}

	raw, err := d.client.Do(ctx, "ACL", "USERS").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list redis users: %w", err)
	}

	result := make([]UserInfo, 0)
	switch vals := raw.(type) {
	case []interface{}:
		for _, item := range vals {
			if name, ok := item.(string); ok {
				result = append(result, UserInfo{Username: name})
			}
		}
	case []string:
		for _, name := range vals {
			result = append(result, UserInfo{Username: name})
		}
	case string:
		result = append(result, UserInfo{Username: vals})
	}
	return result, nil
}

// CreateUser configures an ACL user with role-mapped permissions.
func (d *RedisDriver) CreateUser(ctx context.Context, opts CreateUserOptions) error {
	if d.client == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(opts.Username) == "" {
		return fmt.Errorf("username is required for redis user creation")
	}
	if opts.Password == "" {
		return fmt.Errorf("password is required for redis user creation")
	}

	args := []interface{}{"SETUSER", opts.Username, "on", ">" + opts.Password, "~*"}
	roleRules := redisACLRules(opts.Roles)
	args = append(args, roleRules...)

	if err := d.client.Do(ctx, append([]interface{}{"ACL"}, args...)...).Err(); err != nil {
		return fmt.Errorf("failed to create redis acl user: %w", err)
	}

	return nil
}

// DeleteUser removes an ACL user.
func (d *RedisDriver) DeleteUser(ctx context.Context, username string, _ map[string]interface{}) error {
	if d.client == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username is required")
	}

	if err := d.client.ACLDelUser(ctx, username).Err(); err != nil {
		return fmt.Errorf("failed to delete redis user: %w", err)
	}
	return nil
}

// UpdateUserPassword updates the password (and optionally roles) for an ACL user.
func (d *RedisDriver) UpdateUserPassword(ctx context.Context, username, password string, opts map[string]interface{}) error {
	if d.client == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	roles := extractRoles(opts)
	args := []interface{}{"SETUSER", username, "on", ">" + password, "~*"}
	args = append(args, redisACLRules(roles)...)

	if err := d.client.Do(ctx, append([]interface{}{"ACL"}, args...)...).Err(); err != nil {
		return fmt.Errorf("failed to update redis acl user: %w", err)
	}
	return nil
}

// ExecuteQuery runs a Redis command.
func (d *RedisDriver) ExecuteQuery(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if d.client == nil {
		return nil, ErrNotConnected
	}

	commandParts := make([]interface{}, 0, len(req.Args)+1)
	if len(req.Args) > 0 {
		commandParts = append(commandParts, req.Args...)
	} else {
		parts := strings.Fields(req.Query)
		if len(parts) == 0 {
			return nil, fmt.Errorf("command cannot be empty")
		}
		for _, part := range parts {
			commandParts = append(commandParts, part)
		}
	}

	start := time.Now()
	result, err := d.client.Do(ctx, commandParts...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to execute redis command: %w", err)
	}

	rows := redisResultToRows(result)

	return &QueryResult{
		Columns:       []string{"result"},
		Rows:          rows,
		RowCount:      len(rows),
		ExecutionTime: time.Since(start),
		RawResponse:   result,
	}, nil
}

// GetVersion fetches Redis server version.
func (d *RedisDriver) GetVersion(ctx context.Context) (string, error) {
	if d.client == nil {
		return "", ErrNotConnected
	}

	info, err := d.client.Info(ctx, "server").Result()
	if err != nil {
		return "", fmt.Errorf("failed to query redis server info: %w", err)
	}

	return parseRedisInfoValue(info, "redis_version"), nil
}

// GetUptime returns uptime duration.
func (d *RedisDriver) GetUptime(ctx context.Context) (time.Duration, error) {
	if d.client == nil {
		return 0, ErrNotConnected
	}

	info, err := d.client.Info(ctx, "server").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to query redis uptime: %w", err)
	}

	uptimeStr := parseRedisInfoValue(info, "uptime_in_seconds")
	seconds, err := strconv.ParseInt(uptimeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid uptime_in_seconds value: %w", err)
	}
	return time.Duration(seconds) * time.Second, nil
}

func parseRedisKeyspace(info string) []DatabaseInfo {
	lines := strings.Split(info, "\n")
	results := make([]DatabaseInfo, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "db") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		stats := parts[1]
		infoMap := make(map[string]string)
		for _, kv := range strings.Split(stats, ",") {
			pair := strings.SplitN(kv, "=", 2)
			if len(pair) == 2 {
				infoMap[pair[0]] = pair[1]
			}
		}
		keys, _ := strconv.Atoi(infoMap["keys"])
		expires, _ := strconv.Atoi(infoMap["expires"])

		results = append(results, DatabaseInfo{
			Name:     name,
			KeyCount: keys,
			Extra: map[string]interface{}{
				"expires":      expires,
				"avg_ttl":      infoMap["avg_ttl"],
				"volatileKeys": infoMap["avg_ttl"] != "0",
			},
		})
	}
	return results
}

func parseRedisInfoValue(info, key string) string {
	prefix := key + ":"
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func redisResultToRows(result interface{}) []map[string]interface{} {
	switch v := result.(type) {
	case []interface{}:
		rows := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			rows = append(rows, map[string]interface{}{"result": item})
		}
		return rows
	case []string:
		rows := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			rows = append(rows, map[string]interface{}{"result": item})
		}
		return rows
	case string:
		return []map[string]interface{}{{"result": v}}
	case int64, int, float64, bool:
		return []map[string]interface{}{{"result": v}}
	default:
		return []map[string]interface{}{{"result": v}}
	}
}

func redisACLRules(roles []string) []interface{} {
	if len(roles) == 0 {
		return []interface{}{}
	}
	rules := []interface{}{}
	hasRole := false
	for _, role := range roles {
		switch strings.ToLower(role) {
		case "readonly", "read":
			rules = append(rules, "+@read", "-@write", "-@dangerous")
			hasRole = true
		case "readwrite", "write", "manage":
			rules = append(rules, "+@read", "+@write", "-@dangerous", "-flushdb", "-flushall")
			hasRole = true
		case "none":
			rules = append(rules, "-@all")
			hasRole = true
		}
	}
	if !hasRole {
		rules = append(rules, "-@all")
	}
	return rules
}

func defaultOr(actual, fallback int) int {
	if actual > 0 {
		return actual
	}
	return fallback
}

func extractRoles(opts map[string]interface{}) []string {
	if opts == nil {
		return nil
	}

	raw, ok := opts["roles"]
	if !ok {
		return nil
	}

	switch val := raw.(type) {
	case []string:
		return val
	case []interface{}:
		roles := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				roles = append(roles, s)
			}
		}
		return roles
	default:
		return nil
	}
}
