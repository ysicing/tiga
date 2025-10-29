package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/audit"
)

// K8sAuditService handles K8s operation auditing
// Reference: 010-k8s-pod-009 T021
type K8sAuditService struct {
	asyncLogger *audit.AsyncLogger[*models.AuditEvent]
	clusterRepo repository.ClusterRepositoryInterface
}

// ResourceOperationLog represents a K8s resource operation audit log
type ResourceOperationLog struct {
	ClusterID    string
	Namespace    string
	ResourceType models.ResourceType
	ResourceName string
	Action       models.Action
	UserID       string
	Username     string
	ClientIP     string
	RequestID    string
	Success      bool
	ErrorMessage string
	OldObject    interface{} // For updates
	NewObject    interface{} // For creates/updates
}

// TerminalAccessLog represents a K8s terminal access audit log
type TerminalAccessLog struct {
	ClusterID     string
	NodeName      string // For node terminal
	Namespace     string // For pod exec
	PodName       string // For pod exec
	ContainerName string // For pod exec
	Action        models.Action
	RecordingID   *uuid.UUID
	Duration      int
	UserID        string
	Username      string
	ClientIP      string
	RequestID     string
	Success       bool
}

// ReadOperationLog represents a read-only operation audit log
type ReadOperationLog struct {
	ClusterID    string
	Namespace    string
	ResourceType models.ResourceType
	ResourceName string
	Operation    string // "get", "list", "watch"
	UserID       string
	Username     string
	ClientIP     string
	RequestID    string
}

// NewAuditService creates a new K8s audit service
func NewAuditService(asyncLogger *audit.AsyncLogger[*models.AuditEvent], clusterRepo repository.ClusterRepositoryInterface) *K8sAuditService {
	return &K8sAuditService{
		asyncLogger: asyncLogger,
		clusterRepo: clusterRepo,
	}
}

// LogResourceOperation logs a K8s resource operation (create/update/delete)
func (s *K8sAuditService) LogResourceOperation(ctx context.Context, log *ResourceOperationLog) error {
	now := time.Now()

	// Create resource data
	resourceData := map[string]string{
		"cluster_id":    log.ClusterID,
		"resource_name": log.ResourceName,
		"success":       fmt.Sprintf("%t", log.Success),
	}

	if log.Namespace != "" {
		resourceData["namespace"] = log.Namespace
	}

	if !log.Success && log.ErrorMessage != "" {
		resourceData["error"] = log.ErrorMessage
	}

	// Add change summary for updates
	if log.Action == models.ActionUpdateResource && log.OldObject != nil && log.NewObject != nil {
		// TODO: Generate change summary by comparing old and new objects
		resourceData["change_summary"] = "Resource updated"
	}

	// Create audit event
	auditEvent := &models.AuditEvent{
		ID:           uuid.New().String(),
		Timestamp:    now.UnixMilli(),
		Action:       log.Action,
		ResourceType: log.ResourceType,
		Resource: models.Resource{
			Type:       log.ResourceType,
			Identifier: log.ResourceName,
			Data:       resourceData,
		},
		Subsystem: models.SubsystemKubernetes,
		User: models.Principal{
			UID:      log.UserID,
			Username: log.Username,
			Type:     models.PrincipalTypeUser,
		},
		ClientIP:  log.ClientIP,
		RequestID: log.RequestID,
		CreatedAt: now,
	}

	// Add diff object for updates
	if log.Action == models.ActionUpdateResource && log.OldObject != nil && log.NewObject != nil {
		if err := auditEvent.MarshalOldObject(log.OldObject); err == nil {
			auditEvent.MarshalNewObject(log.NewObject)
		}
	}

	// Log asynchronously
	s.asyncLogger.Log(ctx, auditEvent)

	return nil
}

// LogTerminalAccess logs a K8s terminal access (node terminal or pod exec)
func (s *K8sAuditService) LogTerminalAccess(ctx context.Context, log *TerminalAccessLog) error {
	now := time.Now()

	// Determine resource type and create resource data
	var resourceType models.ResourceType
	var identifier string
	resourceData := map[string]string{
		"cluster_id": log.ClusterID,
		"success":    fmt.Sprintf("%t", log.Success),
	}

	if log.Action == models.ActionNodeTerminalAccess {
		// Node terminal
		resourceType = models.ResourceTypeK8sNode
		identifier = log.NodeName
		resourceData["resource_name"] = log.NodeName
	} else if log.Action == models.ActionPodTerminalAccess {
		// Pod exec
		resourceType = models.ResourceTypeK8sPod
		identifier = log.PodName
		resourceData["namespace"] = log.Namespace
		resourceData["pod_name"] = log.PodName
		resourceData["container_name"] = log.ContainerName
	}

	// Add recording_id if available
	if log.RecordingID != nil {
		resourceData["recording_id"] = log.RecordingID.String()
	}

	// Add duration if session ended
	if log.Duration > 0 {
		resourceData["duration"] = fmt.Sprintf("%d", log.Duration)
	}

	// Create audit event
	auditEvent := &models.AuditEvent{
		ID:           uuid.New().String(),
		Timestamp:    now.UnixMilli(),
		Action:       log.Action,
		ResourceType: resourceType,
		Resource: models.Resource{
			Type:       resourceType,
			Identifier: identifier,
			Data:       resourceData,
		},
		Subsystem: models.SubsystemKubernetes,
		User: models.Principal{
			UID:      log.UserID,
			Username: log.Username,
			Type:     models.PrincipalTypeUser,
		},
		ClientIP:  log.ClientIP,
		RequestID: log.RequestID,
		CreatedAt: now,
	}

	// Log asynchronously
	s.asyncLogger.Log(ctx, auditEvent)

	return nil
}

// LogReadOperation logs a read-only K8s operation (get/list/watch)
func (s *K8sAuditService) LogReadOperation(ctx context.Context, log *ReadOperationLog) error {
	now := time.Now()

	resourceData := map[string]string{
		"cluster_id":    log.ClusterID,
		"resource_name": log.ResourceName,
		"operation":     log.Operation,
		"success":       "true",
	}

	if log.Namespace != "" {
		resourceData["namespace"] = log.Namespace
	}

	auditEvent := &models.AuditEvent{
		ID:           uuid.New().String(),
		Timestamp:    now.UnixMilli(),
		Action:       models.ActionViewResource,
		ResourceType: log.ResourceType,
		Resource: models.Resource{
			Type:       log.ResourceType,
			Identifier: log.ResourceName,
			Data:       resourceData,
		},
		Subsystem: models.SubsystemKubernetes,
		User: models.Principal{
			UID:      log.UserID,
			Username: log.Username,
			Type:     models.PrincipalTypeUser,
		},
		ClientIP:  log.ClientIP,
		RequestID: log.RequestID,
		CreatedAt: now,
	}

	// Log asynchronously
	s.asyncLogger.Log(ctx, auditEvent)

	return nil
}
