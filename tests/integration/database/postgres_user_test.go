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

func TestPostgreSQLUserAndPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL 15 container using testcontainers
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "test123456",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	// Get container host and port
	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	port, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Test 1: Create database
	t.Run("CreateDatabase", func(t *testing.T) {
		driver := dbdriver.NewPostgresDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "postgres",
			Password: "test123456",
			Database: "postgres", // Connect to default database first
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		// Create test database
		opts := dbdriver.CreateDatabaseOptions{
			Name:  "testapp",
			Owner: "postgres",
		}

		err = driver.CreateDatabase(ctx, opts)
		require.NoError(t, err)

		// Verify database was created
		databases, err := driver.ListDatabases(ctx)
		require.NoError(t, err)

		var found bool
		for _, db := range databases {
			if db.Name == "testapp" {
				found = true
				assert.Equal(t, "postgres", db.Owner)
				break
			}
		}
		assert.True(t, found, "testapp database should be created")
	})

	// Test 2: Create user with readonly permissions
	t.Run("CreateUserWithReadonlyPermissions", func(t *testing.T) {
		driver := dbdriver.NewPostgresDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "postgres",
			Password: "test123456",
			Database: "testapp",
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		// Create readonly user
		userOpts := dbdriver.CreateUserOptions{
			Username: "readonly_user",
			Password: "readonly123",
			Roles:    []string{"readonly"},
		}

		err = driver.CreateUser(ctx, userOpts)
		require.NoError(t, err)

		// Verify user was created
		users, err := driver.ListUsers(ctx)
		require.NoError(t, err)

		var foundUser bool
		for _, user := range users {
			if user.Username == "readonly_user" {
				foundUser = true
				break
			}
		}
		assert.True(t, foundUser, "readonly_user should be created")
	})

	// Test 3: Grant readonly permissions
	t.Run("GrantReadonlyPermissions", func(t *testing.T) {
		driver := dbdriver.NewPostgresDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "postgres",
			Password: "test123456",
			Database: "testapp",
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		// Create a test table first
		_, err = driver.ExecuteQuery(ctx, dbdriver.QueryRequest{
			Query: "CREATE TABLE IF NOT EXISTS test_table (id SERIAL PRIMARY KEY, name VARCHAR(100))",
		})
		require.NoError(t, err)

		// Grant SELECT permission to readonly_user
		grantSQL := `
			GRANT CONNECT ON DATABASE testapp TO readonly_user;
			GRANT USAGE ON SCHEMA public TO readonly_user;
			GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly_user;
		`
		_, err = driver.ExecuteQuery(ctx, dbdriver.QueryRequest{
			Query: grantSQL,
		})
		require.NoError(t, err)

		// Test: Try to connect as readonly_user and SELECT (should work)
		readonlyDriver := dbdriver.NewPostgresDriver()
		readonlyCfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "readonly_user",
			Password: "readonly123",
			Database: "testapp",
		}

		err = readonlyDriver.Connect(ctx, readonlyCfg)
		require.NoError(t, err)
		defer readonlyDriver.Disconnect(ctx)

		// SELECT should succeed
		_, err = readonlyDriver.ExecuteQuery(ctx, dbdriver.QueryRequest{
			Query: "SELECT * FROM test_table",
		})
		assert.NoError(t, err, "SELECT should succeed for readonly user")

		// INSERT should fail
		_, err = readonlyDriver.ExecuteQuery(ctx, dbdriver.QueryRequest{
			Query: "INSERT INTO test_table (name) VALUES ('test')",
		})
		assert.Error(t, err, "INSERT should fail for readonly user")
		assert.Contains(t, err.Error(), "permission denied", "Error should indicate permission denied")
	})

	// Test 4: Create user with readwrite permissions
	t.Run("CreateUserWithReadWritePermissions", func(t *testing.T) {
		driver := dbdriver.NewPostgresDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "postgres",
			Password: "test123456",
			Database: "testapp",
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		// Create readwrite user
		userOpts := dbdriver.CreateUserOptions{
			Username: "readwrite_user",
			Password: "readwrite123",
			Roles:    []string{"readwrite"},
		}

		err = driver.CreateUser(ctx, userOpts)
		require.NoError(t, err)

		// Grant ALL permissions to readwrite_user
		grantSQL := `
			GRANT CONNECT ON DATABASE testapp TO readwrite_user;
			GRANT ALL PRIVILEGES ON DATABASE testapp TO readwrite_user;
			GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO readwrite_user;
			GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO readwrite_user;
		`
		_, err = driver.ExecuteQuery(ctx, dbdriver.QueryRequest{
			Query: grantSQL,
		})
		require.NoError(t, err)

		// Test: Connect as readwrite_user and perform INSERT (should work)
		rwDriver := dbdriver.NewPostgresDriver()
		rwCfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "readwrite_user",
			Password: "readwrite123",
			Database: "testapp",
		}

		err = rwDriver.Connect(ctx, rwCfg)
		require.NoError(t, err)
		defer rwDriver.Disconnect(ctx)

		// INSERT should succeed
		_, err = rwDriver.ExecuteQuery(ctx, dbdriver.QueryRequest{
			Query: "INSERT INTO test_table (name) VALUES ('readwrite_test')",
		})
		assert.NoError(t, err, "INSERT should succeed for readwrite user")

		// DELETE should succeed
		_, err = rwDriver.ExecuteQuery(ctx, dbdriver.QueryRequest{
			Query: "DELETE FROM test_table WHERE name = 'readwrite_test'",
		})
		assert.NoError(t, err, "DELETE should succeed for readwrite user")
	})

	// Test 5: Verify permission policy storage
	t.Run("VerifyPermissionPolicyStorage", func(t *testing.T) {
		// This validates that permission policies are correctly stored
		// The actual permission storage verification would require:
		// - PermissionRepository to be injected
		// - Checking that PermissionPolicy records are created with correct role

		// For this integration test, we verify the PostgreSQL-level permissions work correctly
		// The policy storage is tested in unit tests with mock repositories

		driver := dbdriver.NewPostgresDriver()

		cfg := dbdriver.ConnectionConfig{
			Host:     host,
			Port:     port.Int(),
			Username: "postgres",
			Password: "test123456",
			Database: "testapp",
		}

		err := driver.Connect(ctx, cfg)
		require.NoError(t, err)
		defer driver.Disconnect(ctx)

		// Query PostgreSQL system tables to verify grants
		result, err := driver.ExecuteQuery(ctx, dbdriver.QueryRequest{
			Query: `
				SELECT grantee, privilege_type
				FROM information_schema.table_privileges
				WHERE table_name = 'test_table'
				AND grantee IN ('readonly_user', 'readwrite_user')
				ORDER BY grantee, privilege_type
			`,
		})
		require.NoError(t, err)

		// Verify readonly_user has only SELECT
		assert.Greater(t, result.RowCount, 0, "Should have permission grants")

		var hasReadonlySelect, hasReadwriteAll bool
		for _, row := range result.Rows {
			grantee, _ := row["grantee"].(string)
			privilege, _ := row["privilege_type"].(string)

			if grantee == "readonly_user" && privilege == "SELECT" {
				hasReadonlySelect = true
			}
			if grantee == "readwrite_user" && (privilege == "INSERT" || privilege == "DELETE" || privilege == "UPDATE") {
				hasReadwriteAll = true
			}
		}

		assert.True(t, hasReadonlySelect, "readonly_user should have SELECT privilege")
		assert.True(t, hasReadwriteAll, "readwrite_user should have write privileges")
	})
}
