package docker

import (
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// DockerService implements the Docker gRPC service
type DockerService struct {
	pb.UnimplementedDockerServiceServer
	dockerClient *DockerClient
}

// NewDockerService creates a new instance of DockerService
func NewDockerService(dockerClient *DockerClient) *DockerService {
	return &DockerService{
		dockerClient: dockerClient,
	}
}
