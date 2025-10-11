package database

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrSQLDangerousOperation indicates the SQL contains a forbidden statement.
	ErrSQLDangerousOperation = errors.New("SQL contains forbidden operation")
	// ErrSQLDangerousFunction indicates the SQL contains a forbidden function call.
	ErrSQLDangerousFunction = errors.New("SQL contains forbidden function")
	// ErrSQLMissingWhere indicates an UPDATE/DELETE statement is missing a WHERE clause.
	ErrSQLMissingWhere = errors.New("SQL statement must include WHERE clause")
	// ErrRedisDangerousCommand indicates a Redis command is forbidden.
	ErrRedisDangerousCommand = errors.New("Redis command is forbidden")
	// ErrSQLMultipleStatements indicates multiple SQL statements detected.
	ErrSQLMultipleStatements = errors.New("multiple SQL statements are not allowed")
)

// Compiled regex patterns for security checks (case-insensitive, word boundary)
var (
	ddlPattern         = regexp.MustCompile(`(?i)\b(DROP|TRUNCATE|ALTER|CREATE)\b`)
	dmlWithoutWhere    = regexp.MustCompile(`(?i)\b(UPDATE|DELETE)\b.*\bFROM\b`)
	whereClausePattern = regexp.MustCompile(`(?i)\bWHERE\b`)
	semicolonPattern   = regexp.MustCompile(`;`)
)

// SecurityFilter validates SQL and Redis commands against the project's safety rules.
type SecurityFilter struct {
	bannedStatements []string
	bannedFunctions  []string
	redisBlacklist   map[string]struct{}
}

// NewSecurityFilter returns a filter initialised with defaults from the specification.
func NewSecurityFilter() *SecurityFilter {
	return &SecurityFilter{
		bannedStatements: []string{
			"DROP ",
			"TRUNCATE ",
			"ALTER ",
			"CREATE DATABASE",
			"CREATE TABLE",
			"CREATE INDEX",
			"RENAME ",
			"GRANT ",
			"REVOKE ",
		},
		bannedFunctions: []string{
			"LOAD_FILE",
			"INTO OUTFILE",
			"DUMPFILE",
			"XP_CMDSHELL",
		},
		redisBlacklist: map[string]struct{}{
			"FLUSHDB":  {},
			"FLUSHALL": {},
			"SHUTDOWN": {},
			"CONFIG":   {},
			"SAVE":     {},
			"BGSAVE":   {},
		},
	}
}

// ValidateSQL ensures a SQL query complies with the security policy.
func (f *SecurityFilter) ValidateSQL(query string) error {
	if strings.TrimSpace(query) == "" {
		return errors.New("SQL query cannot be empty")
	}

	statements := splitStatements(query)
	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if err := f.validateSingleSQL(strings.TrimSpace(stmt)); err != nil {
			return err
		}
	}
	return nil
}

func (f *SecurityFilter) validateSingleSQL(statement string) error {
	// Remove SQL comments to prevent comment-based injection
	statement = removeComments(statement)
	normalized := normalizeWhitespace(statement)
	upper := strings.ToUpper(normalized)

	// Use word boundary regex to prevent bypasses like "dr/**/op"
	for _, banned := range f.bannedStatements {
		pattern := regexp.MustCompile(`(?i)\b` + strings.TrimSpace(banned) + `\b`)
		if pattern.MatchString(upper) {
			return fmt.Errorf("%w: %s", ErrSQLDangerousOperation, strings.TrimSpace(banned))
		}
	}

	firstKeyword := extractFirstKeyword(upper)
	if firstKeyword == "UPDATE" || firstKeyword == "DELETE" {
		if !containsWhereClause(upper) {
			return ErrSQLMissingWhere
		}
	}

	for _, fn := range f.bannedFunctions {
		if strings.Contains(upper, fn+"(") {
			return fmt.Errorf("%w: %s", ErrSQLDangerousFunction, fn)
		}
	}

	return nil
}

// removeComments strips SQL comments to prevent injection bypasses
func removeComments(sql string) string {
	// Remove single-line comments (-- )
	lines := strings.Split(sql, "\n")
	var cleaned []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	result := strings.Join(cleaned, " ")

	// Remove multi-line comments (/* */)
	commentPattern := regexp.MustCompile(`/\*.*?\*/`)
	result = commentPattern.ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

// ValidateRedisCommand ensures the Redis command is not part of the blacklist.
func (f *SecurityFilter) ValidateRedisCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return errors.New("Redis command cannot be empty")
	}

	first := strings.ToUpper(extractFirstKeyword(command))
	if _, banned := f.redisBlacklist[first]; banned {
		return fmt.Errorf("%w: %s", ErrRedisDangerousCommand, first)
	}
	return nil
}

func splitStatements(query string) []string {
	segments := strings.Split(query, ";")
	if len(segments) == 0 {
		return []string{query}
	}
	return segments
}

func normalizeWhitespace(input string) string {
	fields := strings.Fields(input)
	return strings.Join(fields, " ")
}

func extractFirstKeyword(input string) string {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func containsWhereClause(upperStatement string) bool {
	whereIndex := strings.Index(upperStatement, " WHERE ")
	if whereIndex == -1 {
		// allow WHERE at end without trailing space
		if strings.HasSuffix(upperStatement, " WHERE") {
			return true
		}
		return strings.HasSuffix(upperStatement, "WHERE")
	}
	return true
}
