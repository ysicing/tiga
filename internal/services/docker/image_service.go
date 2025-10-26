package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// DockerStreamManager interface to avoid import cycle
type DockerStreamManager interface {
	CreateSession(instanceID uuid.UUID, agentID string, operation string, containerID, imageName string, params map[string]string) (interface{}, error)
}

// ImageService handles Docker image operations
// T036-T037: Migrated to unified audit system
type ImageService struct {
	instanceService     *DockerInstanceService
	agentForwarder      *AgentForwarderV2
	dockerStreamManager DockerStreamManager
	auditHelper         *AuditHelper
}

// NewImageService creates a new ImageService
func NewImageService(
	instanceService *DockerInstanceService,
	agentForwarder *AgentForwarderV2,
	dockerStreamManager DockerStreamManager,
	auditHelper *AuditHelper,
) *ImageService {
	return &ImageService{
		instanceService:     instanceService,
		agentForwarder:      agentForwarder,
		dockerStreamManager: dockerStreamManager,
		auditHelper:         auditHelper,
	}
}

// DeleteImage deletes an image
func (s *ImageService) DeleteImage(c *gin.Context, instanceID uuid.UUID, imageID string, force bool, noPrune bool) (*pb.DeleteImageResponse, error) {
	startTime := time.Now()

	// Pre-check: ensure instance is online
	instance, err := s.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return nil, fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	// Forward request to agent
	req := &pb.DeleteImageRequest{
		ImageId: imageID,
		Force:   force,
		NoPrune: noPrune,
	}

	resp, err := s.agentForwarder.DeleteImage(instance.AgentID, req)

	// Log audit (after operation)
	duration := time.Since(startTime).Milliseconds()
	s.auditHelper.LogDockerOperation(c, AuditParams{
		Action:       models.DockerActionDeleteImage,
		ResourceType: models.DockerResourceTypeImage,
		ResourceID:   instanceID,
		ResourceName: imageID,
		InstanceID:   instanceID,
		InstanceName: instance.Name,
		AgentID:      &instance.AgentID,
		ExtraData: map[string]interface{}{
			"image_id":      imageID,
			"force":         force,
			"no_prune":      noPrune,
			"deleted_count": len(resp.GetDeleted()),
		},
		Error:    err,
		Duration: duration,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to delete image: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"image_id":      imageID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"deleted_count": len(resp.Deleted),
	}).Info("Image deleted successfully")

	return resp, nil
}

// TagImage tags an image
func (s *ImageService) TagImage(c *gin.Context, instanceID uuid.UUID, source string, target string) error {
	startTime := time.Now()

	instance, err := s.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	req := &pb.TagImageRequest{
		Source: source,
		Target: target,
	}

	resp, err := s.agentForwarder.TagImage(instance.AgentID, req)

	duration := time.Since(startTime).Milliseconds()
	s.auditHelper.LogDockerOperation(c, AuditParams{
		Action:       models.DockerActionTagImage,
		ResourceType: models.DockerResourceTypeImage,
		ResourceID:   instanceID,
		ResourceName: source,
		InstanceID:   instanceID,
		InstanceName: instance.Name,
		AgentID:      &instance.AgentID,
		ExtraData: map[string]interface{}{
			"source": source,
			"target": target,
			"result": getResponseMessage(resp),
		},
		Error:    err,
		Duration: duration,
	})

	if err != nil {
		return fmt.Errorf("failed to tag image: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to tag image: %s", resp.Message)
	}

	logrus.WithFields(logrus.Fields{
		"source":        source,
		"target":        target,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
	}).Info("Image tagged successfully")

	return nil
}

// PullImage initiates an image pull (returns stream session for progress updates)
// Note: Audit log is created in Handler after pull completes
// Returns interface{} which is actually *host.DockerStreamSession to avoid import cycle
func (s *ImageService) PullImage(ctx context.Context, instanceID uuid.UUID, image string, registryAuth string) (interface{}, error) {
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
		"agent_id":      instance.AgentID.String(),
	}).Info("Starting image pull via DockerStream")

	// Create parameters for pull operation
	params := make(map[string]string)
	if registryAuth != "" {
		params["registry_auth"] = registryAuth
	}

	// Create Docker stream session for pull operation
	session, err := s.dockerStreamManager.CreateSession(
		instanceID,
		instance.AgentID.String(), // Convert UUID to string
		"pull_image",
		"",    // container_id not needed for pull
		image, // image_name
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker stream session: %w", err)
	}

	return session, nil
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
