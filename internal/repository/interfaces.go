package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
)

// UserRepositoryInterface defines the interface for user data operations
type UserRepositoryInterface interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, filter *ListUsersFilter) ([]*models.User, int64, error)
	ExistsUsername(ctx context.Context, username string, excludeID *uuid.UUID) (bool, error)
	ExistsEmail(ctx context.Context, email string, excludeID *uuid.UUID) (bool, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
}

// InstanceRepositoryInterface defines the interface for instance data operations
type InstanceRepositoryInterface interface {
	Create(ctx context.Context, instance *models.Instance) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Instance, error)
	GetByName(ctx context.Context, name string) (*models.Instance, error)
	Update(ctx context.Context, instance *models.Instance) error
	UpdateFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	ListInstances(ctx context.Context, filter *ListInstancesFilter) ([]*models.Instance, int64, error)
	ListByServiceType(ctx context.Context, serviceType string) ([]*models.Instance, error)
	ListByStatus(ctx context.Context, status string) ([]*models.Instance, error)
	CountByServiceType(ctx context.Context, serviceType string) (int64, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	ExistsName(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateHealth(ctx context.Context, id uuid.UUID, healthStatus string, healthMessage *string) error
	UpdateVersion(ctx context.Context, id uuid.UUID, version string) error
	AddTags(ctx context.Context, id uuid.UUID, tags []string) error
	RemoveTags(ctx context.Context, id uuid.UUID, tags []string) error
	SearchByTag(ctx context.Context, tags []string) ([]*models.Instance, error)
	GetStatistics(ctx context.Context) (*InstanceStatistics, error)
}

// AlertRepositoryInterface defines the interface for alert data operations
type AlertRepositoryInterface interface {
	// Alert Rule Methods
	CreateRule(ctx context.Context, rule *models.Alert) error
	GetRuleByID(ctx context.Context, id uuid.UUID) (*models.Alert, error)
	GetRuleByName(ctx context.Context, name string) (*models.Alert, error)
	UpdateRule(ctx context.Context, rule *models.Alert) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
	ListRules(ctx context.Context, filter *ListRulesFilter) ([]*models.Alert, int64, error)
	ListEnabledRules(ctx context.Context) ([]*models.Alert, error)
	ListRulesByInstance(ctx context.Context, instanceID uuid.UUID) ([]*models.Alert, error)
	ToggleRule(ctx context.Context, id uuid.UUID, enabled bool) error

	// Alert Event Methods
	CreateEvent(ctx context.Context, event *models.AlertEvent) error
	GetEventByID(ctx context.Context, id uuid.UUID) (*models.AlertEvent, error)
	UpdateEvent(ctx context.Context, event *models.AlertEvent) error
	DeleteEvent(ctx context.Context, id uuid.UUID) error
	ListEvents(ctx context.Context, filter *ListEventsFilter) ([]*models.AlertEvent, int64, error)
	ListActiveEvents(ctx context.Context) ([]*models.AlertEvent, error)
	ListEventsByInstance(ctx context.Context, instanceID uuid.UUID) ([]*models.AlertEvent, error)
	AcknowledgeEvent(ctx context.Context, id uuid.UUID, acknowledgedBy uuid.UUID, note string) error
	ResolveEvent(ctx context.Context, id uuid.UUID) error
	CountEventsByStatus(ctx context.Context, status string) (int64, error)
	CountEventsBySeverity(ctx context.Context, severity string) (int64, error)
	DeleteOldEvents(ctx context.Context, olderThan time.Time) (int64, error)
	GetStatistics(ctx context.Context) (*AlertStatistics, error)
}

// MetricsRepositoryInterface defines the interface for metrics data operations
type MetricsRepositoryInterface interface {
	Create(ctx context.Context, metric *models.Metric) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Metric, error)
	QueryMetrics(ctx context.Context, filter *MetricsQueryFilter) ([]*models.Metric, error)
	GetLatestMetric(ctx context.Context, instanceID uuid.UUID, metricName string) (*models.Metric, error)
	DeleteOldMetrics(ctx context.Context, olderThan time.Time) (int64, error)
	AggregateMetrics(ctx context.Context, instanceID uuid.UUID, metricName string, startTime, endTime time.Time, interval string) ([]*AggregatedMetric, error)
}

// AuditLogRepositoryInterface defines the interface for audit log data operations
type AuditLogRepositoryInterface interface {
	Create(ctx context.Context, log *models.AuditLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error)
	ListAuditLogs(ctx context.Context, filter *ListAuditLogsFilter) ([]*models.AuditLog, int64, error)
	DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error)
	GetStatistics(ctx context.Context) (*AuditLogStatistics, error)
}

// ClusterRepositoryInterface defines the interface for Kubernetes cluster operations
type ClusterRepositoryInterface interface {
	Create(cluster *models.Cluster) error
	GetByID(id uuid.UUID) (*models.Cluster, error)
	GetByName(name string) (*models.Cluster, error)
	Update(cluster *models.Cluster) error
	Delete(id uuid.UUID) error
	List() ([]*models.Cluster, error)
}

// ResourceHistoryRepositoryInterface defines the interface for Kubernetes resource history operations
type ResourceHistoryRepositoryInterface interface {
	Create(history *models.ResourceHistory) error
	GetByID(id uuid.UUID) (*models.ResourceHistory, error)
	ListByResource(clusterName, resourceType, resourceName, namespace string, limit int) ([]*models.ResourceHistory, error)
	Delete(id uuid.UUID) error
}

// OAuthProviderRepositoryInterface defines the interface for OAuth provider operations
type OAuthProviderRepositoryInterface interface {
	Create(ctx context.Context, provider *models.OAuthProvider) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.OAuthProvider, error)
	GetByName(ctx context.Context, name string) (*models.OAuthProvider, error)
	Update(ctx context.Context, provider *models.OAuthProvider) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*models.OAuthProvider, error)
	ListEnabled(ctx context.Context) ([]*models.OAuthProvider, error)
}

// Compile-time interface assertions
var (
	_ UserRepositoryInterface                = (*UserRepository)(nil)
	_ InstanceRepositoryInterface            = (*InstanceRepository)(nil)
	_ AlertRepositoryInterface               = (*AlertRepository)(nil)
	_ MetricsRepositoryInterface             = (*MetricsRepository)(nil)
	_ AuditLogRepositoryInterface            = (*AuditLogRepository)(nil)
	_ ClusterRepositoryInterface             = (*ClusterRepository)(nil)
	_ ResourceHistoryRepositoryInterface     = (*ResourceHistoryRepository)(nil)
	_ OAuthProviderRepositoryInterface       = (*OAuthProviderRepository)(nil)
)
