package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/pkg/dbdriver"

	dbrepo "github.com/ysicing/tiga/internal/repository/database"
)

// UserService manages database-level user accounts.
type UserService struct {
	manager  *DatabaseManager
	userRepo *dbrepo.UserRepository
}

// NewUserService constructs a UserService.
func NewUserService(manager *DatabaseManager, userRepo *dbrepo.UserRepository) *UserService {
	return &UserService{
		manager:  manager,
		userRepo: userRepo,
	}
}

// CreateUserInput captures parameters to create a new database user.
type CreateUserInput struct {
	Username    string
	Password    string
	Host        string
	Description string
	Roles       []string
}

// CreateUser provisions a database user remotely and stores metadata.
func (s *UserService) CreateUser(ctx context.Context, instanceID uuid.UUID, input CreateUserInput) (*models.DatabaseUser, error) {
	if strings.TrimSpace(input.Username) == "" {
		return nil, fmt.Errorf("username is required")
	}
	if input.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	host := input.Host
	if host == "" {
		host = "%"
	}

	driver, instance, err := s.manager.GetConnectedDriver(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	if normalizeDriverType(instance.Type) == "redis" {
		return nil, ErrOperationNotSupported
	}

	roles := input.Roles
	if len(roles) == 0 {
		roles = []string{"readonly"}
	}

	opts := dbdriver.CreateUserOptions{
		Username: input.Username,
		Password: input.Password,
		Host:     host,
		Roles:    roles,
	}

	if err := driver.CreateUser(ctx, opts); err != nil {
		return nil, err
	}

	encrypted, err := encryptSecret(input.Password)
	if err != nil {
		return nil, err
	}

	record := &models.DatabaseUser{
		InstanceID:  instanceID,
		Username:    input.Username,
		Password:    encrypted,
		Host:        host,
		Description: input.Description,
	}

	if err := s.userRepo.Create(ctx, record); err != nil {
		return nil, err
	}

	record.Password = ""
	return record, nil
}

// GetUser returns metadata for a specific database user.
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*models.DatabaseUser, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

// ListUsers merges remote metadata with stored records.
func (s *UserService) ListUsers(ctx context.Context, instanceID uuid.UUID) ([]*models.DatabaseUser, error) {
	driver, instance, err := s.manager.GetConnectedDriver(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	if normalizeDriverType(instance.Type) == "redis" {
		return nil, ErrOperationNotSupported
	}

	remote, err := driver.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	localList, err := s.userRepo.ListByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	localMap := make(map[string]*models.DatabaseUser, len(localList))
	for _, user := range localList {
		localMap[user.Username] = user
	}

	results := make([]*models.DatabaseUser, 0, len(remote)+len(localList))
	for _, info := range remote {
		if existing, ok := localMap[info.Username]; ok {
			if info.Host != "" {
				existing.Host = info.Host
			}
			results = append(results, existing)
			delete(localMap, info.Username)
			continue
		}

		results = append(results, &models.DatabaseUser{
			InstanceID: instanceID,
			Username:   info.Username,
			Host:       info.Host,
		})
	}

	for _, leftover := range localMap {
		results = append(results, leftover)
	}

	for _, user := range results {
		user.Password = ""
	}

	return results, nil
}

// UpdatePassword updates credentials and persists the encrypted value.
func (s *UserService) UpdatePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	if newPassword == "" {
		return fmt.Errorf("new password is required")
	}

	record, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	currentPassword, err := decryptSecret(record.Password)
	if err != nil {
		return err
	}
	if currentPassword != oldPassword {
		return fmt.Errorf("current password does not match")
	}

	driver, instance, err := s.manager.GetConnectedDriver(ctx, record.InstanceID)
	if err != nil {
		return err
	}

	if normalizeDriverType(instance.Type) == "redis" {
		return ErrOperationNotSupported
	}

	opts := map[string]interface{}{
		"host": record.Host,
	}

	if err := driver.UpdateUserPassword(ctx, record.Username, newPassword, opts); err != nil {
		return err
	}

	encrypted, err := encryptSecret(newPassword)
	if err != nil {
		return err
	}

	record.Password = encrypted
	return s.userRepo.Update(ctx, record)
}

// DeleteUser removes a user remotely and locally.
func (s *UserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	record, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	driver, instance, err := s.manager.GetConnectedDriver(ctx, record.InstanceID)
	if err != nil {
		return err
	}

	if normalizeDriverType(instance.Type) == "redis" {
		return ErrOperationNotSupported
	}

	opts := map[string]interface{}{
		"host": record.Host,
	}
	if err := driver.DeleteUser(ctx, record.Username, opts); err != nil {
		return err
	}

	return s.userRepo.Delete(ctx, userID)
}
