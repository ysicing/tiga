package managers

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/ysicing/tiga/internal/models"
)

// MinIOManager manages MinIO instances
type MinIOManager struct {
	*BaseManager
	client *minio.Client
}

// NewMinIOManager creates a new MinIO manager
func NewMinIOManager() *MinIOManager {
	return &MinIOManager{
		BaseManager: NewBaseManager(),
	}
}

// Initialize initializes the MinIO manager
func (m *MinIOManager) Initialize(ctx context.Context, instance *models.Instance) error {
	if err := m.BaseManager.Initialize(ctx, instance); err != nil {
		return err
	}
	return nil
}

// Connect establishes connection to MinIO
func (m *MinIOManager) Connect(ctx context.Context) error {
	connection := m.instance.Connection
	host, _ := connection["host"].(string)
	port, _ := connection["port"].(float64)

	endpoint := fmt.Sprintf("%s:%d", host, int(port))

	accessKey := m.GetConfigValue("access_key", "").(string)
	secretKey := m.GetConfigValue("secret_key", "").(string)
	useSSL := m.GetConfigValue("use_ssl", false).(bool)

	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("%w: access_key and secret_key required", ErrInvalidConfig)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	m.client = client
	return nil
}

// Disconnect closes connection to MinIO
func (m *MinIOManager) Disconnect(ctx context.Context) error {
	m.client = nil
	return nil
}

// HealthCheck checks MinIO health
func (m *MinIOManager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	start := time.Now()
	status := &HealthStatus{
		Healthy:       false,
		LastCheckTime: time.Now().Format(time.RFC3339),
		Details:       make(map[string]interface{}),
	}

	// Try to list buckets as health check
	buckets, err := m.client.ListBuckets(ctx)
	if err != nil {
		status.Message = fmt.Sprintf("Health check failed: %v", err)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status, nil
	}

	status.Healthy = true
	status.Message = "MinIO is healthy"
	status.ResponseTime = time.Since(start).Milliseconds()
	status.Details["bucket_count"] = len(buckets)

	return status, nil
}

// CollectMetrics collects metrics from MinIO
func (m *MinIOManager) CollectMetrics(ctx context.Context) (*ServiceMetrics, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	metrics := &ServiceMetrics{
		InstanceID: m.instance.ID,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metrics:    make(map[string]interface{}),
	}

	// List buckets
	buckets, err := m.client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricsCollectionFailed, err)
	}

	metrics.Metrics["bucket_count"] = len(buckets)

	// Collect bucket sizes
	var totalSize uint64
	bucketSizes := make(map[string]uint64)

	for _, bucket := range buckets {
		objectCh := m.client.ListObjects(ctx, bucket.Name, minio.ListObjectsOptions{
			Recursive: true,
		})

		var bucketSize uint64
		for object := range objectCh {
			if object.Err != nil {
				continue
			}
			bucketSize += uint64(object.Size)
		}

		bucketSizes[bucket.Name] = bucketSize
		totalSize += bucketSize
	}

	metrics.Metrics["total_size_bytes"] = totalSize
	metrics.Metrics["bucket_sizes"] = bucketSizes

	return metrics, nil
}

// GetInfo returns MinIO service information
func (m *MinIOManager) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	info := make(map[string]interface{})

	buckets, err := m.client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	bucketList := make([]map[string]interface{}, 0, len(buckets))
	for _, bucket := range buckets {
		bucketList = append(bucketList, map[string]interface{}{
			"name":       bucket.Name,
			"created_at": bucket.CreationDate.Format(time.RFC3339),
		})
	}

	info["buckets"] = bucketList
	info["bucket_count"] = len(buckets)

	return info, nil
}

// ValidateConfig validates MinIO configuration
func (m *MinIOManager) ValidateConfig(config map[string]interface{}) error {
	accessKey, ok := config["access_key"].(string)
	if !ok || accessKey == "" {
		return fmt.Errorf("%w: access_key is required", ErrInvalidConfig)
	}

	secretKey, ok := config["secret_key"].(string)
	if !ok || secretKey == "" {
		return fmt.Errorf("%w: secret_key is required", ErrInvalidConfig)
	}

	return nil
}

// Type returns the service type
func (m *MinIOManager) Type() string {
	return "minio"
}

// GetClient returns the MinIO client for direct operations
func (m *MinIOManager) GetClient() *minio.Client {
	return m.client
}

// getAdminClient creates a MinIO admin client for user and policy management
func (m *MinIOManager) getAdminClient() (*madmin.AdminClient, error) {
	connection := m.instance.Connection
	host, _ := connection["host"].(string)
	port, _ := connection["port"].(float64)

	endpoint := fmt.Sprintf("%s:%d", host, int(port))

	accessKey := m.GetConfigValue("access_key", "").(string)
	secretKey := m.GetConfigValue("secret_key", "").(string)
	useSSL := m.GetConfigValue("use_ssl", false).(bool)

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("%w: access_key and secret_key required", ErrInvalidConfig)
	}

	// Create admin client
	adminClient, err := madmin.New(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin client: %w", err)
	}

	return adminClient, nil
}
