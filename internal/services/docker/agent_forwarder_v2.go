package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
	"github.com/ysicing/tiga/proto"
)

const (
	defaultTaskTimeout = 30 * time.Second
)

// AgentManager interface for task queue operations
type AgentManager interface {
	QueueTaskAndWait(ctx context.Context, uuid string, task *proto.AgentTask, timeout time.Duration) (*proto.TaskResult, error)
}

// AgentForwarderV2 forwards Docker operations to agents via task queue
type AgentForwarderV2 struct {
	agentManager AgentManager
	db           *gorm.DB
}

// NewAgentForwarderV2 creates a new AgentForwarderV2 instance
func NewAgentForwarderV2(agentManager AgentManager, db *gorm.DB) *AgentForwarderV2 {
	return &AgentForwarderV2{
		agentManager: agentManager,
		db:           db,
	}
}

// getHostUUID retrieves the host UUID for a Docker instance
func (f *AgentForwarderV2) getHostUUID(instanceID uuid.UUID) (string, error) {
	// Step 1: Get docker instance to find agent_id
	var dockerInstance models.DockerInstance
	if err := f.db.Where("id = ?", instanceID).First(&dockerInstance).Error; err != nil {
		return "", fmt.Errorf("docker instance not found: %w", err)
	}

	// Step 2: Get agent connection to find host_node_id
	var agentConn models.AgentConnection
	if err := f.db.Where("id = ?", dockerInstance.AgentID).First(&agentConn).Error; err != nil {
		return "", fmt.Errorf("agent connection not found: %w", err)
	}

	// Step 3: Get host node to find UUID
	var hostNode models.HostNode
	if err := f.db.Where("id = ?", agentConn.HostNodeID).First(&hostNode).Error; err != nil {
		return "", fmt.Errorf("host node not found: %w", err)
	}

	return hostNode.ID.String(), nil
}

// executeTask is a helper to execute a Docker task and unmarshal the result
func (f *AgentForwarderV2) executeTask(ctx context.Context, instanceID uuid.UUID, operation string, params map[string]string, payload interface{}, result interface{}) error {
	// Lookup host UUID
	hostUUID, err := f.getHostUUID(instanceID)
	if err != nil {
		return err
	}

	taskID := uuid.New().String()

	// Marshal payload if provided
	var payloadBytes []byte
	if payload != nil {
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	// Create task
	if params == nil {
		params = make(map[string]string)
	}
	params["operation"] = operation

	task := &proto.AgentTask{
		TaskId:   taskID,
		TaskType: "docker",
		Params:   params,
		Payload:  payloadBytes,
	}

	logrus.WithFields(logrus.Fields{
		"host_uuid":   hostUUID,
		"instance_id": instanceID,
		"task_id":     taskID,
		"operation":   operation,
	}).Debug("[AgentForwarderV2] Executing Docker task")

	// Execute task and wait for result
	taskResult, err := f.agentManager.QueueTaskAndWait(ctx, hostUUID, task, defaultTaskTimeout)
	if err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	if !taskResult.Success {
		return fmt.Errorf("task failed: %s", taskResult.Error)
	}

	// Unmarshal result if needed
	if result != nil && len(taskResult.Payload) > 0 {
		if err := json.Unmarshal(taskResult.Payload, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}

// ListContainers forwards ListContainers request to the agent
func (f *AgentForwarderV2) ListContainers(instanceID uuid.UUID, req *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	var resp pb.ListContainersResponse
	err := f.executeTask(context.Background(), instanceID, "list_containers", nil, req, &resp)
	return &resp, err
}

// GetContainer forwards GetContainer request to the agent
func (f *AgentForwarderV2) GetContainer(instanceID uuid.UUID, req *pb.GetContainerRequest) (*pb.GetContainerResponse, error) {
	params := map[string]string{"container_id": req.ContainerId}
	var resp pb.GetContainerResponse
	err := f.executeTask(context.Background(), instanceID, "get_container", params, nil, &resp)
	return &resp, err
}

// StartContainer forwards StartContainer request to the agent
func (f *AgentForwarderV2) StartContainer(instanceID uuid.UUID, req *pb.StartContainerRequest) (*pb.StartContainerResponse, error) {
	params := map[string]string{"container_id": req.ContainerId}
	var resp pb.StartContainerResponse
	err := f.executeTask(context.Background(), instanceID, "start_container", params, nil, &resp)
	return &resp, err
}

// StopContainer forwards StopContainer request to the agent
func (f *AgentForwarderV2) StopContainer(instanceID uuid.UUID, req *pb.StopContainerRequest) (*pb.StopContainerResponse, error) {
	params := map[string]string{"container_id": req.ContainerId}
	var resp pb.StopContainerResponse
	err := f.executeTask(context.Background(), instanceID, "stop_container", params, nil, &resp)
	return &resp, err
}

// RestartContainer forwards RestartContainer request to the agent
func (f *AgentForwarderV2) RestartContainer(instanceID uuid.UUID, req *pb.RestartContainerRequest) (*pb.RestartContainerResponse, error) {
	params := map[string]string{"container_id": req.ContainerId}
	var resp pb.RestartContainerResponse
	err := f.executeTask(context.Background(), instanceID, "restart_container", params, nil, &resp)
	return &resp, err
}

// PauseContainer forwards PauseContainer request to the agent
func (f *AgentForwarderV2) PauseContainer(instanceID uuid.UUID, req *pb.PauseContainerRequest) (*pb.PauseContainerResponse, error) {
	params := map[string]string{"container_id": req.ContainerId}
	var resp pb.PauseContainerResponse
	err := f.executeTask(context.Background(), instanceID, "pause_container", params, nil, &resp)
	return &resp, err
}

// UnpauseContainer forwards UnpauseContainer request to the agent
func (f *AgentForwarderV2) UnpauseContainer(instanceID uuid.UUID, req *pb.UnpauseContainerRequest) (*pb.UnpauseContainerResponse, error) {
	params := map[string]string{"container_id": req.ContainerId}
	var resp pb.UnpauseContainerResponse
	err := f.executeTask(context.Background(), instanceID, "unpause_container", params, nil, &resp)
	return &resp, err
}

// DeleteContainer forwards DeleteContainer request to the agent
func (f *AgentForwarderV2) DeleteContainer(instanceID uuid.UUID, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error) {
	params := map[string]string{"container_id": req.ContainerId}
	var resp pb.DeleteContainerResponse
	err := f.executeTask(context.Background(), instanceID, "delete_container", params, nil, &resp)
	return &resp, err
}

// ListImages forwards ListImages request to the agent
func (f *AgentForwarderV2) ListImages(instanceID uuid.UUID, req *pb.ListImagesRequest) (*pb.ListImagesResponse, error) {
	var resp pb.ListImagesResponse
	err := f.executeTask(context.Background(), instanceID, "list_images", nil, req, &resp)
	return &resp, err
}

// GetImage forwards GetImage request to the agent
func (f *AgentForwarderV2) GetImage(instanceID uuid.UUID, req *pb.GetImageRequest) (*pb.GetImageResponse, error) {
	var resp pb.GetImageResponse
	params := map[string]string{"image_id": req.ImageId}
	err := f.executeTask(context.Background(), instanceID, "get_image", params, nil, &resp)
	return &resp, err
}

// ListNetworks forwards ListNetworks request to the agent
func (f *AgentForwarderV2) ListNetworks(instanceID uuid.UUID, req *pb.ListNetworksRequest) (*pb.ListNetworksResponse, error) {
	var resp pb.ListNetworksResponse
	err := f.executeTask(context.Background(), instanceID, "list_networks", nil, req, &resp)
	return &resp, err
}

// GetNetwork forwards GetNetwork request to the agent
func (f *AgentForwarderV2) GetNetwork(instanceID uuid.UUID, req *pb.GetNetworkRequest) (*pb.GetNetworkResponse, error) {
	var resp pb.GetNetworkResponse
	params := map[string]string{"network_id": req.NetworkId}
	err := f.executeTask(context.Background(), instanceID, "get_network", params, nil, &resp)
	return &resp, err
}

// CreateNetwork forwards CreateNetwork request to the agent
func (f *AgentForwarderV2) CreateNetwork(instanceID uuid.UUID, req *pb.CreateNetworkRequest) (*pb.CreateNetworkResponse, error) {
	var resp pb.CreateNetworkResponse
	err := f.executeTask(context.Background(), instanceID, "create_network", nil, req, &resp)
	return &resp, err
}

// DeleteNetwork forwards DeleteNetwork request to the agent
func (f *AgentForwarderV2) DeleteNetwork(instanceID uuid.UUID, req *pb.DeleteNetworkRequest) (*pb.DeleteNetworkResponse, error) {
	var resp pb.DeleteNetworkResponse
	params := map[string]string{"network_id": req.NetworkId}
	err := f.executeTask(context.Background(), instanceID, "delete_network", params, nil, &resp)
	return &resp, err
}

// ConnectNetwork forwards ConnectNetwork request to the agent
func (f *AgentForwarderV2) ConnectNetwork(instanceID uuid.UUID, req *pb.ConnectNetworkRequest) (*pb.ConnectNetworkResponse, error) {
	var resp pb.ConnectNetworkResponse
	err := f.executeTask(context.Background(), instanceID, "connect_network", nil, req, &resp)
	return &resp, err
}

// DisconnectNetwork forwards DisconnectNetwork request to the agent
func (f *AgentForwarderV2) DisconnectNetwork(instanceID uuid.UUID, req *pb.DisconnectNetworkRequest) (*pb.DisconnectNetworkResponse, error) {
	var resp pb.DisconnectNetworkResponse
	err := f.executeTask(context.Background(), instanceID, "disconnect_network", nil, req, &resp)
	return &resp, err
}

// ListVolumes forwards ListVolumes request to the agent
func (f *AgentForwarderV2) ListVolumes(instanceID uuid.UUID, req *pb.ListVolumesRequest) (*pb.ListVolumesResponse, error) {
	var resp pb.ListVolumesResponse
	err := f.executeTask(context.Background(), instanceID, "list_volumes", nil, req, &resp)
	return &resp, err
}

// GetVolume forwards GetVolume request to the agent
func (f *AgentForwarderV2) GetVolume(instanceID uuid.UUID, req *pb.GetVolumeRequest) (*pb.GetVolumeResponse, error) {
	var resp pb.GetVolumeResponse
	params := map[string]string{"volume_name": req.Name}
	err := f.executeTask(context.Background(), instanceID, "get_volume", params, nil, &resp)
	return &resp, err
}

// CreateVolume forwards CreateVolume request to the agent
func (f *AgentForwarderV2) CreateVolume(instanceID uuid.UUID, req *pb.CreateVolumeRequest) (*pb.CreateVolumeResponse, error) {
	var resp pb.CreateVolumeResponse
	err := f.executeTask(context.Background(), instanceID, "create_volume", nil, req, &resp)
	return &resp, err
}

// DeleteVolume forwards DeleteVolume request to the agent
func (f *AgentForwarderV2) DeleteVolume(instanceID uuid.UUID, req *pb.DeleteVolumeRequest) (*pb.DeleteVolumeResponse, error) {
	var resp pb.DeleteVolumeResponse
	params := map[string]string{"volume_name": req.Name}
	err := f.executeTask(context.Background(), instanceID, "delete_volume", params, nil, &resp)
	return &resp, err
}

// PruneVolumes forwards PruneVolumes request to the agent
func (f *AgentForwarderV2) PruneVolumes(instanceID uuid.UUID, req *pb.PruneVolumesRequest) (*pb.PruneVolumesResponse, error) {
	var resp pb.PruneVolumesResponse
	err := f.executeTask(context.Background(), instanceID, "prune_volumes", nil, req, &resp)
	return &resp, err
}

// GetSystemInfo forwards GetSystemInfo request to the agent
func (f *AgentForwarderV2) GetSystemInfo(instanceID uuid.UUID, req *pb.GetSystemInfoRequest) (*pb.GetSystemInfoResponse, error) {
	var resp pb.GetSystemInfoResponse
	err := f.executeTask(context.Background(), instanceID, "get_system_info", nil, nil, &resp)
	return &resp, err
}

// GetDockerInfo gets Docker daemon information for an instance
func (f *AgentForwarderV2) GetDockerInfo(instanceID uuid.UUID) (*pb.GetSystemInfoResponse, error) {
	var resp pb.GetSystemInfoResponse
	err := f.executeTask(context.Background(), instanceID, "get_docker_info", nil, nil, &resp)
	return &resp, err
}

// GetVersion forwards GetVersion request to the agent
func (f *AgentForwarderV2) GetVersion(instanceID uuid.UUID, req *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	var resp pb.GetVersionResponse
	err := f.executeTask(context.Background(), instanceID, "get_version", nil, nil, &resp)
	return &resp, err
}

// GetDiskUsage forwards GetDiskUsage request to the agent
func (f *AgentForwarderV2) GetDiskUsage(instanceID uuid.UUID, req *pb.GetDiskUsageRequest) (*pb.GetDiskUsageResponse, error) {
	var resp pb.GetDiskUsageResponse
	err := f.executeTask(context.Background(), instanceID, "get_disk_usage", nil, nil, &resp)
	return &resp, err
}

// Ping forwards Ping request to the agent
func (f *AgentForwarderV2) Ping(instanceID uuid.UUID, req *pb.PingRequest) (*pb.PingResponse, error) {
	var resp pb.PingResponse
	err := f.executeTask(context.Background(), instanceID, "ping", nil, nil, &resp)
	return &resp, err
}

// DeleteImage forwards DeleteImage request to the agent
func (f *AgentForwarderV2) DeleteImage(instanceID uuid.UUID, req *pb.DeleteImageRequest) (*pb.DeleteImageResponse, error) {
	var resp pb.DeleteImageResponse
	err := f.executeTask(context.Background(), instanceID, "delete_image", nil, req, &resp)
	return &resp, err
}

// TagImage forwards TagImage request to the agent
func (f *AgentForwarderV2) TagImage(instanceID uuid.UUID, req *pb.TagImageRequest) (*pb.TagImageResponse, error) {
	var resp pb.TagImageResponse
	err := f.executeTask(context.Background(), instanceID, "tag_image", nil, req, &resp)
	return &resp, err
}

// Streaming operations are not supported in task queue mode
// These methods return errors explaining the limitation

// GetContainerLogs is not supported in task queue mode (streaming operation)
func (f *AgentForwarderV2) GetContainerLogs(instanceID uuid.UUID, req *pb.GetContainerLogsRequest) (pb.DockerService_GetContainerLogsClient, error) {
	return nil, fmt.Errorf("GetContainerLogs streaming operation is not supported in task queue mode - requires direct agent connection")
}

// GetContainerStats is not supported in task queue mode (streaming operation)
func (f *AgentForwarderV2) GetContainerStats(instanceID uuid.UUID, req *pb.GetContainerStatsRequest) (pb.DockerService_GetContainerStatsClient, error) {
	return nil, fmt.Errorf("GetContainerStats streaming operation is not supported in task queue mode - requires direct agent connection")
}

// GetEvents is not supported in task queue mode (streaming operation)
func (f *AgentForwarderV2) GetEvents(instanceID uuid.UUID, req *pb.GetEventsRequest) (pb.DockerService_GetEventsClient, error) {
	return nil, fmt.Errorf("GetEvents streaming operation is not supported in task queue mode - requires direct agent connection")
}

// ExecContainer is not supported in task queue mode (streaming operation)
func (f *AgentForwarderV2) ExecContainer(instanceID uuid.UUID) (pb.DockerService_ExecContainerClient, error) {
	return nil, fmt.Errorf("ExecContainer streaming operation is not supported in task queue mode - requires direct agent connection")
}

// Note: Streaming operations (GetContainerStats, GetContainerLogs, PullImage, ExecContainer, GetEvents)
// are not supported in task queue mode. These need to remain using direct gRPC connections.
// The old AgentForwarder will be kept for these streaming operations only.
