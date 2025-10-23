package docker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// AuditLogService handles Docker-specific audit log operations
type AuditLogService struct {
	auditRepo repository.AuditLogRepositoryInterface
}

// NewAuditLogService creates a new AuditLogService
func NewAuditLogService(auditRepo repository.AuditLogRepositoryInterface) *AuditLogService {
	return &AuditLogService{
		auditRepo: auditRepo,
	}
}

// DockerAuditLogFilter represents Docker-specific audit log filters
type DockerAuditLogFilter struct {
	Page         int
	PageSize     int
	InstanceID   string     // Filter by Docker instance ID
	User         string     // Filter by username or user ID
	Action       string     // Filter by Docker action
	ResourceType string     // Filter by Docker resource type
	StartTime    *time.Time // Filter by start time
	EndTime      *time.Time // Filter by end time
	Success      *bool      // Filter by operation result
}

// GetAuditLogs retrieves Docker audit logs with filtering
func (s *AuditLogService) GetAuditLogs(ctx context.Context, filter *DockerAuditLogFilter) ([]*models.AuditLog, int64, error) {
	// Build repository filter
	repoFilter := &repository.ListAuditLogsFilter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	// Filter by Docker resource types if not specified
	if filter.ResourceType != "" {
		// Specific Docker resource type
		repoFilter.ResourceType = filter.ResourceType
	}
	// Note: If no resource_type is specified, we'll return all audit logs
	// The frontend can specify docker_container, docker_image, docker_instance to filter

	// Filter by action
	if filter.Action != "" {
		repoFilter.Action = filter.Action
	}

	// Filter by user (username or user ID)
	if filter.User != "" {
		// Try to parse as UUID
		if userID, err := uuid.Parse(filter.User); err == nil {
			repoFilter.UserID = &userID
		}
		// Note: If not a valid UUID, we can't filter by username directly
		// The repository's ListAuditLogsFilter doesn't support username filtering
		// This is a limitation we'll accept for now
	}

	// Filter by time range
	if filter.StartTime != nil {
		repoFilter.StartTime = *filter.StartTime
	}
	if filter.EndTime != nil {
		repoFilter.EndTime = *filter.EndTime
	}

	// Filter by status (success/failure)
	if filter.Success != nil {
		if *filter.Success {
			repoFilter.Status = "success"
		} else {
			repoFilter.Status = "failure"
		}
	}

	// Query audit logs
	logs, total, err := s.auditRepo.ListAuditLogs(ctx, repoFilter)
	if err != nil {
		logrus.WithError(err).Error("Failed to list Docker audit logs")
		return nil, 0, err
	}

	// If instance_id filter is provided, we need to filter by parsing Changes field
	// This is less efficient but necessary since instance_id is in the Changes JSON
	if filter.InstanceID != "" {
		filteredLogs := make([]*models.AuditLog, 0)
		for _, log := range logs {
			if details, err := log.ParseDockerDetails(); err == nil && details != nil {
				if details.InstanceID.String() == filter.InstanceID {
					filteredLogs = append(filteredLogs, log)
				}
			}
		}
		return filteredLogs, int64(len(filteredLogs)), nil
	}

	return logs, total, nil
}
