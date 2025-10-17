package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// ClusterRepository handles cluster data operations (Phase 0 更新)
type ClusterRepository struct {
	db *gorm.DB
}

// NewClusterRepository creates a new cluster repository
func NewClusterRepository(db *gorm.DB) *ClusterRepository {
	return &ClusterRepository{db: db}
}

// Create creates a new cluster
func (r *ClusterRepository) Create(ctx context.Context, cluster *models.Cluster) error {
	return r.db.WithContext(ctx).Create(cluster).Error
}

// GetByID retrieves a cluster by ID
func (r *ClusterRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Cluster, error) {
	var cluster models.Cluster
	if err := r.db.WithContext(ctx).First(&cluster, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

// GetByName retrieves a cluster by name
func (r *ClusterRepository) GetByName(ctx context.Context, name string) (*models.Cluster, error) {
	var cluster models.Cluster
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&cluster).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

// Update updates specific fields of a cluster
func (r *ClusterRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Cluster{}).Where("id = ?", id).Updates(updates).Error
}

// Delete soft deletes a cluster
func (r *ClusterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Cluster{}, "id = ?", id).Error
}

// List retrieves all clusters
func (r *ClusterRepository) List(ctx context.Context) ([]*models.Cluster, error) {
	var clusters []*models.Cluster
	if err := r.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

// GetAllEnabled retrieves all enabled clusters (Phase 0 新增)
func (r *ClusterRepository) GetAllEnabled(ctx context.Context) ([]*models.Cluster, error) {
	var clusters []*models.Cluster
	if err := r.db.WithContext(ctx).Where("enable = ?", true).Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

// GetDefault retrieves the default cluster (Phase 0 新增)
func (r *ClusterRepository) GetDefault(ctx context.Context) (*models.Cluster, error) {
	var cluster models.Cluster
	if err := r.db.WithContext(ctx).Where("is_default = ?", true).First(&cluster).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

// SetDefault sets a cluster as default (Phase 0 更新 - 添加 context)
func (r *ClusterRepository) SetDefault(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear existing default
		if err := tx.Model(&models.Cluster{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		// Set new default
		return tx.Model(&models.Cluster{}).Where("id = ?", id).Update("is_default", true).Error
	})
}

// Count counts all clusters
func (r *ClusterRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Cluster{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ClearDefault clears the default flag from all clusters
func (r *ClusterRepository) ClearDefault(ctx context.Context) error {
	return r.db.WithContext(ctx).Model(&models.Cluster{}).Where("is_default = ?", true).Update("is_default", false).Error
}

// Enable enables a cluster
func (r *ClusterRepository) Enable(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Cluster{}).Where("id = ?", id).Update("enable", true).Error
}

// Disable disables a cluster
func (r *ClusterRepository) Disable(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Cluster{}).Where("id = ?", id).Update("enable", false).Error
}

// ResourceHistoryRepository handles resource history data operations
type ResourceHistoryRepository struct {
	db *gorm.DB
}

// NewResourceHistoryRepository creates a new resource history repository
func NewResourceHistoryRepository(db *gorm.DB) *ResourceHistoryRepository {
	return &ResourceHistoryRepository{db: db}
}

// Create creates a new resource history record
func (r *ResourceHistoryRepository) Create(history *models.ResourceHistory) error {
	return r.db.Create(history).Error
}

// GetByID retrieves a resource history by ID
func (r *ResourceHistoryRepository) GetByID(id uuid.UUID) (*models.ResourceHistory, error) {
	var history models.ResourceHistory
	if err := r.db.Preload("Operator").First(&history, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

// ListByResource retrieves resource history by resource identification
func (r *ResourceHistoryRepository) ListByResource(clusterName, resourceType, resourceName, namespace string, limit int) ([]*models.ResourceHistory, error) {
	var histories []*models.ResourceHistory
	query := r.db.Where("cluster_name = ? AND resource_type = ? AND resource_name = ?", clusterName, resourceType, resourceName)
	if namespace != "" {
		query = query.Where("namespace = ?", namespace)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Order("created_at DESC").Preload("Operator").Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}

// ListByCluster retrieves resource history by cluster
func (r *ResourceHistoryRepository) ListByCluster(clusterName string, limit int) ([]*models.ResourceHistory, error) {
	var histories []*models.ResourceHistory
	query := r.db.Where("cluster_name = ?", clusterName)
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Order("created_at DESC").Preload("Operator").Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}

// Delete soft deletes a resource history record
func (r *ResourceHistoryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.ResourceHistory{}, "id = ?", id).Error
}

// DeleteOlderThan deletes resource history records older than a certain time
func (r *ResourceHistoryRepository) DeleteOlderThan(days int) error {
	query := fmt.Sprintf("created_at < datetime('now', '-%d days')", days)
	return r.db.Where(query).Delete(&models.ResourceHistory{}).Error
}
