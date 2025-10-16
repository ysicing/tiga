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
	// ErrSQLInjectionPattern indicates potential SQL injection detected.
	ErrSQLInjectionPattern = errors.New("potential SQL injection pattern detected")
	// ErrSQLUnionInjection indicates UNION-based injection attempt.
	ErrSQLUnionInjection = errors.New("UNION-based SQL injection detected")
)

// Compiled regex patterns for security checks (case-insensitive, word boundary)
var (
	ddlPattern         = regexp.MustCompile(`(?i)\b(DROP|TRUNCATE|ALTER|CREATE)\b`)
	dmlWithoutWhere    = regexp.MustCompile(`(?i)\b(UPDATE|DELETE)\b.*\bFROM\b`)
	whereClausePattern = regexp.MustCompile(`(?i)\bWHERE\b`)
	semicolonPattern   = regexp.MustCompile(`;`)

	// SQL injection patterns
	unionPattern       = regexp.MustCompile(`(?i)\bUNION\s+(ALL\s+)?SELECT\b`)
	hexEncodingPattern = regexp.MustCompile(`(?i)0x[0-9a-f]{2,}`)
	sleepPattern       = regexp.MustCompile(`(?i)\b(SLEEP|BENCHMARK|WAITFOR)\b`)
	stackedQueryPattern = regexp.MustCompile(`;\s*\w`)
)

// SecurityFilter validates SQL and Redis commands against the project's safety rules.
type SecurityFilter struct {
	bannedStatements []string
	bannedFunctions  []string
	redisBlacklist   map[string]struct{}

	// Optional whitelist mode (disabled by default for flexibility)
	enableWhitelist  bool
	allowedTables    map[string]struct{}
	allowedColumns   map[string]struct{}
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
			"CREATE VIEW",
			"CREATE PROCEDURE",
			"CREATE FUNCTION",
			"CREATE TRIGGER",
			"RENAME ",
			"GRANT ",
			"REVOKE ",
			"LOCK TABLES",
			"UNLOCK TABLES",
		},
		bannedFunctions: []string{
			"LOAD_FILE",
			"INTO OUTFILE",
			"DUMPFILE",
			"XP_CMDSHELL",
			"EXEC(",
			"EXECUTE(",
			"SHELL_EXEC",
			"SYSTEM(",
		},
		redisBlacklist: map[string]struct{}{
			"FLUSHDB":   {},
			"FLUSHALL":  {},
			"SHUTDOWN":  {},
			"CONFIG":    {},
			"SAVE":      {},
			"BGSAVE":    {},
			"BGREWRITEAOF": {},
			"DEBUG":     {},
			"SLAVEOF":   {},
			"REPLICAOF": {},
			"SCRIPT":    {},
			"EVAL":      {},
			"EVALSHA":   {},
			"MODULE":    {},
		},
		enableWhitelist: false,
		allowedTables:   make(map[string]struct{}),
		allowedColumns:  make(map[string]struct{}),
	}
}

// ValidateSQL ensures a SQL query complies with the security policy.
func (f *SecurityFilter) ValidateSQL(query string) error {
	if strings.TrimSpace(query) == "" {
		return nil // Allow empty queries for backward compatibility
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

	// Detect SQL injection patterns
	if err := f.detectSQLInjection(upper); err != nil {
		return err
	}

	// Use word boundary regex to prevent bypasses like "dr/**/op"
	for _, banned := range f.bannedStatements {
		bannedTrimmed := strings.TrimSpace(banned)
		// Special handling for "CREATE INDEX" to match "CREATE UNIQUE INDEX" too
		if bannedTrimmed == "CREATE INDEX" {
			if strings.Contains(upper, "CREATE") && strings.Contains(upper, "INDEX") {
				return fmt.Errorf("%w: %s", ErrSQLDangerousOperation, bannedTrimmed)
			}
		} else {
			pattern := regexp.MustCompile(`(?i)\b` + bannedTrimmed + `\b`)
			if pattern.MatchString(upper) {
				return fmt.Errorf("%w: %s", ErrSQLDangerousOperation, bannedTrimmed)
			}
		}
	}

	firstKeyword := extractFirstKeyword(upper)
	if firstKeyword == "UPDATE" || firstKeyword == "DELETE" {
		if !containsWhereClause(upper) {
			return ErrSQLMissingWhere
		}
	}

	for _, fn := range f.bannedFunctions {
		fnUpper := strings.ToUpper(fn)
		// Handle functions already ending with '(' (like "EXEC(")
		if strings.HasSuffix(fnUpper, "(") {
			if strings.Contains(upper, fnUpper) {
				return fmt.Errorf("%w: %s", ErrSQLDangerousFunction, strings.TrimSuffix(fnUpper, "("))
			}
		} else {
			// For keywords/functions without '(', check directly or with '('
			if strings.Contains(upper, fnUpper+" ") || strings.Contains(upper, fnUpper+"(") {
				return fmt.Errorf("%w: %s", ErrSQLDangerousFunction, fnUpper)
			}
		}
	}

	return nil
}

// detectSQLInjection checks for common SQL injection patterns
func (f *SecurityFilter) detectSQLInjection(upperStatement string) error {
	// UNION-based injection
	if unionPattern.MatchString(upperStatement) {
		return ErrSQLUnionInjection
	}

	// Time-based blind injection
	if sleepPattern.MatchString(upperStatement) {
		return fmt.Errorf("%w: time-based blind injection (SLEEP/BENCHMARK/WAITFOR)", ErrSQLInjectionPattern)
	}

	// Hex encoding bypass attempts
	if hexEncodingPattern.MatchString(upperStatement) {
		hexCount := len(hexEncodingPattern.FindAllString(upperStatement, -1))
		if hexCount > 2 { // Allow limited hex values, but not excessive ones
			return fmt.Errorf("%w: excessive hex encoding detected", ErrSQLInjectionPattern)
		}
	}

	return nil
}

// EnableWhitelist enables table/column whitelist validation
func (f *SecurityFilter) EnableWhitelist(tables, columns []string) {
	f.enableWhitelist = true
	f.allowedTables = make(map[string]struct{}, len(tables))
	f.allowedColumns = make(map[string]struct{}, len(columns))

	for _, t := range tables {
		f.allowedTables[strings.ToUpper(t)] = struct{}{}
	}
	for _, c := range columns {
		f.allowedColumns[strings.ToUpper(c)] = struct{}{}
	}
}

// DisableWhitelist disables whitelist validation (default mode)
func (f *SecurityFilter) DisableWhitelist() {
	f.enableWhitelist = false
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
		return nil // Allow empty commands for backward compatibility
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
