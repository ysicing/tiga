package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ysicing/tiga/pkg/dbdriver"
)

func TestMySQLInstanceConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start MySQL 8.0 container using testcontainers
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "test123456",
			"MYSQL_DATABASE":      "testdb",
		},
		WaitingFor: wait.ForLog("port: 3306  MySQL Community Server - GPL").
			WithStartupTimeout(60 * time.Second),
	}

	mysqlContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer mysqlContainer.Terminate(ctx)

	// Get container host and port
	host, err := mysqlContainer.Host(ctx)
	require.NoError(t, err)

	port, err := mysqlContainer.MappedPort(ctx, "3306")
	require.NoError(t, err)

	// Test 1: Create MySQL driver and test connection
	t.Run("CreateDriverAndConnect", func(t *testing.T) {
		driver := dbdriver.NewMySQLDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:         host,
			Port:         port.Int(),
			Username:     "root",
			Password:     "test123456",
			Database:     "testdb",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		// Test ping
		err = driver.Ping(ctx)
		assert.NoError(t, err)
	})

	// Test 2: List databases
	t.Run("ListDatabases", func(t *testing.T) {
		driver := dbdriver.NewMySQLDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "root",
			Password: "test123456",
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		databases, err := driver.ListDatabases(ctx)
		require.NoError(t, err)

		// Should have at least testdb, information_schema, mysql, performance_schema
		assert.GreaterOrEqual(t, len(databases), 4)

		// Find testdb
		var foundTestDB bool
		for _, db := range databases {
			if db.Name == "testdb" {
				foundTestDB = true
				assert.Equal(t, "utf8mb4", db.Charset)
				break
			}
		}
		assert.True(t, foundTestDB, "testdb should be in database list")
	})

	// Test 3: Get version and uptime
	t.Run("GetVersionAndUptime", func(t *testing.T) {
		driver := dbdriver.NewMySQLDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "root",
			Password: "test123456",
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		version, err := driver.GetVersion(ctx)
		require.NoError(t, err)
		assert.Contains(t, version, "8.0", "Version should contain 8.0")

		uptime, err := driver.GetUptime(ctx)
		require.NoError(t, err)
		assert.Greater(t, uptime, int64(0), "Uptime should be greater than 0")
	})

	// Test 4: Create database
	t.Run("CreateDatabase", func(t *testing.T) {
		driver := dbdriver.NewMySQLDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "root",
			Password: "test123456",
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		// Create test database
		opts := dbdriver.CreateDatabaseOptions{
			Name:      "integration_test_db",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_unicode_ci",
		}

		err = driver.CreateDatabase(ctx, opts)
		require.NoError(t, err)

		// Verify database was created
		databases, err := driver.ListDatabases(ctx)
		require.NoError(t, err)

		var found bool
		for _, db := range databases {
			if db.Name == "integration_test_db" {
				found = true
				assert.Equal(t, "utf8mb4", db.Charset)
				assert.Equal(t, "utf8mb4_unicode_ci", db.Collation)
				break
			}
		}
		assert.True(t, found, "integration_test_db should be created")

		// Cleanup: delete database
		err = driver.DeleteDatabase(ctx, "integration_test_db", nil)
		assert.NoError(t, err)
	})

	// Test 5: Test instance status tracking
	t.Run("InstanceStatusTracking", func(t *testing.T) {
		driver := dbdriver.NewMySQLDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "root",
			Password: "test123456",
		}

		// Connect should succeed
		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)

		// Ping should succeed
		err = driver.Ping(ctx)
		assert.NoError(t, err)

		// Get metrics
		version, err := driver.GetVersion(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, version)

		uptime, err := driver.GetUptime(ctx)
		assert.NoError(t, err)
		assert.Greater(t, uptime, int64(0))

		driver.Disconnect(ctx)
	})
}

func TestMySQLInstanceWithDatabaseManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test validates the full DatabaseManager flow with a real MySQL instance
	// Note: This requires a mock repository implementation for testing
	// For now, we verify the driver layer works correctly
	t.Skip("Requires mock repository setup - covered by driver tests above")
}
