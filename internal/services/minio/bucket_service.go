package minio

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/services/managers"

	sdkminio "github.com/minio/minio-go/v7"
	coreRepo "github.com/ysicing/tiga/internal/repository"
)

type BucketService struct {
	instanceRepo *coreRepo.InstanceRepository
}

func NewBucketService(inst *coreRepo.InstanceRepository) *BucketService {
	return &BucketService{instanceRepo: inst}
}

//

// helper to get manager from instance ID
func (s *BucketService) manager(ctx context.Context, instanceID uuid.UUID) (*managers.MinIOManager, error) {
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
	return m, nil
}

func (s *BucketService) ListBuckets(ctx context.Context, instanceID uuid.UUID) ([]sdkminio.BucketInfo, error) {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	defer m.Disconnect(ctx)
	return m.GetClient().ListBuckets(ctx)
}

var bucketNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9\-\.]{1,61}[a-z0-9]$`)

func (s *BucketService) ValidateBucketName(name string) error {
	if !bucketNameRe.MatchString(name) {
		return fmt.Errorf("invalid bucket name")
	}
	return nil
}

func (s *BucketService) CreateBucket(ctx context.Context, instanceID uuid.UUID, name, location string) error {
	if err := s.ValidateBucketName(name); err != nil {
		return err
	}
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return err
	}
	defer m.Disconnect(ctx)
	return m.GetClient().MakeBucket(ctx, name, sdkminio.MakeBucketOptions{Region: location})
}

func (s *BucketService) DeleteBucket(ctx context.Context, instanceID uuid.UUID, name string) error {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return err
	}
	defer m.Disconnect(ctx)
	return m.GetClient().RemoveBucket(ctx, name)
}

func (s *BucketService) UpdateBucketPolicy(ctx context.Context, instanceID uuid.UUID, name string, policyJSON string) error {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return err
	}
	defer m.Disconnect(ctx)
	return m.GetClient().SetBucketPolicy(ctx, name, policyJSON)
}

type BucketInfo struct {
	Name        string `json:"name"`
	ObjectCount int    `json:"object_count"`
	TotalSize   int64  `json:"total_size"`
	CreatedAt   string `json:"created_at"`
}

func (s *BucketService) GetBucketInfo(ctx context.Context, instanceID uuid.UUID, name string) (*BucketInfo, error) {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	defer m.Disconnect(ctx)
	buckets, _ := m.GetClient().ListBuckets(ctx)
	bi := &BucketInfo{Name: name}
	for _, b := range buckets {
		if b.Name == name {
			bi.CreatedAt = b.CreationDate.Format("2006-01-02T15:04:05Z07:00")
		}
	}
	// compute size/count
	var total int64
	var count int
	ch := m.GetClient().ListObjects(ctx, name, sdkminio.ListObjectsOptions{Recursive: true})
	for obj := range ch {
		if obj.Err != nil {
			continue
		}
		total += obj.Size
		count++
	}
	bi.ObjectCount = count
	bi.TotalSize = total
	return bi, nil
}
