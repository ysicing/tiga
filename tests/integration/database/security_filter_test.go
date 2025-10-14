package database_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	dbservices "github.com/ysicing/tiga/internal/services/database"
)

func TestSQLSecurityFilter(t *testing.T) {
	filter := dbservices.NewSecurityFilter()

	// Test 1: DDL operations should be blocked
	t.Run("BlockDDLOperations", func(t *testing.T) {
		testCases := []struct {
			name  string
			query string
		}{
			{"DROP TABLE", "DROP TABLE users"},
			{"TRUNCATE", "TRUNCATE TABLE logs"},
			{"ALTER TABLE", "ALTER TABLE users ADD COLUMN age INT"},
			{"CREATE DATABASE", "CREATE DATABASE testdb"},
			{"CREATE TABLE", "CREATE TABLE test (id INT)"},
			{"CREATE INDEX", "CREATE INDEX idx_name ON users(name)"},
			{"RENAME TABLE", "RENAME TABLE old_users TO users"},
			{"DROP DATABASE", "DROP DATABASE testdb"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateSQL(tc.query)
				assert.Error(t, err, "Should block DDL: "+tc.query)
				assert.ErrorIs(t, err, dbservices.ErrSQLDangerousOperation)
			})
		}
	})

	// Test 2: UPDATE/DELETE without WHERE should be blocked
	t.Run("BlockDMLWithoutWhere", func(t *testing.T) {
		testCases := []struct {
			name  string
			query string
		}{
			{"UPDATE without WHERE", "UPDATE users SET status = 'inactive'"},
			{"DELETE without WHERE", "DELETE FROM logs"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateSQL(tc.query)
				assert.Error(t, err, "Should block DML without WHERE: "+tc.query)
				assert.ErrorIs(t, err, dbservices.ErrSQLMissingWhere)
			})
		}
	})

	// Test 3: Dangerous functions should be blocked
	t.Run("BlockDangerousFunctions", func(t *testing.T) {
		testCases := []struct {
			name  string
			query string
		}{
			{"LOAD_FILE", "SELECT LOAD_FILE('/etc/passwd')"},
			{"INTO OUTFILE", "SELECT * FROM users INTO OUTFILE '/tmp/users.txt'"},
			{"DUMPFILE", "SELECT * FROM users INTO DUMPFILE '/tmp/dump.sql'"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateSQL(tc.query)
				assert.Error(t, err, "Should block dangerous function: "+tc.query)
				assert.ErrorIs(t, err, dbservices.ErrSQLDangerousFunction)
			})
		}
	})

	// Test 4: Safe SELECT queries should pass
	t.Run("AllowSafeSelectQueries", func(t *testing.T) {
		testCases := []string{
			"SELECT * FROM users WHERE id = 1",
			"SELECT name, email FROM users WHERE status = 'active'",
			"SELECT COUNT(*) FROM orders WHERE created_at > '2024-01-01'",
			"SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id WHERE u.id = 10",
			"SELECT * FROM products WHERE price BETWEEN 10 AND 100",
			"SELECT * FROM users WHERE name LIKE 'John%'",
		}

		for _, query := range testCases {
			t.Run(query, func(t *testing.T) {
				err := filter.ValidateSQL(query)
				assert.NoError(t, err, "Should allow safe SELECT: "+query)
			})
		}
	})

	// Test 5: Safe UPDATE/DELETE with WHERE should pass
	t.Run("AllowDMLWithWhere", func(t *testing.T) {
		testCases := []string{
			"UPDATE users SET status = 'inactive' WHERE id = 1",
			"DELETE FROM logs WHERE created_at < '2024-01-01'",
			"UPDATE products SET price = price * 1.1 WHERE category = 'electronics'",
			"DELETE FROM sessions WHERE expired_at < NOW()",
		}

		for _, query := range testCases {
			t.Run(query, func(t *testing.T) {
				err := filter.ValidateSQL(query)
				assert.NoError(t, err, "Should allow DML with WHERE: "+query)
			})
		}
	})

	// Test 6: Comment injection should not bypass filters
	t.Run("PreventCommentBypass", func(t *testing.T) {
		testCases := []string{
			"DROP /* comment */ TABLE users",
			"SELECT * FROM users -- DROP TABLE logs",
			"TRUNCATE /* bypass */ TABLE sessions",
		}

		for _, query := range testCases {
			t.Run(query, func(t *testing.T) {
				err := filter.ValidateSQL(query)
				assert.Error(t, err, "Should block even with comments: "+query)
			})
		}
	})
}

func TestRedisCommandFilter(t *testing.T) {
	filter := dbservices.NewSecurityFilter()

	// Test 1: Dangerous Redis commands should be blocked
	t.Run("BlockDangerousRedisCommands", func(t *testing.T) {
		testCases := []struct {
			name    string
			command string
		}{
			{"FLUSHDB", "FLUSHDB"},
			{"FLUSHALL", "FLUSHALL"},
			{"SHUTDOWN", "SHUTDOWN"},
			{"CONFIG", "CONFIG SET requirepass newpass"},
			{"SAVE", "SAVE"},
			{"BGSAVE", "BGSAVE"},
			{"Lowercase flushdb", "flushdb"},
			{"Mixed case FlushAll", "FlushAll"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateRedisCommand(tc.command)
				assert.Error(t, err, "Should block dangerous command: "+tc.command)
				assert.ErrorIs(t, err, dbservices.ErrRedisDangerousCommand)
			})
		}
	})

	// Test 2: Safe Redis commands should pass
	t.Run("AllowSafeRedisCommands", func(t *testing.T) {
		testCases := []string{
			"GET mykey",
			"SET mykey myvalue",
			"DEL key1 key2",
			"INCR counter",
			"LPUSH mylist value",
			"HGETALL myhash",
			"ZADD myzset 1 member",
			"KEYS pattern:*",
			"SCAN 0",
			"TTL mykey",
			"EXISTS key1",
		}

		for _, command := range testCases {
			t.Run(command, func(t *testing.T) {
				err := filter.ValidateRedisCommand(command)
				assert.NoError(t, err, "Should allow safe command: "+command)
			})
		}
	})
}

func TestSecurityFilterPerformance(t *testing.T) {
	filter := dbservices.NewSecurityFilter()

	// Test that SQL validation completes within performance target (<2ms)
	t.Run("SQLValidationPerformance", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = 1 AND status = 'active' AND created_at > '2024-01-01'"

		// Warm up
		for i := 0; i < 100; i++ {
			filter.ValidateSQL(query)
		}

		// Measure
		start := time.Now()
		iterations := 1000
		for i := 0; i < iterations; i++ {
			filter.ValidateSQL(query)
		}
		elapsed := time.Since(start)

		avgDuration := elapsed / time.Duration(iterations)
		t.Logf("Average SQL validation time: %v", avgDuration)

		// Should be < 2ms per validation (target from research.md)
		assert.Less(t, avgDuration, 2*time.Millisecond, "SQL validation should complete in <2ms")
	})

	// Test Redis command validation performance
	t.Run("RedisValidationPerformance", func(t *testing.T) {
		command := "GET user:12345:profile"

		// Warm up
		for i := 0; i < 100; i++ {
			filter.ValidateRedisCommand(command)
		}

		// Measure
		start := time.Now()
		iterations := 1000
		for i := 0; i < iterations; i++ {
			filter.ValidateRedisCommand(command)
		}
		elapsed := time.Since(start)

		avgDuration := elapsed / time.Duration(iterations)
		t.Logf("Average Redis validation time: %v", avgDuration)

		// Redis validation should be very fast (string comparison)
		assert.Less(t, avgDuration, 100*time.Microsecond, "Redis validation should complete in <100μs")
	})
}
