package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"

	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	postgresdriver "gorm.io/driver/postgres"
)

// setupTestDB creates a PostgreSQL testcontainer and initializes the database
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires Docker)")
	}

	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgrescontainer.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgrescontainer.WithDatabase("testdb"),
		postgrescontainer.WithUsername("testuser"),
		postgrescontainer.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	db, err := gorm.Open(postgresdriver.Open(connStr), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate schemas
	err = db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Instance{},
	)
	require.NoError(t, err)

	// Cleanup function
	cleanup := func() {
		_ = postgresContainer.Terminate(ctx)
	}

	return db, cleanup
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashed_password",
		FullName: "Test User",
		Status:   "active",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// TestInstanceRepository_Create tests instance creation
func TestInstanceRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	// Create test user first
	user := createTestUser(t, db)

	testCases := []struct {
		name        string
		instance    *models.Instance
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid Instance",
			instance: &models.Instance{
				Name:        "test-minio",
				DisplayName: "Test MinIO",
				Type:        "minio",
				Connection:  models.JSONB{"endpoint": "localhost:9000"},
				Status:      "running",
				Health:      "healthy",
				Environment: "dev",
				OwnerID:     user.ID,
			},
			shouldError: false,
		},
		{
			name: "Minimal Instance",
			instance: &models.Instance{
				Name:       "test-redis",
				Type:       "redis",
				Connection: models.JSONB{},
				OwnerID:    user.ID,
			},
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.Create(ctx, tc.instance)

			if tc.shouldError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tc.instance.ID)
			}
		})
	}
}

// TestInstanceRepository_GetByID tests retrieving instance by ID
func TestInstanceRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	// Create test user
	user := createTestUser(t, db)

	// Create test instance
	instance := &models.Instance{
		Name:       "test-instance",
		Type:       "minio",
		Connection: models.JSONB{},
		OwnerID:    user.ID,
	}
	err := repo.Create(ctx, instance)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		id          uuid.UUID
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Existing Instance",
			id:          instance.ID,
			shouldError: false,
		},
		{
			name:        "Non-existent Instance",
			id:          uuid.New(),
			shouldError: true,
			errorMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetByID(ctx, tc.id)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.id, result.ID)
			}
		})
	}
}

// TestInstanceRepository_GetByName tests retrieving instance by name
func TestInstanceRepository_GetByName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instance
	instance := &models.Instance{
		Name:       "unique-name",
		Type:       "mysql",
		Connection: models.JSONB{},
		OwnerID:    user.ID,
	}
	err := repo.Create(ctx, instance)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		instanceName string
		shouldError  bool
	}{
		{
			name:         "Existing Name",
			instanceName: "unique-name",
			shouldError:  false,
		},
		{
			name:         "Non-existent Name",
			instanceName: "non-existent",
			shouldError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetByName(ctx, tc.instanceName)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.instanceName, result.Name)
			}
		})
	}
}

// TestInstanceRepository_Update tests instance update
func TestInstanceRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instance
	instance := &models.Instance{
		Name:       "update-test",
		Type:       "redis",
		Connection: models.JSONB{},
		Status:     "running",
		OwnerID:    user.ID,
	}
	err := repo.Create(ctx, instance)
	require.NoError(t, err)

	// Update instance
	instance.Status = "stopped"
	instance.DisplayName = "Updated Name"

	err = repo.Update(ctx, instance)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, instance.ID)
	assert.NoError(t, err)
	assert.Equal(t, "stopped", updated.Status)
	assert.Equal(t, "Updated Name", updated.DisplayName)
}

// TestInstanceRepository_UpdateFields tests partial field updates
func TestInstanceRepository_UpdateFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instance
	instance := &models.Instance{
		Name:       "field-update-test",
		Type:       "docker",
		Connection: models.JSONB{},
		Status:     "running",
		Version:    "1.0.0",
		OwnerID:    user.ID,
	}
	err := repo.Create(ctx, instance)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		id          uuid.UUID
		fields      map[string]interface{}
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Update Single Field",
			id:   instance.ID,
			fields: map[string]interface{}{
				"status": "stopped",
			},
			shouldError: false,
		},
		{
			name: "Update Multiple Fields",
			id:   instance.ID,
			fields: map[string]interface{}{
				"version": "2.0.0",
				"health":  "healthy",
			},
			shouldError: false,
		},
		{
			name: "Non-existent Instance",
			id:   uuid.New(),
			fields: map[string]interface{}{
				"status": "stopped",
			},
			shouldError: true,
			errorMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.UpdateFields(ctx, tc.id, tc.fields)

			if tc.shouldError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify updates
				updated, err := repo.GetByID(ctx, tc.id)
				assert.NoError(t, err)
				for field, value := range tc.fields {
					switch field {
					case "status":
						assert.Equal(t, value, updated.Status)
					case "version":
						assert.Equal(t, value, updated.Version)
					case "health":
						assert.Equal(t, value, updated.Health)
					}
				}
			}
		})
	}
}

// TestInstanceRepository_Delete tests soft delete
func TestInstanceRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instance
	instance := &models.Instance{
		Name:       "delete-test",
		Type:       "mysql",
		Connection: models.JSONB{},
		OwnerID:    user.ID,
	}
	err := repo.Create(ctx, instance)
	require.NoError(t, err)

	// Delete instance
	err = repo.Delete(ctx, instance.ID)
	assert.NoError(t, err)

	// Verify soft delete
	_, err = repo.GetByID(ctx, instance.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Verify it still exists in DB with deleted_at
	var deletedInstance models.Instance
	err = db.Unscoped().First(&deletedInstance, "id = ?", instance.ID).Error
	assert.NoError(t, err)
	assert.NotNil(t, deletedInstance.DeletedAt)
}

// TestInstanceRepository_ListInstances tests listing with filters
func TestInstanceRepository_ListInstances(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instances
	instances := []*models.Instance{
		{
			Name:        "minio-dev",
			Type:        "minio",
			Connection:  models.JSONB{},
			Status:      "running",
			Environment: "dev",
			Tags:        models.StringArray{"storage", "dev"},
			OwnerID:     user.ID,
		},
		{
			Name:        "mysql-prod",
			Type:        "mysql",
			Connection:  models.JSONB{},
			Status:      "running",
			Environment: "prod",
			Tags:        models.StringArray{"database", "prod"},
			OwnerID:     user.ID,
		},
		{
			Name:        "redis-dev",
			Type:        "redis",
			Connection:  models.JSONB{},
			Status:      "stopped",
			Environment: "dev",
			Tags:        models.StringArray{"cache", "dev"},
			OwnerID:     user.ID,
		},
	}

	for _, inst := range instances {
		err := repo.Create(ctx, inst)
		require.NoError(t, err)
	}

	testCases := []struct {
		name          string
		filter        *ListInstancesFilter
		expectedCount int
	}{
		{
			name:          "No Filter",
			filter:        &ListInstancesFilter{},
			expectedCount: 3,
		},
		{
			name: "Filter by Type",
			filter: &ListInstancesFilter{
				ServiceType: "minio",
			},
			expectedCount: 1,
		},
		{
			name: "Filter by Status",
			filter: &ListInstancesFilter{
				Status: "running",
			},
			expectedCount: 2,
		},
		{
			name: "Filter by Environment",
			filter: &ListInstancesFilter{
				Environment: "dev",
			},
			expectedCount: 2,
		},
		{
			name: "Filter by Tag",
			filter: &ListInstancesFilter{
				Tags: []string{"dev"},
			},
			expectedCount: 2,
		},
		{
			name: "Search by Name",
			filter: &ListInstancesFilter{
				Search: "mysql",
			},
			expectedCount: 1,
		},
		{
			name: "With Pagination",
			filter: &ListInstancesFilter{
				Page:     1,
				PageSize: 2,
			},
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, total, err := repo.ListInstances(ctx, tc.filter)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(results))
			if tc.filter.Page == 0 {
				assert.Equal(t, int64(3), total)
			}
		})
	}
}

// TestInstanceRepository_CountByServiceType tests counting by service type
func TestInstanceRepository_CountByServiceType(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instances
	for i := 0; i < 3; i++ {
		instance := &models.Instance{
			Name:       "minio-" + string(rune(i+'0')),
			Type:       "minio",
			Connection: models.JSONB{},
			OwnerID:    user.ID,
		}
		err := repo.Create(ctx, instance)
		require.NoError(t, err)
	}

	count, err := repo.CountByServiceType(ctx, "minio")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)

	count, err = repo.CountByServiceType(ctx, "redis")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// TestInstanceRepository_ExistsName tests name existence check
func TestInstanceRepository_ExistsName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instance
	instance := &models.Instance{
		Name:       "exists-test",
		Type:       "docker",
		Connection: models.JSONB{},
		OwnerID:    user.ID,
	}
	err := repo.Create(ctx, instance)
	require.NoError(t, err)

	// Test existence
	exists, err := repo.ExistsName(ctx, "exists-test", nil)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test non-existence
	exists, err = repo.ExistsName(ctx, "non-existent", nil)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test with exclude ID
	exists, err = repo.ExistsName(ctx, "exists-test", &instance.ID)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestInstanceRepository_UpdateHealth tests health status update
func TestInstanceRepository_UpdateHealth(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instance
	instance := &models.Instance{
		Name:       "health-test",
		Type:       "redis",
		Connection: models.JSONB{},
		Health:     "unknown",
		OwnerID:    user.ID,
	}
	err := repo.Create(ctx, instance)
	require.NoError(t, err)

	// Update health
	healthMsg := "Service is healthy"
	err = repo.UpdateHealth(ctx, instance.ID, "healthy", &healthMsg)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, instance.ID)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", updated.Health)
	assert.Equal(t, healthMsg, updated.HealthMessage)
}

// TestInstanceRepository_TagsOperations tests tag add/remove operations
func TestInstanceRepository_TagsOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create test instance
	instance := &models.Instance{
		Name:       "tags-test",
		Type:       "kubernetes",
		Connection: models.JSONB{},
		Tags:       models.StringArray{"initial"},
		OwnerID:    user.ID,
	}
	err := repo.Create(ctx, instance)
	require.NoError(t, err)

	// Add tags
	err = repo.AddTags(ctx, instance.ID, []string{"new1", "new2"})
	assert.NoError(t, err)

	// Verify tags added
	updated, err := repo.GetByID(ctx, instance.ID)
	assert.NoError(t, err)
	assert.Contains(t, updated.Tags, "initial")
	assert.Contains(t, updated.Tags, "new1")
	assert.Contains(t, updated.Tags, "new2")

	// Remove tags
	err = repo.RemoveTags(ctx, instance.ID, []string{"new1"})
	assert.NoError(t, err)

	// Verify tags removed
	updated, err = repo.GetByID(ctx, instance.ID)
	assert.NoError(t, err)
	assert.Contains(t, updated.Tags, "initial")
	assert.Contains(t, updated.Tags, "new2")
	assert.NotContains(t, updated.Tags, "new1")
}

// TestInstanceRepository_GetStatistics tests statistics retrieval
func TestInstanceRepository_GetStatistics(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewInstanceRepository(db)
	ctx := context.Background()

	user := createTestUser(t, db)

	// Create diverse test instances
	instances := []*models.Instance{
		{
			Name:        "minio-1",
			Type:        "minio",
			Connection:  models.JSONB{},
			Status:      "running",
			Health:      "healthy",
			Environment: "dev",
			OwnerID:     user.ID,
		},
		{
			Name:        "mysql-1",
			Type:        "mysql",
			Connection:  models.JSONB{},
			Status:      "running",
			Health:      "unhealthy",
			Environment: "prod",
			OwnerID:     user.ID,
		},
		{
			Name:        "redis-1",
			Type:        "redis",
			Connection:  models.JSONB{},
			Status:      "stopped",
			Health:      "healthy",
			Environment: "dev",
			OwnerID:     user.ID,
		},
	}

	for _, inst := range instances {
		err := repo.Create(ctx, inst)
		require.NoError(t, err)
	}

	// Get statistics
	stats, err := repo.GetStatistics(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// Verify counts
	assert.Equal(t, int64(3), stats.TotalInstances)
	assert.Equal(t, int64(1), stats.ByServiceType["minio"])
	assert.Equal(t, int64(1), stats.ByServiceType["mysql"])
	assert.Equal(t, int64(1), stats.ByServiceType["redis"])
	assert.Equal(t, int64(2), stats.ByStatus["running"])
	assert.Equal(t, int64(1), stats.ByStatus["stopped"])
	assert.Equal(t, int64(2), stats.ByEnvironment["dev"])
	assert.Equal(t, int64(1), stats.ByEnvironment["prod"])
	assert.Equal(t, int64(2), stats.HealthyInstances)
	assert.Equal(t, int64(1), stats.UnhealthyInstances)
}
