package docker

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ContainerService handles Docker container operations
// T036-T037: Migrated to unified audit system
type ContainerService struct {
	instanceService *DockerInstanceService
	agentForwarder  *AgentForwarderV2
	auditHelper     *AuditHelper
}

// NewContainerService creates a new ContainerService
func NewContainerService(
	instanceService *DockerInstanceService,
	agentForwarder *AgentForwarderV2,
	auditHelper *AuditHelper,
) *ContainerService {
	return &ContainerService{
		instanceService: instanceService,
		agentForwarder:  agentForwarder,
		auditHelper:     auditHelper,
	}
}

// StartContainer starts a container
func (s *ContainerService) StartContainer(c *gin.Context, instanceID uuid.UUID, containerID string) error {
	startTime := time.Now()

	// Pre-check: ensure instance is online
	instance, err := s.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	// Forward request to agent
	req := &pb.StartContainerRequest{
		ContainerId: containerID,
	}

	resp, err := s.agentForwarder.StartContainer(instance.AgentID, req)

	// Log audit (after operation)
	duration := time.Since(startTime).Milliseconds()
	s.auditHelper.LogDockerOperation(c, AuditParams{
		Action:       models.DockerActionStartContainer,
		ResourceType: models.DockerResourceTypeContainer,
		ResourceID:   instanceID, // Use instance ID as we're operating on its container
		ResourceName: containerID,
		InstanceID:   instanceID,
		InstanceName: instance.Name,
		AgentID:      &instance.AgentID,
		ExtraData: map[string]interface{}{
			"container_id": containerID,
			"result":       getResponseMessage(resp),
		},
		Error:    err,
		Duration: duration,
	})

	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to start container: %s", resp.Message)
	}

	logrus.WithFields(logrus.Fields{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
	}).Info("Container started successfully")

	return nil
}

// StopContainer stops a container
func (s *ContainerService) StopContainer(c *gin.Context, instanceID uuid.UUID, containerID string, timeout int32) error {
	startTime := time.Now()

	instance, err := s.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	req := &pb.StopContainerRequest{
		ContainerId: containerID,
		Timeout:     timeout,
	}

	resp, err := s.agentForwarder.StopContainer(instance.AgentID, req)

	duration := time.Since(startTime).Milliseconds()
	s.auditHelper.LogDockerOperation(c, AuditParams{
		Action:       models.DockerActionStopContainer,
		ResourceType: models.DockerResourceTypeContainer,
		ResourceID:   instanceID,
		ResourceName: containerID,
		InstanceID:   instanceID,
		InstanceName: instance.Name,
		AgentID:      &instance.AgentID,
		ExtraData: map[string]interface{}{
			"container_id": containerID,
			"timeout":      timeout,
			"result":       getResponseMessage(resp),
		},
		Error:    err,
		Duration: duration,
	})

	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to stop container: %s", resp.Message)
	}

	logrus.WithFields(logrus.Fields{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"timeout":       timeout,
	}).Info("Container stopped successfully")

	return nil
}

// RestartContainer restarts a container
func (s *ContainerService) RestartContainer(c *gin.Context, instanceID uuid.UUID, containerID string, timeout int32) error {
	startTime := time.Now()

	instance, err := s.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	req := &pb.RestartContainerRequest{
		ContainerId: containerID,
		Timeout:     timeout,
	}

	resp, err := s.agentForwarder.RestartContainer(instance.AgentID, req)

	duration := time.Since(startTime).Milliseconds()
	s.auditHelper.LogDockerOperation(c, AuditParams{
		Action:       models.DockerActionRestartContainer,
		ResourceType: models.DockerResourceTypeContainer,
		ResourceID:   instanceID,
		ResourceName: containerID,
		InstanceID:   instanceID,
		InstanceName: instance.Name,
		AgentID:      &instance.AgentID,
		ExtraData: map[string]interface{}{
			"container_id": containerID,
			"timeout":      timeout,
			"result":       getResponseMessage(resp),
		},
		Error:    err,
		Duration: duration,
	})

	if err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to restart container: %s", resp.Message)
	}

	logrus.WithFields(logrus.Fields{
		"container_id":  containerID,
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"timeout":       timeout,
	}).Info("Container restarted successfully")

	return nil
}

// PauseContainer pauses a container
func (s *ContainerService) PauseContainer(c *gin.Context, instanceID uuid.UUID, containerID string) error {
	startTime := time.Now()

	instance, err := s.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	req := &pb.PauseContainerRequest{ContainerId: containerID}
	resp, err := s.agentForwarder.PauseContainer(instance.AgentID, req)

	duration := time.Since(startTime).Milliseconds()
	s.auditHelper.LogDockerOperation(c, AuditParams{
		Action:       models.DockerActionPauseContainer,
		ResourceType: models.DockerResourceTypeContainer,
		ResourceID:   instanceID,
		ResourceName: containerID,
		InstanceID:   instanceID,
		InstanceName: instance.Name,
		AgentID:      &instance.AgentID,
		ExtraData: map[string]interface{}{
			"container_id": containerID,
			"result":       getResponseMessage(resp),
		},
		Error:    err,
		Duration: duration,
	})

	if err != nil {
		return fmt.Errorf("failed to pause container: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to pause container: %s", resp.Message)
	}

	logrus.WithFields(logrus.Fields{
		"container_id": containerID,
		"instance_id":  instanceID,
	}).Info("Container paused successfully")

	return nil
}

// UnpauseContainer unpauses a container
func (s *ContainerService) UnpauseContainer(c *gin.Context, instanceID uuid.UUID, containerID string) error {
	startTime := time.Now()

	instance, err := s.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	req := &pb.UnpauseContainerRequest{ContainerId: containerID}
	resp, err := s.agentForwarder.UnpauseContainer(instance.AgentID, req)

	duration := time.Since(startTime).Milliseconds()
	s.auditHelper.LogDockerOperation(c, AuditParams{
		Action:       models.DockerActionUnpauseContainer,
		ResourceType: models.DockerResourceTypeContainer,
		ResourceID:   instanceID,
		ResourceName: containerID,
		InstanceID:   instanceID,
		InstanceName: instance.Name,
		AgentID:      &instance.AgentID,
		ExtraData: map[string]interface{}{
			"container_id": containerID,
			"result":       getResponseMessage(resp),
		},
		Error:    err,
		Duration: duration,
	})

	if err != nil {
		return fmt.Errorf("failed to unpause container: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to unpause container: %s", resp.Message)
	}

	logrus.WithFields(logrus.Fields{
		"container_id": containerID,
		"instance_id":  instanceID,
	}).Info("Container unpaused successfully")

	return nil
}

// DeleteContainer deletes a container
func (s *ContainerService) DeleteContainer(c *gin.Context, instanceID uuid.UUID, containerID string, force bool, removeVolumes bool) error {
	startTime := time.Now()

	instance, err := s.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if !instance.CanOperate() {
		return fmt.Errorf("instance is not online (status: %s)", instance.HealthStatus)
	}

	req := &pb.DeleteContainerRequest{
		ContainerId:   containerID,
		Force:         force,
		RemoveVolumes: removeVolumes,
	}

	resp, err := s.agentForwarder.DeleteContainer(instance.AgentID, req)

	duration := time.Since(startTime).Milliseconds()
	s.auditHelper.LogDockerOperation(c, AuditParams{
		Action:       models.DockerActionDeleteContainer,
		ResourceType: models.DockerResourceTypeContainer,
		ResourceID:   instanceID,
		ResourceName: containerID,
		InstanceID:   instanceID,
		InstanceName: instance.Name,
		AgentID:      &instance.AgentID,
		ExtraData: map[string]interface{}{
			"container_id":   containerID,
			"force":          force,
			"remove_volumes": removeVolumes,
			"result":         getResponseMessage(resp),
		},
		Error:    err,
		Duration: duration,
	})

	if err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to delete container: %s", resp.Message)
	}

	logrus.WithFields(logrus.Fields{
		"container_id":   containerID,
		"instance_id":    instanceID,
		"force":          force,
		"remove_volumes": removeVolumes,
	}).Info("Container deleted successfully")

	return nil
}

// getResponseMessage safely extracts message from Docker response
func getResponseMessage(resp interface{}) string {
	if resp == nil {
		return ""
	}

	type messageGetter interface {
		GetMessage() string
	}

	if mg, ok := resp.(messageGetter); ok {
		return mg.GetMessage()
	}

	return ""
}
