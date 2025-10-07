package managers

import (
	"context"
	"database/sql"
	"fmt"
)

// ListDatabases lists all PostgreSQL databases
func (m *PostgreSQLManager) ListDatabases(ctx context.Context) ([]string, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	query := "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname"
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, fmt.Errorf("failed to scan database name: %w", err)
		}
		databases = append(databases, dbName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating databases: %w", err)
	}

	return databases, nil
}

// CreateDatabase creates a new PostgreSQL database
func (m *PostgreSQLManager) CreateDatabase(ctx context.Context, dbName string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	query := fmt.Sprintf("CREATE DATABASE \"%s\"", dbName)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}

// DropDatabase drops a PostgreSQL database
func (m *PostgreSQLManager) DropDatabase(ctx context.Context, dbName string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	query := fmt.Sprintf("DROP DATABASE \"%s\"", dbName)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	return nil
}

// ListUsers lists all PostgreSQL roles/users
func (m *PostgreSQLManager) ListUsers(ctx context.Context) ([]map[string]interface{}, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	query := `
		SELECT rolname, rolsuper, rolcreaterole, rolcreatedb, rolcanlogin
		FROM pg_roles
		WHERE rolcanlogin = true
		ORDER BY rolname
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var rolname string
		var rolsuper, rolcreaterole, rolcreatedb, rolcanlogin bool
		if err := rows.Scan(&rolname, &rolsuper, &rolcreaterole, &rolcreatedb, &rolcanlogin); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, map[string]interface{}{
			"username":     rolname,
			"is_superuser": rolsuper,
			"create_role":  rolcreaterole,
			"create_db":    rolcreatedb,
			"can_login":    rolcanlogin,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// CreateUser creates a new PostgreSQL user (role with login)
func (m *PostgreSQLManager) CreateUser(ctx context.Context, username, password string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	query := fmt.Sprintf("CREATE USER \"%s\" WITH PASSWORD '%s'", username, password)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// DropUser drops a PostgreSQL user (role)
func (m *PostgreSQLManager) DropUser(ctx context.Context, username string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	query := fmt.Sprintf("DROP USER \"%s\"", username)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to drop user: %w", err)
	}

	return nil
}

// GrantPrivileges grants privileges to a PostgreSQL user
func (m *PostgreSQLManager) GrantPrivileges(ctx context.Context, username, database string, privileges []string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	// Default to ALL PRIVILEGES if none specified
	privStr := "ALL PRIVILEGES"
	if len(privileges) > 0 {
		privStr = ""
		for i, priv := range privileges {
			if i > 0 {
				privStr += ", "
			}
			privStr += priv
		}
	}

	query := fmt.Sprintf("GRANT %s ON DATABASE \"%s\" TO \"%s\"", privStr, database, username)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to grant privileges: %w", err)
	}

	return nil
}

// ExecuteQuery executes a SQL query on PostgreSQL
func (m *PostgreSQLManager) ExecuteQuery(ctx context.Context, database, query string, limit int) (*QueryResult, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	// For PostgreSQL, we need to connect to the target database
	// Since we can't change database in an existing connection, we'll just execute in current connection
	// The client should connect to the right database initially

	// Check if it's a SELECT query
	isSelect := len(query) >= 6 && (query[0:6] == "SELECT" || query[0:6] == "select")

	if isSelect {
		// Add LIMIT if not present
		if limit > 0 {
			query = fmt.Sprintf("%s LIMIT %d", query, limit)
		}

		rows, err := m.db.QueryContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("failed to get columns: %w", err)
		}

		var results []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				// Convert []byte to string for better JSON serialization
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
			results = append(results, row)
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating rows: %w", err)
		}

		return &QueryResult{
			Columns:       columns,
			Rows:          results,
			RowCount:      len(results),
			AffectedRows:  0,
			ExecutionTime: 0,
		}, nil
	}

	// For non-SELECT queries
	result, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	affected, _ := result.RowsAffected()

	return &QueryResult{
		Columns:       []string{},
		Rows:          []map[string]interface{}{},
		RowCount:      0,
		AffectedRows:  affected,
		ExecutionTime: 0,
	}, nil
}

// GetDatabaseSize returns the size of a PostgreSQL database in bytes
func (m *PostgreSQLManager) GetDatabaseSize(ctx context.Context, dbName string) (int64, error) {
	if m.db == nil {
		return 0, ErrNotConnected
	}

	query := "SELECT pg_database_size($1)"

	var size sql.NullInt64
	err := m.db.QueryRowContext(ctx, query, dbName).Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("failed to get database size: %w", err)
	}

	if !size.Valid {
		return 0, nil
	}

	return size.Int64, nil
}

// ListSchemas lists all schemas in the current database
func (m *PostgreSQLManager) ListSchemas(ctx context.Context) ([]string, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schema_name
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return nil, fmt.Errorf("failed to scan schema name: %w", err)
		}
		schemas = append(schemas, schemaName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schemas: %w", err)
	}

	return schemas, nil
}
