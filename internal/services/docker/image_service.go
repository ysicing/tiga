package docker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ImageService handles Docker image operations
type ImageService struct {
	db              *gorm.DB
	instanceService *DockerInstanceService
	agentForwarder  *AgentForwarder
	enableAuditLog  bool
}

// NewImageService creates a new ImageService
func NewImageService(
	db *gorm.DB,
	instanceService *DockerInstanceService,
	agentForwarder *AgentForwarder,
) *ImageService {
	return &ImageService{
		db:              db,
		instanceService: instanceService,
		agentForwarder:  agentForwarder,
		enableAuditLog:  true, // Enable audit logging by default
	}
}

// DeleteImage deletes an image
func (s *ImageService) DeleteImage(ctx context.Context, instanceID uuid.UUID, imageID string, force bool, noPrune bool, userID *uuid.UUID, username string, ipAddress string) (*pb.DeleteImageResponse, error) {
	// Pre-check: ensure instance is online
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return nil, fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	// Create audit log (before operation)
	auditBefore := s.createAuditLog(userID, username, ipAddress, "delete", "image", imageID, fmt.Sprintf("Deleting image %s on instance %s (force: %v, no_prune: %v)", imageID, instance.Name, force, noPrune))

	// Forward request to agent
	req := &pb.DeleteImageRequest{
		ImageId: imageID,
		Force:   force,
		NoPrune: noPrune,
	}

	resp, err := s.agentForwarder.DeleteImage(instance.AgentID, req)
	if err != nil {
		s.updateAuditLogFailure(ctx, auditBefore, err.Error())
		return nil, fmt.Errorf("failed to delete image: %w", err)
	}

	// Update audit log with success
	deletedCount := len(resp.Deleted)
	s.updateAuditLogSuccess(ctx, auditBefore, map[string]interface{}{
		"image_id":      imageID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"force":         force,
		"no_prune":      noPrune,
		"deleted_count": deletedCount,
	})

	logrus.WithFields(logrus.Fields{
		"image_id":      imageID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"deleted_count": deletedCount,
		"user":          username,
	}).Info("Image deleted successfully")

	return resp, nil
}

// TagImage tags an image
func (s *ImageService) TagImage(ctx context.Context, instanceID uuid.UUID, source string, target string, userID *uuid.UUID, username string, ipAddress string) error {
	// Pre-check: ensure instance is online
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	// Create audit log (before operation)
	auditBefore := s.createAuditLog(userID, username, ipAddress, "tag", "image", source, fmt.Sprintf("Tagging image %s as %s on instance %s", source, target, instance.Name))

	// Forward request to agent
	req := &pb.TagImageRequest{
		Source: source,
		Target: target,
	}

	resp, err := s.agentForwarder.TagImage(instance.AgentID, req)
	if err != nil {
		s.updateAuditLogFailure(ctx, auditBefore, err.Error())
		return fmt.Errorf("failed to tag image: %w", err)
	}

	if !resp.Success {
		s.updateAuditLogFailure(ctx, auditBefore, resp.Message)
		return fmt.Errorf("failed to tag image: %s", resp.Message)
	}

	// Update audit log with success
	s.updateAuditLogSuccess(ctx, auditBefore, map[string]interface{}{
		"source":        source,
		"target":        target,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"result":        resp.Message,
	})

	logrus.WithFields(logrus.Fields{
		"source":        source,
		"target":        target,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"user":          username,
	}).Info("Image tagged successfully")

	return nil
}

// PullImage initiates an image pull (returns stream client for progress updates)
// Note: Audit log is created when pull completes (in the caller that reads the stream)
func (s *ImageService) PullImage(ctx context.Context, instanceID uuid.UUID, image string, registryAuth string) (pb.DockerService_PullImageClient, error) {
	// Pre-check: ensure instance is online
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return nil, fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	logrus.WithFields(logrus.Fields{
		"image":         image,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
	}).Info("Starting image pull")

	// Forward request to agent
	req := &pb.PullImageRequest{
		Image:        image,
		RegistryAuth: registryAuth,
	}

	stream, err := s.agentForwarder.PullImage(instance.AgentID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start image pull: %w", err)
	}

	return stream, nil
}

// CreatePullAuditLog creates an audit log for image pull operation
// This should be called after pull completes
func (s *ImageService) CreatePullAuditLog(ctx context.Context, instanceID uuid.UUID, image string, success bool, errorMessage string, userID *uuid.UUID, username string, ipAddress string) error {
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	audit := s.createAuditLog(userID, username, ipAddress, "pull", "image", image, fmt.Sprintf("Pulling image %s on instance %s", image, instance.Name))

	if success {
		s.updateAuditLogSuccess(ctx, audit, map[string]interface{}{
			"image":         image,
			"instance_id":   instanceID,
			"instance_name": instance.Name,
		})
	} else {
		s.updateAuditLogFailure(ctx, audit, errorMessage)
	}

	return nil
}

// GetRegistryAuth retrieves registry authentication for image operations
// Phase 1: Read from Docker config.json
// TODO: Phase 2: Support custom registry credentials stored in database
func (s *ImageService) GetRegistryAuth(registry string) (string, error) {
	// Phase 1: Return empty string (Agent will use default Docker config)
	// The Agent's Docker daemon will use its own config.json for authentication
	logrus.WithField("registry", registry).Debug("Using Agent's Docker config for registry auth")
	return "", nil
}

// createAuditLog creates an audit log entry
func (s *ImageService) createAuditLog(userID *uuid.UUID, username, ipAddress, action, resourceType, resourceID, description string) *models.AuditLog {
	if !s.enableAuditLog {
		return nil
	}

	return &models.AuditLog{
		UserID:       userID,
		Username:     username,
		Action:       action,
		ResourceType: resourceType,
		ResourceName: resourceID, // Use image ID/name as resource name
		Description:  description,
		IPAddress:    ipAddress,
		Status:       "pending",
	}
}

// updateAuditLogSuccess updates audit log with success status
func (s *ImageService) updateAuditLogSuccess(ctx context.Context, audit *models.AuditLog, changes map[string]interface{}) {
	if !s.enableAuditLog || audit == nil {
		return
	}

	audit.Status = "success"
	audit.Changes = models.JSONB(changes)

	if err := s.db.WithContext(ctx).Create(audit).Error; err != nil {
		logrus.WithError(err).Error("Failed to create audit log")
	}
}

// updateAuditLogFailure updates audit log with failure status
func (s *ImageService) updateAuditLogFailure(ctx context.Context, audit *models.AuditLog, errorMessage string) {
	if !s.enableAuditLog || audit == nil {
		return
	}

	audit.Status = "failure"
	audit.ErrorMessage = errorMessage

	if err := s.db.WithContext(ctx).Create(audit).Error; err != nil {
		logrus.WithError(err).Error("Failed to create audit log")
	}
}
