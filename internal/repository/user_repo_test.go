package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/internal/models"
)

// TestUserRepository_Create tests user creation
func TestUserRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	testCases := []struct {
		name        string
		user        *models.User
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid User",
			user: &models.User{
				Username: "johndoe",
				Email:    "john@example.com",
				Password: "hashed_password",
				FullName: "John Doe",
				Status:   "active",
			},
			shouldError: false,
		},
		{
			name: "Minimal User",
			user: &models.User{
				Username: "minimal",
				Email:    "minimal@example.com",
				Password: "hashed",
			},
			shouldError: false,
		},
		{
			name: "Duplicate Username",
			user: &models.User{
				Username: "johndoe",
				Email:    "different@example.com",
				Password: "hashed",
			},
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.Create(ctx, tc.user)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tc.user.ID)
			}
		})
	}
}

// TestUserRepository_GetByID tests retrieving user by ID
func TestUserRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "getbyid",
		Email:    "getbyid@example.com",
		Password: "hashed",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		id          uuid.UUID
		shouldError bool
	}{
		{
			name:        "Existing User",
			id:          user.ID,
			shouldError: false,
		},
		{
			name:        "Non-existent User",
			id:          uuid.New(),
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetByID(ctx, tc.id)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.id, result.ID)
			}
		})
	}
}

// TestUserRepository_GetByUsername tests retrieving user by username
func TestUserRepository_GetByUsername(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "uniqueuser",
		Email:    "unique@example.com",
		Password: "hashed",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		username    string
		shouldError bool
	}{
		{
			name:        "Existing Username",
			username:    "uniqueuser",
			shouldError: false,
		},
		{
			name:        "Non-existent Username",
			username:    "nonexistent",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetByUsername(ctx, tc.username)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.username, result.Username)
			}
		})
	}
}

// TestUserRepository_GetByEmail tests retrieving user by email
func TestUserRepository_GetByEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "emailuser",
		Email:    "test@domain.com",
		Password: "hashed",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		email       string
		shouldError bool
	}{
		{
			name:        "Existing Email",
			email:       "test@domain.com",
			shouldError: false,
		},
		{
			name:        "Non-existent Email",
			email:       "none@domain.com",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetByEmail(ctx, tc.email)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.email, result.Email)
			}
		})
	}
}

// TestUserRepository_GetByUsernameOrEmail tests dual lookup
func TestUserRepository_GetByUsernameOrEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "dualuser",
		Email:    "dual@example.com",
		Password: "hashed",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		identifier  string
		shouldError bool
	}{
		{
			name:        "Find by Username",
			identifier:  "dualuser",
			shouldError: false,
		},
		{
			name:        "Find by Email",
			identifier:  "dual@example.com",
			shouldError: false,
		},
		{
			name:        "Not Found",
			identifier:  "notfound",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := repo.GetByUsernameOrEmail(ctx, tc.identifier)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestUserRepository_Update tests user update
func TestUserRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "updateuser",
		Email:    "update@example.com",
		Password: "hashed",
		FullName: "Original Name",
		Status:   "active",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Update user
	user.FullName = "Updated Name"
	user.Status = "inactive"

	err = repo.Update(ctx, user)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.FullName)
	assert.Equal(t, "inactive", updated.Status)
}

// TestUserRepository_UpdateFields tests partial field updates
func TestUserRepository_UpdateFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "fieldsuser",
		Email:    "fields@example.com",
		Password: "hashed",
		Status:   "active",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		id          uuid.UUID
		fields      map[string]interface{}
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Update Status",
			id:   user.ID,
			fields: map[string]interface{}{
				"status": "suspended",
			},
			shouldError: false,
		},
		{
			name: "Update Multiple Fields",
			id:   user.ID,
			fields: map[string]interface{}{
				"full_name":      "New Name",
				"email_verified": true,
			},
			shouldError: false,
		},
		{
			name: "Non-existent User",
			id:   uuid.New(),
			fields: map[string]interface{}{
				"status": "active",
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
			}
		})
	}
}

// TestUserRepository_Delete tests user soft delete
func TestUserRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "deleteuser",
		Email:    "delete@example.com",
		Password: "hashed",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Delete user
	err = repo.Delete(ctx, user.ID)
	assert.NoError(t, err)

	// Verify soft delete
	_, err = repo.GetByID(ctx, user.ID)
	assert.Error(t, err)

	// Delete non-existent user
	err = repo.Delete(ctx, uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestUserRepository_ListUsers tests user listing with filters
func TestUserRepository_ListUsers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test users
	users := []*models.User{
		{
			Username: "active1",
			Email:    "active1@example.com",
			Password: "hashed",
			FullName: "Active User One",
			Status:   "active",
		},
		{
			Username: "active2",
			Email:    "active2@example.com",
			Password: "hashed",
			FullName: "Active User Two",
			Status:   "active",
		},
		{
			Username: "inactive1",
			Email:    "inactive1@example.com",
			Password: "hashed",
			FullName: "Inactive User",
			Status:   "inactive",
		},
	}

	for _, u := range users {
		err := repo.Create(ctx, u)
		require.NoError(t, err)
	}

	testCases := []struct {
		name          string
		filter        *ListUsersFilter
		expectedCount int
	}{
		{
			name:          "No Filter",
			filter:        &ListUsersFilter{},
			expectedCount: 3,
		},
		{
			name: "Filter by Active Status",
			filter: &ListUsersFilter{
				Status: "active",
			},
			expectedCount: 2,
		},
		{
			name: "Filter by Inactive Status",
			filter: &ListUsersFilter{
				Status: "inactive",
			},
			expectedCount: 1,
		},
		{
			name: "Search by Username",
			filter: &ListUsersFilter{
				Search: "active1",
			},
			expectedCount: 1,
		},
		{
			name: "Search by Full Name",
			filter: &ListUsersFilter{
				Search: "Active User",
			},
			expectedCount: 2,
		},
		{
			name: "With Pagination",
			filter: &ListUsersFilter{
				Page:     1,
				PageSize: 2,
			},
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, total, err := repo.ListUsers(ctx, tc.filter)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedCount, len(results))
			if tc.filter.Page == 0 {
				assert.Equal(t, int64(3), total)
			}
		})
	}
}

// TestUserRepository_CountByStatus tests counting by status
func TestUserRepository_CountByStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test users
	for i := 0; i < 5; i++ {
		user := &models.User{
			Username: "countuser" + string(rune(i+'0')),
			Email:    "count" + string(rune(i+'0')) + "@example.com",
			Password: "hashed",
			Status:   "active",
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	count, err := repo.CountByStatus(ctx, "active")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)

	count, err = repo.CountByStatus(ctx, "inactive")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// TestUserRepository_ExistsUsername tests username existence check
func TestUserRepository_ExistsUsername(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "existsuser",
		Email:    "exists@example.com",
		Password: "hashed",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Test existence
	exists, err := repo.ExistsUsername(ctx, "existsuser", nil)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test non-existence
	exists, err = repo.ExistsUsername(ctx, "nonexistent", nil)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test with exclude ID
	exists, err = repo.ExistsUsername(ctx, "existsuser", &user.ID)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestUserRepository_ExistsEmail tests email existence check
func TestUserRepository_ExistsEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "emailexists",
		Email:    "emailexists@example.com",
		Password: "hashed",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Test existence
	exists, err := repo.ExistsEmail(ctx, "emailexists@example.com", nil)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test non-existence
	exists, err = repo.ExistsEmail(ctx, "none@example.com", nil)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test with exclude ID
	exists, err = repo.ExistsEmail(ctx, "emailexists@example.com", &user.ID)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestUserRepository_UpdateLastLogin tests last login update
func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "loginuser",
		Email:    "login@example.com",
		Password: "hashed",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Initially no last login
	assert.Nil(t, user.LastLoginAt)

	// Update last login
	err = repo.UpdateLastLogin(ctx, user.ID)
	assert.NoError(t, err)

	// Verify last login updated
	updated, err := repo.GetByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updated.LastLoginAt)
}

// TestUserRepository_UpdateStatus tests status update
func TestUserRepository_UpdateStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &models.User{
		Username: "statususer",
		Email:    "status@example.com",
		Password: "hashed",
		Status:   "active",
	}
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		id          uuid.UUID
		status      string
		shouldError bool
	}{
		{
			name:        "Update to Inactive",
			id:          user.ID,
			status:      "inactive",
			shouldError: false,
		},
		{
			name:        "Update to Suspended",
			id:          user.ID,
			status:      "suspended",
			shouldError: false,
		},
		{
			name:        "Non-existent User",
			id:          uuid.New(),
			status:      "active",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.UpdateStatus(ctx, tc.id, tc.status)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify status updated
				updated, err := repo.GetByID(ctx, tc.id)
				assert.NoError(t, err)
				assert.Equal(t, tc.status, updated.Status)
			}
		})
	}
}
