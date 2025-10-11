package dbdriver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLDriver implements DatabaseDriver for MySQL-compatible databases.
type MySQLDriver struct {
	db     *sql.DB
	config ConnectionConfig
}

// NewMySQLDriver constructs a new MySQL driver.
func NewMySQLDriver() *MySQLDriver {
	return &MySQLDriver{}
}

// Connect establishes a connection using the provided configuration.
func (d *MySQLDriver) Connect(ctx context.Context, cfg ConnectionConfig) error {
	if cfg.Host == "" || cfg.Port == 0 {
		return fmt.Errorf("invalid mysql connection config: host and port required")
	}

	username := cfg.Username
	if username == "" {
		username = "root"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=false&charset=utf8mb4",
		username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		sanitizeDatabaseName(cfg.Database),
	)

	if cfg.Params != nil {
		for key, val := range cfg.Params {
			dsn += fmt.Sprintf("&%s=%s", key, val)
		}
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// Use shared connection pool configuration
	base := &SQLDriverBase{}
	base.setConnectionPool(db, cfg)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping mysql: %w", err)
	}

	d.db = db
	d.config = cfg
	return nil
}

// Disconnect closes the underlying database connection.
func (d *MySQLDriver) Disconnect(_ context.Context) error {
	if d.db == nil {
		return nil
	}
	err := d.db.Close()
	d.db = nil
	return err
}

// Ping checks the connection health.
func (d *MySQLDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		return ErrNotConnected
	}
	return d.db.PingContext(ctx)
}

// ListDatabases returns metadata for all accessible databases.
func (d *MySQLDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	if d.db == nil {
		return nil, ErrNotConnected
	}

	const query = `
SELECT
    s.schema_name,
    s.default_character_set_name,
    s.default_collation_name,
    IFNULL(SUM(t.data_length + t.index_length), 0) AS size_bytes,
    COUNT(t.table_name) AS table_count
FROM information_schema.schemata s
LEFT JOIN information_schema.tables t
    ON t.table_schema = s.schema_name
GROUP BY s.schema_name, s.default_character_set_name, s.default_collation_name
ORDER BY s.schema_name`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list mysql databases: %w", err)
	}
	defer rows.Close()

	var databases []DatabaseInfo
	for rows.Next() {
		var info DatabaseInfo
		if err := rows.Scan(&info.Name, &info.Charset, &info.Collation, &info.SizeBytes, &info.TableCount); err != nil {
			return nil, fmt.Errorf("failed to scan mysql database row: %w", err)
		}
		databases = append(databases, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mysql databases: %w", err)
	}

	return databases, nil
}

// CreateDatabase creates a new schema with optional charset/collation.
func (d *MySQLDriver) CreateDatabase(ctx context.Context, opts CreateDatabaseOptions) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if opts.Name == "" {
		return fmt.Errorf("database name is required")
	}

	builder := strings.Builder{}
	builder.WriteString("CREATE DATABASE ")
	builder.WriteString(quoteIdentifier(opts.Name))

	if opts.Charset != "" {
		builder.WriteString(" CHARACTER SET ")
		builder.WriteString(opts.Charset)
	}
	if opts.Collation != "" {
		builder.WriteString(" COLLATE ")
		builder.WriteString(opts.Collation)
	}

	if _, err := d.db.ExecContext(ctx, builder.String()); err != nil {
		return fmt.Errorf("failed to create mysql database: %w", err)
	}
	return nil
}

// DeleteDatabase drops a schema (opts may contain confirm_name).
func (d *MySQLDriver) DeleteDatabase(ctx context.Context, name string, opts map[string]interface{}) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if name == "" {
		return fmt.Errorf("database name is required")
	}

	if confirm, ok := opts["confirm_name"].(string); ok && confirm != name {
		return fmt.Errorf("confirmation name mismatch")
	}

	stmt := fmt.Sprintf("DROP DATABASE %s", quoteIdentifier(name))
	if _, err := d.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop mysql database: %w", err)
	}
	return nil
}

// ListUsers returns available database users.
func (d *MySQLDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	if d.db == nil {
		return nil, ErrNotConnected
	}

	const query = `SELECT user, host FROM mysql.user ORDER BY user`
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list mysql users: %w", err)
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var info UserInfo
		if err := rows.Scan(&info.Username, &info.Host); err != nil {
			return nil, fmt.Errorf("failed to scan mysql user row: %w", err)
		}
		users = append(users, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mysql users: %w", err)
	}
	return users, nil
}

// CreateUser creates a new database user and optionally grants privileges.
func (d *MySQLDriver) CreateUser(ctx context.Context, opts CreateUserOptions) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(opts.Username) == "" {
		return fmt.Errorf("username is required for mysql user creation")
	}
	if opts.Password == "" {
		return fmt.Errorf("password is required for mysql user creation")
	}

	host := opts.Host
	if host == "" {
		host = "%"
	}

	createStmt := fmt.Sprintf("CREATE USER '%s'@'%s' IDENTIFIED BY ?", escapeSingleQuotes(opts.Username), escapeSingleQuotes(host))
	if _, err := d.db.ExecContext(ctx, createStmt, opts.Password); err != nil {
		return fmt.Errorf("failed to create mysql user: %w", err)
	}

	if len(opts.Roles) > 0 {
		if grantStmt, ok := buildMySQLGrantStatement(opts.Username, host, opts.Roles); ok {
			if _, err := d.db.ExecContext(ctx, grantStmt); err != nil {
				return fmt.Errorf("failed to grant mysql privileges: %w", err)
			}
		}
	}

	if _, err := d.db.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		return fmt.Errorf("failed to flush mysql privileges: %w", err)
	}
	return nil
}

// DeleteUser removes a database user.
func (d *MySQLDriver) DeleteUser(ctx context.Context, username string, opts map[string]interface{}) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username is required")
	}

	host := "%"
	if v, ok := opts["host"].(string); ok && v != "" {
		host = v
	}

	stmt := fmt.Sprintf("DROP USER '%s'@'%s'", escapeSingleQuotes(username), escapeSingleQuotes(host))
	if _, err := d.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop mysql user: %w", err)
	}
	return nil
}

// UpdateUserPassword updates credentials for an existing user.
func (d *MySQLDriver) UpdateUserPassword(ctx context.Context, username, password string, opts map[string]interface{}) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	host := "%"
	if v, ok := opts["host"].(string); ok && v != "" {
		host = v
	}

	stmt := fmt.Sprintf("ALTER USER '%s'@'%s' IDENTIFIED BY ?", escapeSingleQuotes(username), escapeSingleQuotes(host))
	if _, err := d.db.ExecContext(ctx, stmt, password); err != nil {
		return fmt.Errorf("failed to update mysql user password: %w", err)
	}

	if _, err := d.db.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		return fmt.Errorf("failed to flush mysql privileges: %w", err)
	}

	return nil
}

// ExecuteQuery runs an arbitrary SQL query and returns structured results.
func (d *MySQLDriver) ExecuteQuery(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if d.db == nil {
		return nil, ErrNotConnected
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Switch database explicitly if requested.
	if req.Database != "" && !strings.EqualFold(req.Database, d.config.Database) {
		if _, err := d.db.ExecContext(ctx, fmt.Sprintf("USE %s", quoteIdentifier(req.Database))); err != nil {
			return nil, fmt.Errorf("failed to switch database: %w", err)
		}
	}

	isSelect := strings.HasPrefix(strings.ToUpper(query), "SELECT")

	start := time.Now()

	if isSelect {
		// Apply query limit using shared helper
		query = applyQueryLimit(query, req.Limit)

		rows, err := d.db.QueryContext(ctx, query, req.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute mysql query: %w", err)
		}
		defer rows.Close()

		// Use shared result scanning logic
		columns, resultRows, err := scanQueryResults(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql query error: %w", err)
		}

		return &QueryResult{
			Columns:       columns,
			Rows:          resultRows,
			RowCount:      len(resultRows),
			ExecutionTime: time.Since(start),
		}, nil
	}

	result, err := d.db.ExecContext(ctx, query, req.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute mysql statement: %w", err)
	}
	affected, _ := result.RowsAffected()

	return &QueryResult{
		AffectedRows:  affected,
		ExecutionTime: time.Since(start),
	}, nil
}

// GetVersion returns the MySQL server version string.
func (d *MySQLDriver) GetVersion(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", ErrNotConnected
	}

	var version string
	if err := d.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return "", fmt.Errorf("failed to query mysql version: %w", err)
	}
	return version, nil
}

// GetUptime returns the server uptime.
func (d *MySQLDriver) GetUptime(ctx context.Context) (time.Duration, error) {
	if d.db == nil {
		return 0, ErrNotConnected
	}

	var uptimeSeconds int64
	if err := d.db.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Uptime'").Scan(new(string), &uptimeSeconds); err != nil {
		return 0, fmt.Errorf("failed to query mysql uptime: %w", err)
	}
	return time.Duration(uptimeSeconds) * time.Second, nil
}

func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func sanitizeDatabaseName(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return name
}

func escapeSingleQuotes(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func buildMySQLGrantStatement(username, host string, roles []string) (string, bool) {
	privileges := make([]string, 0, len(roles))
	for _, role := range roles {
		switch strings.ToLower(role) {
		case "readonly", "read":
			privileges = append(privileges, "SELECT")
		case "readwrite", "write", "manage":
			privileges = append(privileges, "ALL PRIVILEGES")
		default:
			// ignore unknown roles
		}
	}

	if len(privileges) == 0 {
		return "", false
	}

	privClause := strings.Join(privileges, ", ")
	return fmt.Sprintf("GRANT %s ON *.* TO '%s'@'%s'", privClause, escapeSingleQuotes(username), escapeSingleQuotes(host)), true
}
