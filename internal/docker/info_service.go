package docker

import (
	"context"

	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// GetDockerInfo implements the GetDockerInfo RPC method
// It retrieves comprehensive information about the Docker daemon
func (s *DockerService) GetDockerInfo(ctx context.Context, req *pb.GetDockerInfoRequest) (*pb.GetDockerInfoResponse, error) {
	// Get Docker version information
	version, err := s.dockerClient.Client().ServerVersion(ctx)
	if err != nil {
		return nil, err
	}

	// Get Docker daemon information (containers, images, system resources)
	info, err := s.dockerClient.Client().Info(ctx)
	if err != nil {
		return nil, err
	}

	// Construct response from Docker API data
	response := &pb.GetDockerInfoResponse{
		Version:           version.Version,
		ApiVersion:        version.APIVersion,
		MinApiVersion:     version.MinAPIVersion,
		GitCommit:         version.GitCommit,
		GoVersion:         version.GoVersion,
		Os:                version.Os,
		Arch:              version.Arch,
		KernelVersion:     version.KernelVersion,
		StorageDriver:     info.Driver,
		Containers:        int32(info.Containers),
		ContainersRunning: int32(info.ContainersRunning),
		ContainersPaused:  int32(info.ContainersPaused),
		ContainersStopped: int32(info.ContainersStopped),
		Images:            int32(info.Images),
		MemTotal:          info.MemTotal,
		NCpu:              int32(info.NCPU),
		OperatingSystem:   info.OperatingSystem,
	}

	return response, nil
}
