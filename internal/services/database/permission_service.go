package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	dbrepo "github.com/ysicing/tiga/internal/repository/database"
	"github.com/ysicing/tiga/pkg/dbdriver"
)

// PermissionService manages database permission policies.
type PermissionService struct {
	manager        *DatabaseManager
	userRepo       *dbrepo.UserRepository
	databaseRepo   *dbrepo.DatabaseRepository
	permissionRepo *dbrepo.PermissionRepository
}

// NewPermissionService constructs a PermissionService.
func NewPermissionService(
	manager *DatabaseManager,
	userRepo *dbrepo.UserRepository,
	databaseRepo *dbrepo.DatabaseRepository,
	permissionRepo *dbrepo.PermissionRepository,
) *PermissionService {
	return &PermissionService{
		manager:        manager,
		userRepo:       userRepo,
		databaseRepo:   databaseRepo,
		permissionRepo: permissionRepo,
	}
}

// GrantPermission grants database level access to a user.
func (s *PermissionService) GrantPermission(ctx context.Context, userID, databaseID uuid.UUID, role, grantedBy string) (*models.PermissionPolicy, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "readonly" && role != "readwrite" {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	database, err := s.databaseRepo.GetByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	if user.InstanceID != database.InstanceID {
		return nil, fmt.Errorf("user and database belong to different instances")
	}

	exists, err := s.permissionRepo.CheckExisting(ctx, userID, databaseID, role)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("permission already exists")
	}

	driver, instance, err := s.manager.GetConnectedDriver(ctx, user.InstanceID)
	if err != nil {
		return nil, err
	}

	if normalizeDriverType(instance.Type) == "redis" {
		return nil, ErrOperationNotSupported
	}

	if err := s.applyGrant(ctx, driver, instance.Type, user, database, role); err != nil {
		return nil, err
	}

	policy := &models.PermissionPolicy{
		UserID:     userID,
		DatabaseID: databaseID,
		Role:       role,
		GrantedBy:  grantedBy,
		GrantedAt:  time.Now().UTC(),
	}

	if err := s.permissionRepo.Grant(ctx, policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// RevokePermission revokes an existing permission.
func (s *PermissionService) RevokePermission(ctx context.Context, permissionID uuid.UUID) error {
	policy, err := s.permissionRepo.GetByID(ctx, permissionID)
	if err != nil {
		return err
	}
	if policy.RevokedAt != nil {
		return fmt.Errorf("permission already revoked")
	}

	user := policy.User
	if user == nil {
		user, err = s.userRepo.GetByID(ctx, policy.UserID)
		if err != nil {
			return err
		}
		policy.User = user
	}

	database := policy.Database
	if database == nil {
		database, err = s.databaseRepo.GetByID(ctx, policy.DatabaseID)
		if err != nil {
			return err
		}
		policy.Database = database
	}

	driver, instance, err := s.manager.GetConnectedDriver(ctx, user.InstanceID)
	if err != nil {
		return err
	}

	if normalizeDriverType(instance.Type) == "redis" {
		return ErrOperationNotSupported
	}

	if err := s.applyRevoke(ctx, driver, instance.Type, user, database); err != nil {
		return err
	}

	return s.permissionRepo.Revoke(ctx, permissionID)
}

// GetUserPermissions retrieves active permissions for a user.
func (s *PermissionService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]*models.PermissionPolicy, error) {
	return s.permissionRepo.ListByUser(ctx, userID)
}

func (s *PermissionService) applyGrant(ctx context.Context, driver dbdriver.DatabaseDriver, instanceType string, user *models.DatabaseUser, database *models.Database, role string) error {
	switch normalizeDriverType(instanceType) {
	case "mysql":
		privileges := "SELECT"
		if role == "readwrite" {
			privileges = "ALL PRIVILEGES"
		}
		statements := []string{
			fmt.Sprintf("GRANT %s ON %s.* TO '%s'@'%s'", privileges, quoteIdentifier(database.Name), escapeSingleQuotes(user.Username), escapeSingleQuotes(user.Host)),
			"FLUSH PRIVILEGES",
		}
		return executeStatements(ctx, driver, statements)
	case "postgresql":
		username := quotePGIdentifier(user.Username)
		dbName := quotePGIdentifier(database.Name)
		var statements []string
		if role == "readwrite" {
			statements = []string{
				fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", dbName, username),
				fmt.Sprintf("GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %s", username),
			}
		} else {
			statements = []string{
				fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", dbName, username),
				fmt.Sprintf("GRANT USAGE ON SCHEMA public TO %s", username),
				fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA public TO %s", username),
			}
		}
		return executeStatements(ctx, driver, statements)
	case "redis":
		password, err := decryptSecret(user.Password)
		if err != nil {
			return err
		}
		opts := map[string]interface{}{
			"roles": []string{role},
		}
		return driver.UpdateUserPassword(ctx, user.Username, password, opts)
	default:
		return fmt.Errorf("permissions not supported for instance type: %s", instanceType)
	}
}

func (s *PermissionService) applyRevoke(ctx context.Context, driver dbdriver.DatabaseDriver, instanceType string, user *models.DatabaseUser, database *models.Database) error {
	switch normalizeDriverType(instanceType) {
	case "mysql":
		statements := []string{
			fmt.Sprintf("REVOKE ALL PRIVILEGES, GRANT OPTION FROM '%s'@'%s'", escapeSingleQuotes(user.Username), escapeSingleQuotes(user.Host)),
			"FLUSH PRIVILEGES",
		}
		return executeStatements(ctx, driver, statements)
	case "postgresql":
		username := quotePGIdentifier(user.Username)
		dbName := quotePGIdentifier(database.Name)
		statements := []string{
			fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM %s", dbName, username),
			fmt.Sprintf("REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM %s", username),
		}
		return executeStatements(ctx, driver, statements)
	case "redis":
		password, err := decryptSecret(user.Password)
		if err != nil {
			return err
		}
		opts := map[string]interface{}{
			"roles": []string{"none"},
		}
		return driver.UpdateUserPassword(ctx, user.Username, password, opts)
	default:
		return fmt.Errorf("permissions not supported for instance type: %s", instanceType)
	}
}

func executeStatements(ctx context.Context, driver dbdriver.DatabaseDriver, statements []string) error {
	for _, stmt := range statements {
		req := dbdriver.QueryRequest{Query: stmt}
		if _, err := driver.ExecuteQuery(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func escapeSingleQuotes(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func quotePGIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
