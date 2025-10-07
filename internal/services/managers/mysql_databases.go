package managers

import (
	"context"
	"database/sql"
	"fmt"
)

// ListDatabases lists all databases
func (m *MySQLManager) ListDatabases(ctx context.Context) ([]string, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	rows, err := m.db.QueryContext(ctx, "SHOW DATABASES")
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

// CreateDatabase creates a new database
func (m *MySQLManager) CreateDatabase(ctx context.Context, dbName string, charset string, collation string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	// Set defaults if not provided
	if charset == "" {
		charset = "utf8mb4"
	}
	if collation == "" {
		collation = "utf8mb4_unicode_ci"
	}

	query := fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s", dbName, charset, collation)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	return nil
}

// DropDatabase drops a database
func (m *MySQLManager) DropDatabase(ctx context.Context, dbName string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	query := fmt.Sprintf("DROP DATABASE `%s`", dbName)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	return nil
}

// ListUsers lists all MySQL users
func (m *MySQLManager) ListUsers(ctx context.Context) ([]map[string]interface{}, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	query := "SELECT User, Host FROM mysql.user ORDER BY User, Host"
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var username, host string
		if err := rows.Scan(&username, &host); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, map[string]interface{}{
			"username": username,
			"host":     host,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// CreateUser creates a new MySQL user
func (m *MySQLManager) CreateUser(ctx context.Context, username, password, host string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	if host == "" {
		host = "%"
	}

	query := fmt.Sprintf("CREATE USER '%s'@'%s' IDENTIFIED BY '%s'", username, host, password)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// DropUser drops a MySQL user
func (m *MySQLManager) DropUser(ctx context.Context, username, host string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	if host == "" {
		host = "%"
	}

	query := fmt.Sprintf("DROP USER '%s'@'%s'", username, host)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to drop user: %w", err)
	}

	return nil
}

// GrantPrivileges grants privileges to a user
func (m *MySQLManager) GrantPrivileges(ctx context.Context, username, host, database string, privileges []string) error {
	if m.db == nil {
		return ErrNotConnected
	}

	if host == "" {
		host = "%"
	}

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

	query := fmt.Sprintf("GRANT %s ON `%s`.* TO '%s'@'%s'", privStr, database, username, host)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to grant privileges: %w", err)
	}

	// Flush privileges
	if _, err := m.db.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		return fmt.Errorf("failed to flush privileges: %w", err)
	}

	return nil
}

// QueryResult represents query execution result
type QueryResult struct {
	Columns       []string                 `json:"columns"`
	Rows          []map[string]interface{} `json:"rows"`
	AffectedRows  int64                    `json:"affected_rows"`
	RowCount      int                      `json:"row_count"`
	ExecutionTime int64                    `json:"execution_time"` // in milliseconds
}

// ExecuteQuery executes a SQL query
func (m *MySQLManager) ExecuteQuery(ctx context.Context, database, query string, limit int) (*QueryResult, error) {
	if m.db == nil {
		return nil, ErrNotConnected
	}

	// Switch database if specified
	if database != "" {
		if _, err := m.db.ExecContext(ctx, fmt.Sprintf("USE `%s`", database)); err != nil {
			return nil, fmt.Errorf("failed to switch database: %w", err)
		}
	}

	// Check if it's a SELECT query
	isSelect := len(query) >= 6 && query[0:6] == "SELECT" || query[0:6] == "select"

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

// GetDatabaseSize returns the size of a database in bytes
func (m *MySQLManager) GetDatabaseSize(ctx context.Context, dbName string) (int64, error) {
	if m.db == nil {
		return 0, ErrNotConnected
	}

	query := `
		SELECT SUM(data_length + index_length) as size
		FROM information_schema.TABLES
		WHERE table_schema = ?
	`

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
