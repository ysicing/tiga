package minio

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/managers"

	coreRepo "github.com/ysicing/tiga/internal/repository"
	mrepo "github.com/ysicing/tiga/internal/repository/minio"
)

type PermissionService struct {
	instanceRepo *coreRepo.InstanceRepository
	userRepo     *mrepo.UserRepository
	permRepo     *mrepo.PermissionRepository
}

func NewPermissionService(inst *coreRepo.InstanceRepository, users *mrepo.UserRepository, perms *mrepo.PermissionRepository) *PermissionService {
	return &PermissionService{instanceRepo: inst, userRepo: users, permRepo: perms}
}

func (s *PermissionService) ValidatePermission(p string) error {
	switch p {
	case "readonly", "writeonly", "readwrite":
		return nil
	default:
		return fmt.Errorf("invalid permission: %s", p)
	}
}

// GrantPermission grants a bucket/prefix permission, attaches policy on MinIO, and persists a permission record
func (s *PermissionService) GrantPermission(ctx context.Context, instanceID uuid.UUID, userAccessKey, bucket, prefix, permission string, grantedBy *uuid.UUID, desc string) (string, error) {
	if err := s.ValidatePermission(permission); err != nil {
		return "", err
	}
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return "", err
	}
	if inst.Type != "minio" {
		return "", fmt.Errorf("instance is not MinIO type")
	}

	// Ensure user record exists in DB (best effort)
	var user *models.MinIOUser
	if u, err := s.userRepo.GetByUsername(ctx, instanceID, userAccessKey); err == nil {
		user = u
	} else {
		u2 := &models.MinIOUser{InstanceID: instanceID, Username: userAccessKey, AccessKey: userAccessKey, Status: "enabled"}
		_ = s.userRepo.Create(ctx, u2)
		user = u2
	}

	// Build policy
	mgr := managers.NewMinIOManager()
	if err := mgr.Initialize(ctx, inst); err != nil {
		return "", err
	}
	if err := mgr.Connect(ctx); err != nil {
		return "", err
	}
	defer mgr.Disconnect(ctx)

	var policy map[string]interface{}
	if prefix == "" {
		policy, err = mgr.GenerateBucketPolicy(bucket, permission)
	} else {
		policy, err = mgr.GeneratePrefixPolicy(bucket, prefix, permission)
	}
	if err != nil {
		return "", err
	}

	pfx := base64.RawURLEncoding.EncodeToString([]byte(prefix))
	policyName := fmt.Sprintf("tiga-%s-%s-%s-%s-%s", inst.ID.String(), userAccessKey, bucket, pfx, permission)

	if err := mgr.CreatePolicy(ctx, policyName, policy); err != nil {
		return "", err
	}
	if err := mgr.AttachUserPolicy(ctx, userAccessKey, policyName); err != nil {
		return "", err
	}

	// Persist permission (best effort)
	perm := &models.BucketPermission{InstanceID: instanceID, UserID: user.ID, BucketName: bucket, Prefix: prefix, Permission: permission, GrantedBy: grantedBy, Description: desc}
	_ = s.permRepo.Create(ctx, perm)

	return policyName, nil
}

// RevokePermission detaches policy from user and removes canned policy
func (s *PermissionService) RevokePermission(ctx context.Context, instanceID uuid.UUID, userAccessKey, policyName string) error {
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.Type != "minio" {
		return fmt.Errorf("instance is not MinIO type")
	}
	mgr := managers.NewMinIOManager()
	if err := mgr.Initialize(ctx, inst); err != nil {
		return err
	}
	if err := mgr.Connect(ctx); err != nil {
		return err
	}
	defer mgr.Disconnect(ctx)

	if err := mgr.DetachUserPolicy(ctx, userAccessKey, policyName); err != nil {
		return err
	}
	_ = mgr.DeletePolicy(ctx, policyName)
	return nil
}

// ListPermissions returns DB-backed permissions for the instance (optionally filter by user/bucket)
func (s *PermissionService) ListPermissions(ctx context.Context, instanceID uuid.UUID, userID *uuid.UUID, bucket *string) ([]*models.BucketPermission, error) {
	return s.permRepo.List(ctx, instanceID, userID, bucket)
}
