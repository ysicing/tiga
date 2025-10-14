package dbdriver

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SQLDriverBase provides common functionality for SQL database drivers (MySQL, PostgreSQL).
// This eliminates code duplication between mysql.go and postgres.go.
type SQLDriverBase struct {
	db     *sql.DB
	config ConnectionConfig
}

// setConnectionPool configures connection pool parameters with defaults.
func (b *SQLDriverBase) setConnectionPool(db *sql.DB, cfg ConnectionConfig) {
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 50
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime == 0 {
		lifetime = 5 * time.Minute
	}
	idleTimeout := cfg.ConnMaxIdleTime
	if idleTimeout == 0 {
		idleTimeout = 2 * time.Minute
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)
	db.SetConnMaxIdleTime(idleTimeout)
}

const (
	// DefaultMaxRowCount is the default maximum number of rows to scan from a query result.
	// This prevents memory exhaustion from very large result sets.
	DefaultMaxRowCount = 10000
)

// scanQueryResults scans all rows from a SQL query result set with a maximum row limit.
// This is the common logic shared by MySQL and PostgreSQL ExecuteQuery implementations.
// Returns ErrRowLimitExceeded if the result set exceeds DefaultMaxRowCount rows.
func scanQueryResults(rows *sql.Rows) (columns []string, results []map[string]interface{}, err error) {
	columns, err = rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get columns: %w", err)
	}

	columnTypes, _ := rows.ColumnTypes()
	results = make([]map[string]interface{}, 0)

	rowCount := 0
	for rows.Next() {
		rowCount++
		if rowCount > DefaultMaxRowCount {
			return nil, nil, fmt.Errorf("%w (scanned %d rows)", ErrRowLimitExceeded, rowCount)
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			row[col] = convertSQLValue(values[i], columnTypes[i])
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iteration error: %w", err)
	}

	return columns, results, nil
}

// convertSQLValue converts database values to JSON-friendly types.
// Handles NULL, byte arrays, timestamps, and numeric types.
func convertSQLValue(value interface{}, columnType *sql.ColumnType) interface{} {
	switch v := value.(type) {
	case nil:
		return nil
	case []byte:
		// For numeric types, convert to string to avoid precision loss in JSON
		if columnType != nil {
			switch strings.ToUpper(columnType.DatabaseTypeName()) {
			case "INT", "INTEGER", "SMALLINT", "BIGINT", "DECIMAL", "NUMERIC", "FLOAT", "DOUBLE":
				return string(v)
			}
		}
		return string(v)
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano)
	default:
		return v
	}
}

// containsLimitClause checks if a SQL query already has a LIMIT clause.
func containsLimitClause(query string) bool {
	upper := strings.ToUpper(query)
	return strings.Contains(upper, " LIMIT ")
}

// applyQueryLimit adds a LIMIT clause to a SELECT query if needed.
func applyQueryLimit(query string, limit int) string {
	if limit > 0 && !containsLimitClause(query) {
		return fmt.Sprintf("%s LIMIT %d", query, limit)
	}
	return query
}
