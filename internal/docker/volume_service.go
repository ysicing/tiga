package docker

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"

	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ListVolumes implements the ListVolumes RPC method
// It retrieves a list of Docker volumes
func (s *DockerService) ListVolumes(ctx context.Context, req *pb.ListVolumesRequest) (*pb.ListVolumesResponse, error) {
	// Build Docker API options (no filters in proto definition)
	options := volume.ListOptions{}

	// Call Docker API to list volumes
	volumesResponse, err := s.dockerClient.Client().VolumeList(ctx, options)
	if err != nil {
		return nil, err
	}

	// Convert Docker volumes to protobuf format
	pbVolumes := make([]*pb.Volume, 0)
	if volumesResponse.Volumes != nil {
		pbVolumes = make([]*pb.Volume, len(volumesResponse.Volumes))
		for i, vol := range volumesResponse.Volumes {
			pbVolume := &pb.Volume{
				Name:       vol.Name,
				Driver:     vol.Driver,
				Mountpoint: vol.Mountpoint,
				Scope:      vol.Scope,
				Labels:     vol.Labels,
				Options:    vol.Options,
			}

			// Parse CreatedAt timestamp (Docker returns RFC3339 format)
			if vol.CreatedAt != "" {
				// For simplicity, store as string in Status field since proto expects string
				// In production, you'd parse to Unix timestamp
				pbVolume.Status = vol.CreatedAt
			}

			// Add UsageData if available
			if vol.UsageData != nil {
				pbVolume.UsageData = &pb.VolumeUsageData{
					Size:     vol.UsageData.Size,
					RefCount: vol.UsageData.RefCount,
				}
			}

			pbVolumes[i] = pbVolume
		}
	}

	return &pb.ListVolumesResponse{
		Volumes:  pbVolumes,
		Warnings: volumesResponse.Warnings,
	}, nil
}

// GetVolume implements the GetVolume RPC method
// It retrieves detailed information about a specific Docker volume
func (s *DockerService) GetVolume(ctx context.Context, req *pb.GetVolumeRequest) (*pb.GetVolumeResponse, error) {
	// Inspect the volume
	volumeInfo, err := s.dockerClient.Client().VolumeInspect(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	// Convert to protobuf format
	pbVolume := &pb.Volume{
		Name:       volumeInfo.Name,
		Driver:     volumeInfo.Driver,
		Mountpoint: volumeInfo.Mountpoint,
		Scope:      volumeInfo.Scope,
		Labels:     volumeInfo.Labels,
		Options:    volumeInfo.Options,
	}

	// Store CreatedAt in Status field
	if volumeInfo.CreatedAt != "" {
		pbVolume.Status = volumeInfo.CreatedAt
	}

	// Add UsageData if available
	if volumeInfo.UsageData != nil {
		pbVolume.UsageData = &pb.VolumeUsageData{
			Size:     volumeInfo.UsageData.Size,
			RefCount: volumeInfo.UsageData.RefCount,
		}
	}

	return &pb.GetVolumeResponse{
		Volume: pbVolume,
	}, nil
}

// CreateVolume implements the CreateVolume RPC method
// It creates a new Docker volume
func (s *DockerService) CreateVolume(ctx context.Context, req *pb.CreateVolumeRequest) (*pb.CreateVolumeResponse, error) {
	// Build volume create options
	createOptions := volume.CreateOptions{
		Name:       req.Name,
		Driver:     req.Driver,
		DriverOpts: req.DriverOpts,
		Labels:     req.Labels,
	}

	// Create the volume
	volumeInfo, err := s.dockerClient.Client().VolumeCreate(ctx, createOptions)
	if err != nil {
		return nil, err
	}

	// Convert response to protobuf
	pbVolume := &pb.Volume{
		Name:       volumeInfo.Name,
		Driver:     volumeInfo.Driver,
		Mountpoint: volumeInfo.Mountpoint,
		Scope:      volumeInfo.Scope,
		Labels:     volumeInfo.Labels,
		Options:    volumeInfo.Options,
	}

	// Store CreatedAt in Status field
	if volumeInfo.CreatedAt != "" {
		pbVolume.Status = volumeInfo.CreatedAt
	}

	// Add UsageData if available
	if volumeInfo.UsageData != nil {
		pbVolume.UsageData = &pb.VolumeUsageData{
			Size:     volumeInfo.UsageData.Size,
			RefCount: volumeInfo.UsageData.RefCount,
		}
	}

	return &pb.CreateVolumeResponse{
		Volume: pbVolume,
	}, nil
}

// DeleteVolume implements the DeleteVolume RPC method
// It removes a Docker volume
func (s *DockerService) DeleteVolume(ctx context.Context, req *pb.DeleteVolumeRequest) (*pb.DeleteVolumeResponse, error) {
	// Remove the volume
	err := s.dockerClient.Client().VolumeRemove(ctx, req.Name, req.Force)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteVolumeResponse{
		Success: true,
	}, nil
}

// PruneVolumes implements the PruneVolumes RPC method
// It removes all unused Docker volumes
func (s *DockerService) PruneVolumes(ctx context.Context, req *pb.PruneVolumesRequest) (*pb.PruneVolumesResponse, error) {
	// Parse Docker filters from map
	filterArgs := filters.NewArgs()
	if req.Filters != nil && len(req.Filters) > 0 {
		for key, value := range req.Filters {
			filterArgs.Add(key, value)
		}
	}

	// Prune volumes
	report, err := s.dockerClient.Client().VolumesPrune(ctx, filterArgs)
	if err != nil {
		return nil, err
	}

	// Convert response
	return &pb.PruneVolumesResponse{
		VolumesDeleted: report.VolumesDeleted,
		SpaceReclaimed: report.SpaceReclaimed,
	}, nil
}
