package minio

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	sdkminio "github.com/minio/minio-go/v7"

	coreRepo "github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

type FileService struct {
	instanceRepo *coreRepo.InstanceRepository
}

func NewFileService(inst *coreRepo.InstanceRepository) *FileService {
	return &FileService{instanceRepo: inst}
}

// helper
func (s *FileService) manager(ctx context.Context, instanceID uuid.UUID) (*managers.MinIOManager, error) {
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

type ObjectItem struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ETag         string    `json:"etag"`
	LastModified time.Time `json:"last_modified"`
	StorageClass string    `json:"storage_class"`
}

func (s *FileService) ListObjects(ctx context.Context, instanceID uuid.UUID, bucket, prefix string, recursive bool) ([]ObjectItem, error) {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	defer m.Disconnect(ctx)
	ch := m.GetClient().ListObjects(ctx, bucket, sdkminio.ListObjectsOptions{Prefix: prefix, Recursive: recursive})
	var out []ObjectItem
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out = append(out, ObjectItem{Key: obj.Key, Size: obj.Size, ETag: obj.ETag, LastModified: obj.LastModified, StorageClass: obj.StorageClass})
	}
	return out, nil
}

func (s *FileService) Upload(ctx context.Context, instanceID uuid.UUID, bucket, key string, r io.Reader, size int64, contentType string) (*sdkminio.UploadInfo, error) {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	defer m.Disconnect(ctx)
	info, err := m.GetClient().PutObject(ctx, bucket, key, r, size, sdkminio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *FileService) Delete(ctx context.Context, instanceID uuid.UUID, bucket, key string) error {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return err
	}
	defer m.Disconnect(ctx)
	return m.GetClient().RemoveObject(ctx, bucket, key, sdkminio.RemoveObjectOptions{})
}

func (s *FileService) DeleteBatch(ctx context.Context, instanceID uuid.UUID, bucket string, keys []string) error {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return err
	}
	defer m.Disconnect(ctx)
	for _, k := range keys {
		if err := m.GetClient().RemoveObject(ctx, bucket, k, sdkminio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileService) PresignedDownload(ctx context.Context, instanceID uuid.UUID, bucket, key string, expiry time.Duration) (string, error) {
	m, err := s.manager(ctx, instanceID)
	if err != nil {
		return "", err
	}
	defer m.Disconnect(ctx)
	u, err := m.GetClient().PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
