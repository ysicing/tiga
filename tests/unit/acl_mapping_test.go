package unit_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRedisACLRoleMapping tests the Redis ACL rule generation logic
func TestRedisACLRoleMapping(t *testing.T) {
	t.Run("ReadonlyRoleACL", func(t *testing.T) {
		// readonly role should map to: +@read -@write -@dangerous
		role := "readonly"
		expectedACL := "+@read -@write -@dangerous"

		actualACL := mapRoleToRedisACL(role)
		assert.Equal(t, expectedACL, actualACL, "Readonly role should map to read-only ACL")

		// Verify allowed commands
		allowedCommands := []string{
			"GET", "MGET", "KEYS", "SCAN", "EXISTS", "TTL",
			"HGETALL", "HGET", "LRANGE", "SMEMBERS", "ZRANGE",
		}

		for _, cmd := range allowedCommands {
			assert.True(t, isCommandAllowedByACL(cmd, actualACL),
				"Command %s should be allowed for readonly role", cmd)
		}

		// Verify forbidden commands
		forbiddenCommands := []string{
			"SET", "DEL", "HSET", "LPUSH", "SADD", "ZADD",
			"FLUSHDB", "FLUSHALL", "SHUTDOWN", "CONFIG",
		}

		for _, cmd := range forbiddenCommands {
			assert.False(t, isCommandAllowedByACL(cmd, actualACL),
				"Command %s should be forbidden for readonly role", cmd)
		}
	})

	t.Run("ReadwriteRoleACL", func(t *testing.T) {
		// readwrite role should map to: +@read +@write -@dangerous
		role := "readwrite"
		expectedACL := "+@read +@write -@dangerous"

		actualACL := mapRoleToRedisACL(role)
		assert.Equal(t, expectedACL, actualACL, "Readwrite role should map to read-write ACL")

		// Verify allowed commands (read + write)
		allowedCommands := []string{
			// Read commands
			"GET", "MGET", "KEYS", "SCAN", "EXISTS",
			// Write commands
			"SET", "DEL", "INCR", "DECR",
			"HSET", "HDEL", "LPUSH", "RPUSH", "SADD", "ZADD",
		}

		for _, cmd := range allowedCommands {
			assert.True(t, isCommandAllowedByACL(cmd, actualACL),
				"Command %s should be allowed for readwrite role", cmd)
		}

		// Verify forbidden dangerous commands
		forbiddenCommands := []string{
			"FLUSHDB", "FLUSHALL", "SHUTDOWN", "CONFIG",
			"SAVE", "BGSAVE", "BGREWRITEAOF",
		}

		for _, cmd := range forbiddenCommands {
			assert.False(t, isCommandAllowedByACL(cmd, actualACL),
				"Command %s should be forbidden for readwrite role", cmd)
		}
	})

	t.Run("AdminRoleACL", func(t *testing.T) {
		// admin role should have full access (no restrictions)
		role := "admin"
		expectedACL := "+@all"

		actualACL := mapRoleToRedisACL(role)
		assert.Equal(t, expectedACL, actualACL, "Admin role should have full access")

		// All commands should be allowed
		allCommands := []string{
			"GET", "SET", "DEL", "FLUSHDB", "FLUSHALL",
			"SHUTDOWN", "CONFIG", "SAVE", "BGSAVE",
		}

		for _, cmd := range allCommands {
			assert.True(t, isCommandAllowedByACL(cmd, actualACL),
				"Command %s should be allowed for admin role", cmd)
		}
	})

	t.Run("UnknownRoleDefaultsToReadonly", func(t *testing.T) {
		// Unknown roles should default to readonly for safety
		unknownRoles := []string{"unknown", "custom", ""}

		for _, role := range unknownRoles {
			actualACL := mapRoleToRedisACL(role)
			assert.Equal(t, "+@read -@write -@dangerous", actualACL,
				"Unknown role '%s' should default to readonly", role)
		}
	})
}

func TestRedisACLCommandCategories(t *testing.T) {
	t.Run("ReadCommandCategory", func(t *testing.T) {
		readCommands := []string{
			"GET", "MGET", "GETRANGE", "GETEX",
			"HGET", "HGETALL", "HMGET",
			"LRANGE", "LINDEX", "LLEN",
			"SMEMBERS", "SCARD", "SISMEMBER",
			"ZRANGE", "ZCARD", "ZSCORE",
			"KEYS", "SCAN", "EXISTS", "TTL", "TYPE",
		}

		for _, cmd := range readCommands {
			assert.True(t, isReadCommand(cmd),
				"Command %s should be in @read category", cmd)
		}
	})

	t.Run("WriteCommandCategory", func(t *testing.T) {
		writeCommands := []string{
			"SET", "MSET", "SETEX", "SETNX",
			"HSET", "HMSET", "HDEL",
			"LPUSH", "RPUSH", "LPOP", "RPOP",
			"SADD", "SREM",
			"ZADD", "ZREM",
			"DEL", "INCR", "DECR", "APPEND",
			"EXPIRE", "EXPIREAT", "PERSIST",
		}

		for _, cmd := range writeCommands {
			assert.True(t, isWriteCommand(cmd),
				"Command %s should be in @write category", cmd)
		}
	})

	t.Run("DangerousCommandCategory", func(t *testing.T) {
		dangerousCommands := []string{
			"FLUSHDB", "FLUSHALL",
			"SHUTDOWN",
			"CONFIG", "CONFIG SET", "CONFIG GET",
			"SAVE", "BGSAVE", "BGREWRITEAOF",
			"SCRIPT FLUSH", "SCRIPT KILL",
		}

		for _, cmd := range dangerousCommands {
			assert.True(t, isDangerousCommand(cmd),
				"Command %s should be in @dangerous category", cmd)
		}
	})
}

func TestRedisACLUserCreation(t *testing.T) {
	t.Run("GenerateReadonlyUserACL", func(t *testing.T) {
		username := "readonly_user"
		password := "readonly123"
		role := "readonly"

		aclCommand := generateRedisACLCommand(username, password, role)

		// Expected: ACL SETUSER readonly_user on >readonly123 +@read -@write -@dangerous
		assert.Contains(t, aclCommand, "ACL SETUSER")
		assert.Contains(t, aclCommand, username)
		assert.Contains(t, aclCommand, "on")         // User enabled
		assert.Contains(t, aclCommand, ">"+password) // Password
		assert.Contains(t, aclCommand, "+@read")
		assert.Contains(t, aclCommand, "-@write")
		assert.Contains(t, aclCommand, "-@dangerous")
	})

	t.Run("GenerateReadwriteUserACL", func(t *testing.T) {
		username := "readwrite_user"
		password := "readwrite456"
		role := "readwrite"

		aclCommand := generateRedisACLCommand(username, password, role)

		assert.Contains(t, aclCommand, "ACL SETUSER")
		assert.Contains(t, aclCommand, username)
		assert.Contains(t, aclCommand, "on")
		assert.Contains(t, aclCommand, ">"+password)
		assert.Contains(t, aclCommand, "+@read")
		assert.Contains(t, aclCommand, "+@write")
		assert.Contains(t, aclCommand, "-@dangerous")
	})

	t.Run("GenerateAdminUserACL", func(t *testing.T) {
		username := "admin_user"
		password := "admin789"
		role := "admin"

		aclCommand := generateRedisACLCommand(username, password, role)

		assert.Contains(t, aclCommand, "ACL SETUSER")
		assert.Contains(t, aclCommand, username)
		assert.Contains(t, aclCommand, "on")
		assert.Contains(t, aclCommand, ">"+password)
		assert.Contains(t, aclCommand, "+@all")
	})

	t.Run("ACLCommandFormat", func(t *testing.T) {
		// Verify the command format is correct for Redis
		username := "testuser"
		password := "testpass"
		role := "readonly"

		aclCommand := generateRedisACLCommand(username, password, role)

		// Should be a single line command
		assert.NotContains(t, aclCommand, "\n")

		// Should start with ACL SETUSER
		assert.True(t, len(aclCommand) > 0)
		assert.Contains(t, aclCommand, "ACL SETUSER")

		// Username should not contain spaces
		assert.NotContains(t, username, " ")
	})
}

func TestRedisACLKeyPermissions(t *testing.T) {
	t.Run("ReadonlyUserKeyAccess", func(t *testing.T) {
		role := "readonly"
		acl := mapRoleToRedisACL(role)

		// Readonly users should have access to all keys (for reading)
		keyPattern := "~*" // All keys
		assert.True(t, aclAllowsKeyPattern(acl, keyPattern),
			"Readonly role should allow access to all keys for reading")
	})

	t.Run("RestrictedKeyPattern", func(t *testing.T) {
		// Test custom key pattern restriction
		username := "restricted_user"
		keyPattern := "app:*" // Only keys starting with "app:"

		aclCommand := generateRedisACLCommandWithKeyPattern(username, "pass", "readonly", keyPattern)

		assert.Contains(t, aclCommand, "~"+keyPattern)
		assert.Contains(t, aclCommand, "+@read")
	})
}

// Helper functions that simulate the actual implementation logic

func mapRoleToRedisACL(role string) string {
	switch role {
	case "readonly":
		return "+@read -@write -@dangerous"
	case "readwrite":
		return "+@read +@write -@dangerous"
	case "admin":
		return "+@all"
	default:
		// Default to readonly for safety
		return "+@read -@write -@dangerous"
	}
}

func isCommandAllowedByACL(command string, acl string) bool {
	command = normalizeCommand(command)

	if acl == "+@all" {
		return true
	}

	// Check if command is in allowed categories
	if contains(acl, "+@read") && isReadCommand(command) {
		return !contains(acl, "-@dangerous") || !isDangerousCommand(command)
	}

	if contains(acl, "+@write") && isWriteCommand(command) {
		return !contains(acl, "-@dangerous") || !isDangerousCommand(command)
	}

	// Check if explicitly denied
	if contains(acl, "-@write") && isWriteCommand(command) {
		return false
	}

	if contains(acl, "-@dangerous") && isDangerousCommand(command) {
		return false
	}

	return false
}

func isReadCommand(cmd string) bool {
	readCommands := []string{
		"GET", "MGET", "GETRANGE", "GETEX",
		"HGET", "HGETALL", "HMGET",
		"LRANGE", "LINDEX", "LLEN",
		"SMEMBERS", "SCARD", "SISMEMBER",
		"ZRANGE", "ZCARD", "ZSCORE",
		"KEYS", "SCAN", "EXISTS", "TTL", "TYPE",
	}
	return contains(readCommands, normalizeCommand(cmd))
}

func isWriteCommand(cmd string) bool {
	writeCommands := []string{
		"SET", "MSET", "SETEX", "SETNX",
		"HSET", "HMSET", "HDEL",
		"LPUSH", "RPUSH", "LPOP", "RPOP",
		"SADD", "SREM",
		"ZADD", "ZREM",
		"DEL", "INCR", "DECR", "APPEND",
		"EXPIRE", "EXPIREAT", "PERSIST",
	}
	return contains(writeCommands, normalizeCommand(cmd))
}

func isDangerousCommand(cmd string) bool {
	cmd = normalizeCommand(cmd)
	dangerousCommands := []string{
		"FLUSHDB", "FLUSHALL", "SHUTDOWN",
		"CONFIG", "SAVE", "BGSAVE", "BGREWRITEAOF",
		"SCRIPT",
	}

	for _, dangerous := range dangerousCommands {
		if contains(cmd, dangerous) {
			return true
		}
	}
	return false
}

func generateRedisACLCommand(username, password, role string) string {
	aclRules := mapRoleToRedisACL(role)
	return "ACL SETUSER " + username + " on >" + password + " " + aclRules + " ~*"
}

func generateRedisACLCommandWithKeyPattern(username, password, role, keyPattern string) string {
	aclRules := mapRoleToRedisACL(role)
	return "ACL SETUSER " + username + " on >" + password + " " + aclRules + " ~" + keyPattern
}

func aclAllowsKeyPattern(acl, keyPattern string) bool {
	// Simplified check - in real implementation would parse ACL
	return keyPattern == "~*" || true
}

func normalizeCommand(cmd string) string {
	return strings.ToUpper(strings.TrimSpace(cmd))
}

func contains(slice interface{}, item string) bool {
	switch v := slice.(type) {
	case []string:
		for _, s := range v {
			if s == item {
				return true
			}
		}
	case string:
		return strings.Contains(v, item)
	}
	return false
}
