package dbdriver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PostgresDriver implements DatabaseDriver for PostgreSQL.
type PostgresDriver struct {
	db     *sql.DB
	config ConnectionConfig
}

// NewPostgresDriver constructs a PostgreSQL driver.
func NewPostgresDriver() *PostgresDriver {
	return &PostgresDriver{}
}

// Connect establishes a PostgreSQL connection.
func (d *PostgresDriver) Connect(ctx context.Context, cfg ConnectionConfig) error {
	if cfg.Host == "" || cfg.Port == 0 {
		return fmt.Errorf("invalid postgres connection config: host and port required")
	}

	username := cfg.Username
	if username == "" {
		username = "postgres"
	}

	sslmode := cfg.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
		cfg.Host,
		cfg.Port,
		username,
		cfg.Password,
		cfg.Database,
		sslmode,
	)

	for key, value := range cfg.Params {
		dsn += fmt.Sprintf(" %s=%s", key, value)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Use shared connection pool configuration
	base := &SQLDriverBase{}
	base.setConnectionPool(db, cfg)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping postgres: %w", err)
	}

	d.db = db
	d.config = cfg
	return nil
}

// Disconnect closes the database connection.
func (d *PostgresDriver) Disconnect(_ context.Context) error {
	if d.db == nil {
		return nil
	}
	err := d.db.Close()
	d.db = nil
	return err
}

// Ping verifies connectivity.
func (d *PostgresDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		return ErrNotConnected
	}
	return d.db.PingContext(ctx)
}

// ListDatabases enumerates databases with metadata.
func (d *PostgresDriver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	if d.db == nil {
		return nil, ErrNotConnected
	}

	const query = `
SELECT
    datname,
    pg_encoding_to_char(encoding) AS encoding,
    datcollate,
    pg_database_size(datname) AS size_bytes
FROM pg_database
WHERE datistemplate = FALSE
ORDER BY datname`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list postgres databases: %w", err)
	}
	defer rows.Close()

	var databases []DatabaseInfo
	for rows.Next() {
		var info DatabaseInfo
		if err := rows.Scan(&info.Name, &info.Charset, &info.Collation, &info.SizeBytes); err != nil {
			return nil, fmt.Errorf("failed to scan postgres database row: %w", err)
		}
		info.TableCount = 0
		databases = append(databases, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iteration error listing postgres databases: %w", err)
	}
	return databases, nil
}

// CreateDatabase creates a new PostgreSQL database.
func (d *PostgresDriver) CreateDatabase(ctx context.Context, opts CreateDatabaseOptions) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(opts.Name) == "" {
		return fmt.Errorf("database name is required")
	}

	builder := strings.Builder{}
	builder.WriteString("CREATE DATABASE ")
	builder.WriteString(quotePGIdentifier(opts.Name))

	withClauses := make([]string, 0, 2)
	if opts.Owner != "" {
		withClauses = append(withClauses, fmt.Sprintf("OWNER %s", quotePGIdentifier(opts.Owner)))
	}
	if opts.Charset != "" {
		withClauses = append(withClauses, fmt.Sprintf("ENCODING '%s'", opts.Charset))
	}
	if opts.Collation != "" {
		withClauses = append(withClauses, fmt.Sprintf("LC_COLLATE '%s'", opts.Collation))
	}

	if len(withClauses) > 0 {
		builder.WriteString(" WITH ")
		builder.WriteString(strings.Join(withClauses, " "))
	}

	if _, err := d.db.ExecContext(ctx, builder.String()); err != nil {
		return fmt.Errorf("failed to create postgres database: %w", err)
	}
	return nil
}

// DeleteDatabase drops a PostgreSQL database.
func (d *PostgresDriver) DeleteDatabase(ctx context.Context, name string, opts map[string]interface{}) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("database name is required")
	}

	if confirm, ok := opts["confirm_name"].(string); ok && confirm != name {
		return fmt.Errorf("confirmation name mismatch")
	}

	stmt := fmt.Sprintf("DROP DATABASE %s", quotePGIdentifier(name))
	if _, err := d.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop postgres database: %w", err)
	}
	return nil
}

// ListUsers returns roles with LOGIN privilege.
func (d *PostgresDriver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	if d.db == nil {
		return nil, ErrNotConnected
	}

	const query = `
SELECT rolname, rolsuper, rolcreaterole, rolcreatedb
FROM pg_roles
WHERE rolcanlogin = TRUE
ORDER BY rolname`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list postgres roles: %w", err)
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var (
			name          string
			isSuper       bool
			canCreateRole bool
			canCreateDB   bool
		)
		if err := rows.Scan(&name, &isSuper, &canCreateRole, &canCreateDB); err != nil {
			return nil, fmt.Errorf("failed to scan postgres role row: %w", err)
		}

		info := UserInfo{
			Username:    name,
			Permissions: []string{},
			Extra: map[string]interface{}{
				"is_superuser":    isSuper,
				"can_create_db":   canCreateDB,
				"can_create_role": canCreateRole,
			},
		}
		users = append(users, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iteration error listing postgres users: %w", err)
	}
	return users, nil
}

// CreateUser creates a PostgreSQL role with login.
func (d *PostgresDriver) CreateUser(ctx context.Context, opts CreateUserOptions) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(opts.Username) == "" {
		return fmt.Errorf("username is required for postgres role creation")
	}
	if opts.Password == "" {
		return fmt.Errorf("password is required for postgres role creation")
	}

	stmt := fmt.Sprintf("CREATE ROLE %s WITH LOGIN PASSWORD $1", quotePGIdentifier(opts.Username))
	if _, err := d.db.ExecContext(ctx, stmt, opts.Password); err != nil {
		return fmt.Errorf("failed to create postgres role: %w", err)
	}

	for _, role := range opts.Roles {
		switch strings.ToLower(role) {
		case "readonly", "read":
			// Grant read-only access to all schemas
			if _, err := d.db.ExecContext(ctx,
				fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA public TO %s", quotePGIdentifier(opts.Username))); err != nil {
				return fmt.Errorf("failed to grant readonly privileges: %w", err)
			}
		case "readwrite", "write", "manage":
			if _, err := d.db.ExecContext(ctx,
				fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %s", quotePGIdentifier(opts.Username))); err != nil {
				return fmt.Errorf("failed to grant readwrite privileges: %w", err)
			}
		}
	}

	return nil
}

// DeleteUser removes a role.
func (d *PostgresDriver) DeleteUser(ctx context.Context, username string, _ map[string]interface{}) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username is required")
	}

	stmt := fmt.Sprintf("DROP ROLE %s", quotePGIdentifier(username))
	if _, err := d.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("failed to drop postgres role: %w", err)
	}
	return nil
}

// UpdateUserPassword updates role password.
func (d *PostgresDriver) UpdateUserPassword(ctx context.Context, username, password string, _ map[string]interface{}) error {
	if d.db == nil {
		return ErrNotConnected
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	stmt := fmt.Sprintf("ALTER ROLE %s WITH PASSWORD $1", quotePGIdentifier(username))
	if _, err := d.db.ExecContext(ctx, stmt, password); err != nil {
		return fmt.Errorf("failed to update postgres password: %w", err)
	}
	return nil
}

// ExecuteQuery executes an arbitrary SQL statement.
func (d *PostgresDriver) ExecuteQuery(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if d.db == nil {
		return nil, ErrNotConnected
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if req.Database != "" && !strings.EqualFold(req.Database, d.config.Database) {
		if _, err := d.db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", quotePGIdentifier(req.Database))); err != nil {
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
			return nil, fmt.Errorf("failed to execute postgres query: %w", err)
		}
		defer rows.Close()

		// Use shared result scanning logic
		columns, results, err := scanQueryResults(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres query error: %w", err)
		}

		return &QueryResult{
			Columns:       columns,
			Rows:          results,
			RowCount:      len(results),
			ExecutionTime: time.Since(start),
		}, nil
	}

	result, err := d.db.ExecContext(ctx, query, req.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute postgres statement: %w", err)
	}
	affected, _ := result.RowsAffected()

	return &QueryResult{
		AffectedRows:  affected,
		ExecutionTime: time.Since(start),
	}, nil
}

// GetVersion returns the server version string.
func (d *PostgresDriver) GetVersion(ctx context.Context) (string, error) {
	if d.db == nil {
		return "", ErrNotConnected
	}

	var version string
	if err := d.db.QueryRowContext(ctx, "SHOW server_version").Scan(&version); err != nil {
		return "", fmt.Errorf("failed to query postgres version: %w", err)
	}
	return version, nil
}

// GetUptime returns the duration since postmaster start.
func (d *PostgresDriver) GetUptime(ctx context.Context) (time.Duration, error) {
	if d.db == nil {
		return 0, ErrNotConnected
	}

	var seconds float64
	if err := d.db.QueryRowContext(ctx, "SELECT EXTRACT(EPOCH FROM now() - pg_postmaster_start_time())").Scan(&seconds); err != nil {
		return 0, fmt.Errorf("failed to query postgres uptime: %w", err)
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func quotePGIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
