package docker

import (
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ListContainers implements the ListContainers RPC method
// It retrieves a paginated list of containers with optional filtering
func (s *DockerService) ListContainers(ctx context.Context, req *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	// Parse Docker filters from JSON string
	filterArgs := filters.NewArgs()
	if req.Filters != "" {
		var filterMap map[string][]string
		if err := json.Unmarshal([]byte(req.Filters), &filterMap); err == nil {
			for key, values := range filterMap {
				for _, value := range values {
					filterArgs.Add(key, value)
				}
			}
		}
	}

	// Build Docker API options
	options := container.ListOptions{
		All:     req.All,
		Filters: filterArgs,
	}

	// Call Docker API to list containers
	containers, err := s.dockerClient.Client().ContainerList(ctx, options)
	if err != nil {
		return nil, err
	}

	// Calculate pagination
	total := int32(len(containers))
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 50 // Default page size
	}

	page := req.Page
	if page < 1 {
		page = 1
	}

	// Calculate start and end indices
	start := (page - 1) * pageSize
	end := start + pageSize

	// Validate pagination boundaries
	if start >= total {
		return &pb.ListContainersResponse{
			Containers: []*pb.Container{},
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
		}, nil
	}

	if end > total {
		end = total
	}

	// Slice containers for current page
	pageContainers := containers[start:end]

	// Convert Docker API types to protobuf types
	pbContainers := make([]*pb.Container, len(pageContainers))
	for i, c := range pageContainers {
		pbContainers[i] = convertContainerToProto(&c)
	}

	return &pb.ListContainersResponse{
		Containers: pbContainers,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// convertContainerToProto converts Docker API Container type to protobuf Container
func convertContainerToProto(c *types.Container) *pb.Container {
	container := &pb.Container{
		Id:         c.ID[:12], // Short ID (12 chars)
		Names:      c.Names,
		Image:      c.Image,
		ImageId:    c.ImageID,
		Command:    c.Command,
		Created:    c.Created,
		State:      c.State,
		Status:     c.Status,
		Ports:      convertPortsToProto(c.Ports),
		Labels:     c.Labels,
		SizeRw:     c.SizeRw,
		SizeRootFs: c.SizeRootFs,
		Mounts:     convertMountsToProto(c.Mounts),
		Networks:   make(map[string]*pb.NetworkConfig),
	}

	// Convert networks
	for name, network := range c.NetworkSettings.Networks {
		container.Networks[name] = &pb.NetworkConfig{
			NetworkId:   network.NetworkID,
			EndpointId:  network.EndpointID,
			Gateway:     network.Gateway,
			IpAddress:   network.IPAddress,
			IpPrefixLen: int32(network.IPPrefixLen),
			MacAddress:  network.MacAddress,
		}
	}

	return container
}

// convertPortsToProto converts Docker API Port array to protobuf Port array
func convertPortsToProto(ports []types.Port) []*pb.Port {
	pbPorts := make([]*pb.Port, len(ports))
	for i, port := range ports {
		pbPorts[i] = &pb.Port{
			Ip:          port.IP,
			PrivatePort: int32(port.PrivatePort),
			PublicPort:  int32(port.PublicPort),
			Type:        port.Type,
		}
	}
	return pbPorts
}

// convertMountsToProto converts Docker API MountPoint array to protobuf Mount array
func convertMountsToProto(mounts []types.MountPoint) []*pb.Mount {
	pbMounts := make([]*pb.Mount, len(mounts))
	for i, mount := range mounts {
		pbMounts[i] = &pb.Mount{
			Type:        string(mount.Type),
			Source:      mount.Source,
			Destination: mount.Destination,
			Mode:        mount.Mode,
			Rw:          mount.RW,
			Propagation: string(mount.Propagation),
		}
	}
	return pbMounts
}

// StartContainer implements the StartContainer RPC method
func (s *DockerService) StartContainer(ctx context.Context, req *pb.StartContainerRequest) (*pb.StartContainerResponse, error) {
	err := s.dockerClient.Client().ContainerStart(ctx, req.ContainerId, container.StartOptions{})
	if err != nil {
		return &pb.StartContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.StartContainerResponse{
		Success: true,
		Message: "Container started successfully",
	}, nil
}

// StopContainer implements the StopContainer RPC method
func (s *DockerService) StopContainer(ctx context.Context, req *pb.StopContainerRequest) (*pb.StopContainerResponse, error) {
	// Set default timeout if not specified
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 10 // Default 10 seconds
	}

	// Convert int32 to int for Docker SDK
	timeoutInt := int(timeout)
	stopOptions := container.StopOptions{
		Timeout: &timeoutInt,
	}

	err := s.dockerClient.Client().ContainerStop(ctx, req.ContainerId, stopOptions)
	if err != nil {
		return &pb.StopContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.StopContainerResponse{
		Success: true,
		Message: "Container stopped successfully",
	}, nil
}

// RestartContainer implements the RestartContainer RPC method
func (s *DockerService) RestartContainer(ctx context.Context, req *pb.RestartContainerRequest) (*pb.RestartContainerResponse, error) {
	// Set default timeout if not specified
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 10 // Default 10 seconds
	}

	// Convert int32 to int for Docker SDK
	timeoutInt := int(timeout)
	stopOptions := container.StopOptions{
		Timeout: &timeoutInt,
	}

	err := s.dockerClient.Client().ContainerRestart(ctx, req.ContainerId, stopOptions)
	if err != nil {
		return &pb.RestartContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.RestartContainerResponse{
		Success: true,
		Message: "Container restarted successfully",
	}, nil
}

// PauseContainer implements the PauseContainer RPC method
func (s *DockerService) PauseContainer(ctx context.Context, req *pb.PauseContainerRequest) (*pb.PauseContainerResponse, error) {
	err := s.dockerClient.Client().ContainerPause(ctx, req.ContainerId)
	if err != nil {
		return &pb.PauseContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.PauseContainerResponse{
		Success: true,
		Message: "Container paused successfully",
	}, nil
}

// UnpauseContainer implements the UnpauseContainer RPC method
func (s *DockerService) UnpauseContainer(ctx context.Context, req *pb.UnpauseContainerRequest) (*pb.UnpauseContainerResponse, error) {
	err := s.dockerClient.Client().ContainerUnpause(ctx, req.ContainerId)
	if err != nil {
		return &pb.UnpauseContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.UnpauseContainerResponse{
		Success: true,
		Message: "Container unpaused successfully",
	}, nil
}

// DeleteContainer implements the DeleteContainer RPC method
func (s *DockerService) DeleteContainer(ctx context.Context, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error) {
	removeOptions := container.RemoveOptions{
		Force:         req.Force,
		RemoveVolumes: req.RemoveVolumes,
	}

	err := s.dockerClient.Client().ContainerRemove(ctx, req.ContainerId, removeOptions)
	if err != nil {
		return &pb.DeleteContainerResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.DeleteContainerResponse{
		Success: true,
		Message: "Container deleted successfully",
	}, nil
}
