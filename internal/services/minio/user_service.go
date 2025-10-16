package minio

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/managers"

	coreRepo "github.com/ysicing/tiga/internal/repository"
	repo "github.com/ysicing/tiga/internal/repository/minio"
)

type UserService struct {
	instanceRepo *coreRepo.InstanceRepository
	userRepo     *repo.UserRepository
}

func NewUserService(instanceRepo *coreRepo.InstanceRepository, userRepo *repo.UserRepository) *UserService {
	return &UserService{instanceRepo: instanceRepo, userRepo: userRepo}
}

// CreateUser creates a MinIO user in the server and persists a record
func (s *UserService) CreateUser(ctx context.Context, instanceID uuid.UUID, accessKey, secretKey, description string) (*models.MinIOUser, error) {
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.Type != "minio" {
		return nil, fmt.Errorf("instance is not MinIO type")
	}

	m := managers.NewMinIOManager()
	if err := m.Initialize(ctx, inst); err != nil {
		return nil, err
	}
	if err := m.Connect(ctx); err != nil {
		return nil, err
	}
	defer m.Disconnect(ctx)

	if err := m.CreateUser(ctx, accessKey, secretKey); err != nil {
		return nil, err
	}

	u := &models.MinIOUser{
		InstanceID:  instanceID,
		Username:    accessKey,
		AccessKey:   accessKey,
		SecretKey:   models.SecretString(secretKey),
		Status:      "enabled",
		Description: description,
	}
	_ = s.userRepo.Create(ctx, u) // best effort persist; errors here should not block API
	return u, nil
}

// ListUsers lists users from server (authoritative)
func (s *UserService) ListUsers(ctx context.Context, instanceID uuid.UUID) ([]managers.MinIOUserInfo, error) {
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.Type != "minio" {
		return nil, fmt.Errorf("instance is not MinIO type")
	}
	m := managers.NewMinIOManager()
	if err := m.Initialize(ctx, inst); err != nil {
		return nil, err
	}
	if err := m.Connect(ctx); err != nil {
		return nil, err
	}
	defer m.Disconnect(ctx)
	return m.ListUsers(ctx)
}

// DeleteUser deletes from server and removes record if exists
func (s *UserService) DeleteUser(ctx context.Context, instanceID uuid.UUID, username string) error {
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.Type != "minio" {
		return fmt.Errorf("instance is not MinIO type")
	}
	m := managers.NewMinIOManager()
	if err := m.Initialize(ctx, inst); err != nil {
		return err
	}
	if err := m.Connect(ctx); err != nil {
		return err
	}
	defer m.Disconnect(ctx)
	if err := m.DeleteUser(ctx, username); err != nil {
		return err
	}
	// Try delete DB record
	if u, err := s.userRepo.GetByUsername(ctx, instanceID, username); err == nil {
		_ = s.userRepo.Delete(ctx, u.ID)
	}
	return nil
}
