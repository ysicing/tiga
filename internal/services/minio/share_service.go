package minio

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/managers"

	coreRepo "github.com/ysicing/tiga/internal/repository"
	mrepo "github.com/ysicing/tiga/internal/repository/minio"
)

type ShareService struct {
	instanceRepo *coreRepo.InstanceRepository
	shareRepo    *mrepo.ShareRepository
}

func NewShareService(inst *coreRepo.InstanceRepository, repo *mrepo.ShareRepository) *ShareService {
	return &ShareService{instanceRepo: inst, shareRepo: repo}
}

func (s *ShareService) CreateShareLink(ctx context.Context, instanceID uuid.UUID, bucket, key string, expiry time.Duration, createdBy *uuid.UUID) (*models.MinIOShareLink, string, error) {
	inst, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, "", err
	}
	m := managers.NewMinIOManager()
	if err := m.Initialize(ctx, inst); err != nil {
		return nil, "", err
	}
	if err := m.Connect(ctx); err != nil {
		return nil, "", err
	}
	defer m.Disconnect(ctx)

	u, err := m.GetClient().PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return nil, "", err
	}

	token := randomToken(12)
	link := &models.MinIOShareLink{InstanceID: instanceID, BucketName: bucket, ObjectKey: key, Token: token, ExpiresAt: time.Now().Add(expiry), Status: "active"}
	if createdBy != nil {
		link.CreatedBy = createdBy
	}
	_ = s.shareRepo.Create(ctx, link)
	return link, u.String(), nil
}

func (s *ShareService) ListShares(ctx context.Context, instanceID uuid.UUID, createdBy *uuid.UUID) ([]*models.MinIOShareLink, error) {
	return s.shareRepo.List(ctx, instanceID, createdBy)
}

func (s *ShareService) RevokeShare(ctx context.Context, id uuid.UUID) error {
	return s.shareRepo.Revoke(ctx, id)
}
func (s *ShareService) CleanupExpiredShares(ctx context.Context) error {
	return s.shareRepo.DeleteExpired(ctx)
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
