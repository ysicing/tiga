package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/docker"
	"github.com/ysicing/tiga/proto"

	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// DockerTaskHandler handles Docker operation tasks sent by the server
type DockerTaskHandler struct {
	dockerClient  *docker.DockerClient
	dockerService *docker.DockerService
}

// NewDockerTaskHandler creates a new Docker task handler
func NewDockerTaskHandler() (*DockerTaskHandler, error) {
	// Initialize Docker client
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}

	// Test Docker connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dockerClient.Ping(ctx); err != nil {
		dockerClient.Close()
		return nil, err
	}

	// Create Docker service
	dockerService := docker.NewDockerService(dockerClient)

	logrus.WithFields(logrus.Fields{
		"docker_version": dockerClient.DockerVersion(),
		"api_version":    dockerClient.APIVersion(),
	}).Info("Docker task handler initialized")

	return &DockerTaskHandler{
		dockerClient:  dockerClient,
		dockerService: dockerService,
	}, nil
}

// Close closes the Docker client
func (h *DockerTaskHandler) Close() {
	if h.dockerClient != nil {
		h.dockerClient.Close()
	}
}

// HandleDockerTask executes a Docker operation task and returns the result
func (h *DockerTaskHandler) HandleDockerTask(task *proto.AgentTask) *proto.TaskResult {
	startTime := time.Now()

	// Parse operation type from params
	operation, ok := task.Params["operation"]
	if !ok {
		return &proto.TaskResult{
			TaskId:    task.TaskId,
			Success:   false,
			Error:     "missing 'operation' parameter",
			Timestamp: time.Now().UnixMilli(),
		}
	}

	logrus.Infof("[DockerTask] Executing operation: %s, task_id: %s", operation, task.TaskId)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result interface{}
	var err error

	// Execute operation based on type
	switch operation {
	case "list_containers":
		result, err = h.listContainers(ctx, task)
	case "get_container":
		result, err = h.getContainer(ctx, task)
	case "start_container":
		result, err = h.startContainer(ctx, task)
	case "stop_container":
		result, err = h.stopContainer(ctx, task)
	case "restart_container":
		result, err = h.restartContainer(ctx, task)
	case "pause_container":
		result, err = h.pauseContainer(ctx, task)
	case "unpause_container":
		result, err = h.unpauseContainer(ctx, task)
	case "delete_container":
		result, err = h.deleteContainer(ctx, task)
	case "list_images":
		result, err = h.listImages(ctx, task)
	case "get_image":
		result, err = h.getImage(ctx, task)
	case "delete_image":
		result, err = h.deleteImage(ctx, task)
	case "tag_image":
		result, err = h.tagImage(ctx, task)
	case "list_networks":
		result, err = h.listNetworks(ctx, task)
	case "get_network":
		result, err = h.getNetwork(ctx, task)
	case "create_network":
		result, err = h.createNetwork(ctx, task)
	case "delete_network":
		result, err = h.deleteNetwork(ctx, task)
	case "connect_network":
		result, err = h.connectNetwork(ctx, task)
	case "disconnect_network":
		result, err = h.disconnectNetwork(ctx, task)
	case "list_volumes":
		result, err = h.listVolumes(ctx, task)
	case "get_volume":
		result, err = h.getVolume(ctx, task)
	case "create_volume":
		result, err = h.createVolume(ctx, task)
	case "delete_volume":
		result, err = h.deleteVolume(ctx, task)
	case "prune_volumes":
		result, err = h.pruneVolumes(ctx, task)
	case "get_system_info":
		result, err = h.getSystemInfo(ctx, task)
	case "get_docker_info":
		result, err = h.getDockerInfo(ctx, task)
	case "get_version":
		result, err = h.getVersion(ctx, task)
	case "get_disk_usage":
		result, err = h.getDiskUsage(ctx, task)
	case "ping":
		result, err = h.ping(ctx, task)
	default:
		return &proto.TaskResult{
			TaskId:    task.TaskId,
			Success:   false,
			Error:     "unknown operation: " + operation,
			Timestamp: time.Now().UnixMilli(),
		}
	}

	duration := time.Since(startTime)

	// Build task result
	taskResult := &proto.TaskResult{
		TaskId:    task.TaskId,
		Timestamp: time.Now().UnixMilli(),
	}

	if err != nil {
		logrus.WithError(err).Errorf("[DockerTask] Operation failed: %s (duration: %v)", operation, duration)
		taskResult.Success = false
		taskResult.Error = err.Error()
	} else {
		logrus.Infof("[DockerTask] Operation succeeded: %s (duration: %v)", operation, duration)
		taskResult.Success = true

		// Marshal result to JSON
		if result != nil {
			payload, err := json.Marshal(result)
			if err != nil {
				taskResult.Success = false
				taskResult.Error = "failed to marshal result: " + err.Error()
			} else {
				taskResult.Payload = payload
			}
		}
	}

	return taskResult
}

// Container operations
func (h *DockerTaskHandler) listContainers(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	// Parse request from payload
	var req pb.ListContainersRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.ListContainers(ctx, &req)
	return resp, err
}

func (h *DockerTaskHandler) getContainer(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	containerID, ok := task.Params["container_id"]
	if !ok {
		return nil, ErrMissingParameter("container_id")
	}

	req := &pb.GetContainerRequest{ContainerId: containerID}
	resp, err := h.dockerService.GetContainer(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) startContainer(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	containerID, ok := task.Params["container_id"]
	if !ok {
		return nil, ErrMissingParameter("container_id")
	}

	req := &pb.StartContainerRequest{ContainerId: containerID}
	resp, err := h.dockerService.StartContainer(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) stopContainer(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	containerID, ok := task.Params["container_id"]
	if !ok {
		return nil, ErrMissingParameter("container_id")
	}

	req := &pb.StopContainerRequest{ContainerId: containerID}
	resp, err := h.dockerService.StopContainer(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) restartContainer(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	containerID, ok := task.Params["container_id"]
	if !ok {
		return nil, ErrMissingParameter("container_id")
	}

	req := &pb.RestartContainerRequest{ContainerId: containerID}
	resp, err := h.dockerService.RestartContainer(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) pauseContainer(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	containerID, ok := task.Params["container_id"]
	if !ok {
		return nil, ErrMissingParameter("container_id")
	}

	req := &pb.PauseContainerRequest{ContainerId: containerID}
	resp, err := h.dockerService.PauseContainer(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) unpauseContainer(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	containerID, ok := task.Params["container_id"]
	if !ok {
		return nil, ErrMissingParameter("container_id")
	}

	req := &pb.UnpauseContainerRequest{ContainerId: containerID}
	resp, err := h.dockerService.UnpauseContainer(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) deleteContainer(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	containerID, ok := task.Params["container_id"]
	if !ok {
		return nil, ErrMissingParameter("container_id")
	}

	req := &pb.DeleteContainerRequest{ContainerId: containerID}
	resp, err := h.dockerService.DeleteContainer(ctx, req)
	return resp, err
}

// Image operations
func (h *DockerTaskHandler) listImages(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.ListImagesRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.ListImages(ctx, &req)
	return resp, err
}

func (h *DockerTaskHandler) getImage(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	imageID, ok := task.Params["image_id"]
	if !ok {
		return nil, ErrMissingParameter("image_id")
	}

	req := &pb.GetImageRequest{ImageId: imageID}
	resp, err := h.dockerService.GetImage(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) deleteImage(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.DeleteImageRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.DeleteImage(ctx, &req)
	return resp, err
}

func (h *DockerTaskHandler) tagImage(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.TagImageRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.TagImage(ctx, &req)
	return resp, err
}

// Network operations
func (h *DockerTaskHandler) listNetworks(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.ListNetworksRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.ListNetworks(ctx, &req)
	return resp, err
}

func (h *DockerTaskHandler) getNetwork(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	networkID, ok := task.Params["network_id"]
	if !ok {
		return nil, ErrMissingParameter("network_id")
	}

	req := &pb.GetNetworkRequest{NetworkId: networkID}
	resp, err := h.dockerService.GetNetwork(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) createNetwork(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.CreateNetworkRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.CreateNetwork(ctx, &req)
	return resp, err
}

func (h *DockerTaskHandler) deleteNetwork(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	networkID, ok := task.Params["network_id"]
	if !ok {
		return nil, ErrMissingParameter("network_id")
	}

	req := &pb.DeleteNetworkRequest{NetworkId: networkID}
	resp, err := h.dockerService.DeleteNetwork(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) connectNetwork(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.ConnectNetworkRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.ConnectNetwork(ctx, &req)
	return resp, err
}

func (h *DockerTaskHandler) disconnectNetwork(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.DisconnectNetworkRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.DisconnectNetwork(ctx, &req)
	return resp, err
}

// Volume operations
func (h *DockerTaskHandler) listVolumes(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.ListVolumesRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.ListVolumes(ctx, &req)
	return resp, err
}

func (h *DockerTaskHandler) getVolume(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	volumeName, ok := task.Params["volume_name"]
	if !ok {
		return nil, ErrMissingParameter("volume_name")
	}

	req := &pb.GetVolumeRequest{Name: volumeName}
	resp, err := h.dockerService.GetVolume(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) createVolume(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.CreateVolumeRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.CreateVolume(ctx, &req)
	return resp, err
}

func (h *DockerTaskHandler) deleteVolume(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	volumeName, ok := task.Params["volume_name"]
	if !ok {
		return nil, ErrMissingParameter("volume_name")
	}

	req := &pb.DeleteVolumeRequest{Name: volumeName}
	resp, err := h.dockerService.DeleteVolume(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) pruneVolumes(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	var req pb.PruneVolumesRequest
	if len(task.Payload) > 0 {
		if err := json.Unmarshal(task.Payload, &req); err != nil {
			return nil, err
		}
	}

	resp, err := h.dockerService.PruneVolumes(ctx, &req)
	return resp, err
}

// System operations
func (h *DockerTaskHandler) getSystemInfo(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	req := &pb.GetSystemInfoRequest{}
	resp, err := h.dockerService.GetSystemInfo(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) getDockerInfo(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	req := &pb.GetSystemInfoRequest{}
	resp, err := h.dockerService.GetSystemInfo(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) getVersion(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	req := &pb.GetVersionRequest{}
	resp, err := h.dockerService.GetVersion(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) getDiskUsage(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	req := &pb.GetDiskUsageRequest{}
	resp, err := h.dockerService.GetDiskUsage(ctx, req)
	return resp, err
}

func (h *DockerTaskHandler) ping(ctx context.Context, task *proto.AgentTask) (interface{}, error) {
	req := &pb.PingRequest{}
	resp, err := h.dockerService.Ping(ctx, req)
	return resp, err
}

// ErrMissingParameter creates a parameter missing error
func ErrMissingParameter(param string) error {
	return &MissingParameterError{Param: param}
}

// MissingParameterError represents a missing parameter error
type MissingParameterError struct {
	Param string
}

func (e *MissingParameterError) Error() string {
	return "missing required parameter: " + e.Param
}
