package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	dbservices "github.com/ysicing/tiga/internal/services/database"
)

func TestSecurityFilterSQLParsing(t *testing.T) {
	filter := dbservices.NewSecurityFilter()

	t.Run("DDLDetection", func(t *testing.T) {
		testCases := []struct {
			name        string
			sql         string
			shouldBlock bool
		}{
			// DDL operations - should be blocked
			{"DROP TABLE", "DROP TABLE users", true},
			{"DROP TABLE with IF EXISTS", "DROP TABLE IF EXISTS users", true},
			{"TRUNCATE", "TRUNCATE TABLE logs", true},
			{"ALTER TABLE ADD", "ALTER TABLE users ADD COLUMN age INT", true},
			{"ALTER TABLE DROP", "ALTER TABLE users DROP COLUMN age", true},
			{"CREATE TABLE", "CREATE TABLE test (id INT)", true},
			{"CREATE INDEX", "CREATE INDEX idx_name ON users(name)", true},
			{"CREATE UNIQUE INDEX", "CREATE UNIQUE INDEX idx_email ON users(email)", true},
			{"RENAME TABLE", "RENAME TABLE old_users TO users", true},
			{"DROP DATABASE", "DROP DATABASE testdb", true},
			{"CREATE DATABASE", "CREATE DATABASE newdb", true},
			{"ALTER DATABASE", "ALTER DATABASE testdb CHARACTER SET utf8mb4", true},

			// Safe SELECT - should pass
			{"Simple SELECT", "SELECT * FROM users", false},
			{"SELECT with WHERE", "SELECT * FROM users WHERE id = 1", false},
			{"SELECT with JOIN", "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id", false},
			{"SELECT with aggregate", "SELECT COUNT(*) FROM users WHERE status = 'active'", false},
			{"SELECT with subquery", "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateSQL(tc.sql)
				if tc.shouldBlock {
					assert.Error(t, err, "Should block: "+tc.sql)
				} else {
					assert.NoError(t, err, "Should allow: "+tc.sql)
				}
			})
		}
	})

	t.Run("DMLWithoutWhereDetection", func(t *testing.T) {
		testCases := []struct {
			name        string
			sql         string
			shouldBlock bool
		}{
			// UPDATE/DELETE without WHERE - should be blocked
			{"UPDATE without WHERE", "UPDATE users SET status = 'inactive'", true},
			{"DELETE without WHERE", "DELETE FROM logs", true},

			// UPDATE/DELETE with WHERE - should pass
			{"UPDATE with WHERE", "UPDATE users SET status = 'inactive' WHERE id = 1", false},
			{"DELETE with WHERE", "DELETE FROM logs WHERE created_at < '2024-01-01'", false},
			{"UPDATE with complex WHERE", "UPDATE users SET status = 'active' WHERE id > 100 AND created_at > NOW()", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateSQL(tc.sql)
				if tc.shouldBlock {
					assert.Error(t, err, "Should block: "+tc.sql)
				} else {
					assert.NoError(t, err, "Should allow: "+tc.sql)
				}
			})
		}
	})

	t.Run("DangerousFunctionDetection", func(t *testing.T) {
		testCases := []struct {
			name        string
			sql         string
			shouldBlock bool
		}{
			// Dangerous functions - should be blocked
			{"LOAD_FILE", "SELECT LOAD_FILE('/etc/passwd')", true},
			{"INTO OUTFILE", "SELECT * FROM users INTO OUTFILE '/tmp/users.txt'", true},
			{"INTO DUMPFILE", "SELECT * FROM users INTO DUMPFILE '/tmp/dump.sql'", true},

			// Safe functions - should pass
			{"NOW()", "SELECT NOW() FROM users", false},
			{"COUNT()", "SELECT COUNT(*) FROM users", false},
			{"CONCAT()", "SELECT CONCAT(first_name, ' ', last_name) FROM users", false},
			{"DATE_FORMAT()", "SELECT DATE_FORMAT(created_at, '%Y-%m-%d') FROM users", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateSQL(tc.sql)
				if tc.shouldBlock {
					assert.Error(t, err, "Should block: "+tc.sql)
				} else {
					assert.NoError(t, err, "Should allow: "+tc.sql)
				}
			})
		}
	})

	t.Run("CommentBypassPrevention", func(t *testing.T) {
		// SQL injection attempts using comments should still be blocked
		testCases := []string{
			"DROP /* comment */ TABLE users",
			"SELECT * FROM users; -- DROP TABLE logs",
			"TRUNCATE /* inline comment */ TABLE sessions",
			"ALTER TABLE users /* comment */ ADD COLUMN age INT",
		}

		for _, sql := range testCases {
			t.Run(sql, func(t *testing.T) {
				err := filter.ValidateSQL(sql)
				assert.Error(t, err, "Should block SQL with comments: "+sql)
			})
		}
	})

	t.Run("CaseInsensitiveDetection", func(t *testing.T) {
		// Should detect dangerous operations regardless of case
		testCases := []string{
			"drop table users",
			"DROP TABLE users",
			"DrOp TaBlE users",
			"truncate table logs",
			"TRUNCATE TABLE logs",
			"TrUnCaTe TaBlE logs",
		}

		for _, sql := range testCases {
			t.Run(sql, func(t *testing.T) {
				err := filter.ValidateSQL(sql)
				assert.Error(t, err, "Should block regardless of case: "+sql)
			})
		}
	})

	t.Run("EdgeCases", func(t *testing.T) {
		testCases := []struct {
			name        string
			sql         string
			shouldBlock bool
		}{
			{"Empty string", "", false},
			{"Whitespace only", "   \n\t   ", false},
			{"Single word", "SELECT", false},
			{"Multiple statements safe", "SELECT * FROM users; SELECT * FROM orders", false},
			{"Multiple statements mixed", "SELECT * FROM users; DROP TABLE logs", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateSQL(tc.sql)
				if tc.shouldBlock {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestSecurityFilterRedisCommand(t *testing.T) {
	filter := dbservices.NewSecurityFilter()

	t.Run("DangerousCommandDetection", func(t *testing.T) {
		testCases := []struct {
			name        string
			command     string
			shouldBlock bool
		}{
			// Dangerous commands - should be blocked
			{"FLUSHDB", "FLUSHDB", true},
			{"FLUSHALL", "FLUSHALL", true},
			{"SHUTDOWN", "SHUTDOWN", true},
			{"SHUTDOWN SAVE", "SHUTDOWN SAVE", true},
			{"CONFIG SET", "CONFIG SET requirepass newpass", true},
			{"CONFIG GET", "CONFIG GET *", true},
			{"SAVE", "SAVE", true},
			{"BGSAVE", "BGSAVE", true},
			{"BGREWRITEAOF", "BGREWRITEAOF", true},

			// Safe commands - should pass
			{"GET", "GET mykey", false},
			{"SET", "SET mykey myvalue", false},
			{"DEL", "DEL key1 key2 key3", false},
			{"INCR", "INCR counter", false},
			{"DECR", "DECR counter", false},
			{"LPUSH", "LPUSH mylist value", false},
			{"RPUSH", "RPUSH mylist value", false},
			{"LPOP", "LPOP mylist", false},
			{"HGETALL", "HGETALL myhash", false},
			{"HSET", "HSET myhash field value", false},
			{"ZADD", "ZADD myzset 1 member", false},
			{"KEYS", "KEYS pattern:*", false},
			{"SCAN", "SCAN 0", false},
			{"TTL", "TTL mykey", false},
			{"EXPIRE", "EXPIRE mykey 3600", false},
			{"EXISTS", "EXISTS key1", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateRedisCommand(tc.command)
				if tc.shouldBlock {
					assert.Error(t, err, "Should block: "+tc.command)
					assert.ErrorIs(t, err, dbservices.ErrRedisDangerousCommand)
				} else {
					assert.NoError(t, err, "Should allow: "+tc.command)
				}
			})
		}
	})

	t.Run("CaseInsensitiveRedisCommand", func(t *testing.T) {
		testCases := []string{
			"flushdb",
			"FLUSHDB",
			"FlushDb",
			"FLUSHdb",
			"flushall",
			"FLUSHALL",
			"FlushAll",
			"shutdown",
			"SHUTDOWN",
			"ShutDown",
		}

		for _, command := range testCases {
			t.Run(command, func(t *testing.T) {
				err := filter.ValidateRedisCommand(command)
				assert.Error(t, err, "Should block regardless of case: "+command)
			})
		}
	})

	t.Run("CommandWithArguments", func(t *testing.T) {
		testCases := []struct {
			command     string
			shouldBlock bool
		}{
			{"CONFIG SET maxmemory 1gb", true},
			{"CONFIG GET maxmemory", true},
			{"SET user:123:name John", false},
			{"GET user:123:name", false},
			{"DEL user:123:name user:123:email", false},
		}

		for _, tc := range testCases {
			t.Run(tc.command, func(t *testing.T) {
				err := filter.ValidateRedisCommand(tc.command)
				if tc.shouldBlock {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("EdgeCases", func(t *testing.T) {
		testCases := []struct {
			name        string
			command     string
			shouldBlock bool
		}{
			{"Empty string", "", false},
			{"Whitespace only", "   \n\t   ", false},
			{"Single word safe", "PING", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := filter.ValidateRedisCommand(tc.command)
				if tc.shouldBlock {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}
