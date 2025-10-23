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

// ContainerService handles Docker container operations
type ContainerService struct {
	db              *gorm.DB
	instanceService *DockerInstanceService
	agentForwarder  *AgentForwarder
	enableAuditLog  bool
}

// NewContainerService creates a new ContainerService
func NewContainerService(
	db *gorm.DB,
	instanceService *DockerInstanceService,
	agentForwarder *AgentForwarder,
) *ContainerService {
	return &ContainerService{
		db:              db,
		instanceService: instanceService,
		agentForwarder:  agentForwarder,
		enableAuditLog:  true, // Enable audit logging by default
	}
}

// StartContainer starts a container
func (s *ContainerService) StartContainer(ctx context.Context, instanceID uuid.UUID, containerID string, userID *uuid.UUID, username string, ipAddress string) error {
	// Pre-check: ensure instance is online
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	// Create audit log (before operation)
	auditBefore := s.createAuditLog(userID, username, ipAddress, "start", "container", containerID, fmt.Sprintf("Starting container %s on instance %s", containerID, instance.Name))

	// Forward request to agent
	req := &pb.StartContainerRequest{
		ContainerId: containerID,
	}

	resp, err := s.agentForwarder.StartContainer(instance.AgentID, req)
	if err != nil {
		// Update audit log with failure
		s.updateAuditLogFailure(ctx, auditBefore, err.Error())
		return fmt.Errorf("failed to start container: %w", err)
	}

	if !resp.Success {
		// Update audit log with failure
		s.updateAuditLogFailure(ctx, auditBefore, resp.Message)
		return fmt.Errorf("failed to start container: %s", resp.Message)
	}

	// Update audit log with success
	s.updateAuditLogSuccess(ctx, auditBefore, map[string]interface{}{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"result":        resp.Message,
	})

	logrus.WithFields(logrus.Fields{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"user":          username,
	}).Info("Container started successfully")

	return nil
}

// StopContainer stops a container
func (s *ContainerService) StopContainer(ctx context.Context, instanceID uuid.UUID, containerID string, timeout int32, userID *uuid.UUID, username string, ipAddress string) error {
	// Pre-check: ensure instance is online
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	// Create audit log (before operation)
	auditBefore := s.createAuditLog(userID, username, ipAddress, "stop", "container", containerID, fmt.Sprintf("Stopping container %s on instance %s (timeout: %ds)", containerID, instance.Name, timeout))

	// Forward request to agent
	req := &pb.StopContainerRequest{
		ContainerId: containerID,
		Timeout:     timeout,
	}

	resp, err := s.agentForwarder.StopContainer(instance.AgentID, req)
	if err != nil {
		s.updateAuditLogFailure(ctx, auditBefore, err.Error())
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if !resp.Success {
		s.updateAuditLogFailure(ctx, auditBefore, resp.Message)
		return fmt.Errorf("failed to stop container: %s", resp.Message)
	}

	// Update audit log with success
	s.updateAuditLogSuccess(ctx, auditBefore, map[string]interface{}{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"timeout":       timeout,
		"result":        resp.Message,
	})

	logrus.WithFields(logrus.Fields{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"timeout":       timeout,
		"user":          username,
	}).Info("Container stopped successfully")

	return nil
}

// RestartContainer restarts a container
func (s *ContainerService) RestartContainer(ctx context.Context, instanceID uuid.UUID, containerID string, timeout int32, userID *uuid.UUID, username string, ipAddress string) error {
	// Pre-check: ensure instance is online
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	// Create audit log (before operation)
	auditBefore := s.createAuditLog(userID, username, ipAddress, "restart", "container", containerID, fmt.Sprintf("Restarting container %s on instance %s (timeout: %ds)", containerID, instance.Name, timeout))

	// Forward request to agent
	req := &pb.RestartContainerRequest{
		ContainerId: containerID,
		Timeout:     timeout,
	}

	resp, err := s.agentForwarder.RestartContainer(instance.AgentID, req)
	if err != nil {
		s.updateAuditLogFailure(ctx, auditBefore, err.Error())
		return fmt.Errorf("failed to restart container: %w", err)
	}

	if !resp.Success {
		s.updateAuditLogFailure(ctx, auditBefore, resp.Message)
		return fmt.Errorf("failed to restart container: %s", resp.Message)
	}

	// Update audit log with success
	s.updateAuditLogSuccess(ctx, auditBefore, map[string]interface{}{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"timeout":       timeout,
		"result":        resp.Message,
	})

	logrus.WithFields(logrus.Fields{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"timeout":       timeout,
		"user":          username,
	}).Info("Container restarted successfully")

	return nil
}

// PauseContainer pauses a container
func (s *ContainerService) PauseContainer(ctx context.Context, instanceID uuid.UUID, containerID string, userID *uuid.UUID, username string, ipAddress string) error {
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	auditBefore := s.createAuditLog(userID, username, ipAddress, "pause", "container", containerID, fmt.Sprintf("Pausing container %s on instance %s", containerID, instance.Name))

	req := &pb.PauseContainerRequest{ContainerId: containerID}
	resp, err := s.agentForwarder.PauseContainer(instance.AgentID, req)
	if err != nil {
		s.updateAuditLogFailure(ctx, auditBefore, err.Error())
		return fmt.Errorf("failed to pause container: %w", err)
	}

	if !resp.Success {
		s.updateAuditLogFailure(ctx, auditBefore, resp.Message)
		return fmt.Errorf("failed to pause container: %s", resp.Message)
	}

	s.updateAuditLogSuccess(ctx, auditBefore, map[string]interface{}{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
	})

	logrus.WithFields(logrus.Fields{
		"container_id": containerID,
		"instance_id":  instanceID,
		"user":         username,
	}).Info("Container paused successfully")

	return nil
}

// UnpauseContainer unpauses a container
func (s *ContainerService) UnpauseContainer(ctx context.Context, instanceID uuid.UUID, containerID string, userID *uuid.UUID, username string, ipAddress string) error {
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	auditBefore := s.createAuditLog(userID, username, ipAddress, "unpause", "container", containerID, fmt.Sprintf("Unpausing container %s on instance %s", containerID, instance.Name))

	req := &pb.UnpauseContainerRequest{ContainerId: containerID}
	resp, err := s.agentForwarder.UnpauseContainer(instance.AgentID, req)
	if err != nil {
		s.updateAuditLogFailure(ctx, auditBefore, err.Error())
		return fmt.Errorf("failed to unpause container: %w", err)
	}

	if !resp.Success {
		s.updateAuditLogFailure(ctx, auditBefore, resp.Message)
		return fmt.Errorf("failed to unpause container: %s", resp.Message)
	}

	s.updateAuditLogSuccess(ctx, auditBefore, map[string]interface{}{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
	})

	logrus.WithFields(logrus.Fields{
		"container_id": containerID,
		"instance_id":  instanceID,
		"user":         username,
	}).Info("Container unpaused successfully")

	return nil
}

// DeleteContainer deletes a container
func (s *ContainerService) DeleteContainer(ctx context.Context, instanceID uuid.UUID, containerID string, force bool, removeVolumes bool, userID *uuid.UUID, username string, ipAddress string) error {
	instance, err := s.instanceService.GetByID(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	auditBefore := s.createAuditLog(userID, username, ipAddress, "delete", "container", containerID, fmt.Sprintf("Deleting container %s on instance %s (force: %v, remove_volumes: %v)", containerID, instance.Name, force, removeVolumes))

	req := &pb.DeleteContainerRequest{
		ContainerId:   containerID,
		Force:         force,
		RemoveVolumes: removeVolumes,
	}

	resp, err := s.agentForwarder.DeleteContainer(instance.AgentID, req)
	if err != nil {
		s.updateAuditLogFailure(ctx, auditBefore, err.Error())
		return fmt.Errorf("failed to delete container: %w", err)
	}

	if !resp.Success {
		s.updateAuditLogFailure(ctx, auditBefore, resp.Message)
		return fmt.Errorf("failed to delete container: %s", resp.Message)
	}

	s.updateAuditLogSuccess(ctx, auditBefore, map[string]interface{}{
		"container_id":   containerID,
		"instance_id":    instanceID,
		"instance_name":  instance.Name,
		"force":          force,
		"remove_volumes": removeVolumes,
	})

	logrus.WithFields(logrus.Fields{
		"container_id":   containerID,
		"instance_id":    instanceID,
		"force":          force,
		"remove_volumes": removeVolumes,
		"user":           username,
	}).Info("Container deleted successfully")

	return nil
}

// createAuditLog creates an audit log entry
func (s *ContainerService) createAuditLog(userID *uuid.UUID, username, ipAddress, action, resourceType, resourceID, description string) *models.AuditLog {
	if !s.enableAuditLog {
		return nil
	}

	return &models.AuditLog{
		UserID:       userID,
		Username:     username,
		Action:       action,
		ResourceType: resourceType,
		ResourceName: resourceID, // Use container ID as resource name
		Description:  description,
		IPAddress:    ipAddress,
		Status:       "pending",
	}
}

// updateAuditLogSuccess updates audit log with success status
func (s *ContainerService) updateAuditLogSuccess(ctx context.Context, audit *models.AuditLog, changes map[string]interface{}) {
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
func (s *ContainerService) updateAuditLogFailure(ctx context.Context, audit *models.AuditLog, errorMessage string) {
	if !s.enableAuditLog || audit == nil {
		return
	}

	audit.Status = "failure"
	audit.ErrorMessage = errorMessage

	if err := s.db.WithContext(ctx).Create(audit).Error; err != nil {
		logrus.WithError(err).Error("Failed to create audit log")
	}
}
