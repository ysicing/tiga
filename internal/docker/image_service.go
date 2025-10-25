package docker

import (
	"context"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ListImages implements the ListImages RPC method
func (s *DockerService) ListImages(ctx context.Context, req *pb.ListImagesRequest) (*pb.ListImagesResponse, error) {
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
	options := image.ListOptions{
		All:     req.All,
		Filters: filterArgs,
	}

	// Call Docker API to list images
	images, err := s.dockerClient.Client().ImageList(ctx, options)
	if err != nil {
		return nil, err
	}

	// Convert Docker API types to protobuf types
	pbImages := make([]*pb.Image, len(images))
	for i, img := range images {
		pbImages[i] = convertImageToProto(&img)
	}

	return &pb.ListImagesResponse{
		Images: pbImages,
	}, nil
}

// GetImage implements the GetImage RPC method
func (s *DockerService) GetImage(ctx context.Context, req *pb.GetImageRequest) (*pb.GetImageResponse, error) {
	// Call Docker API to inspect image
	imageInspect, _, err := s.dockerClient.Client().ImageInspectWithRaw(ctx, req.ImageId)
	if err != nil {
		return nil, err
	}

	// Convert to protobuf ImageDetail
	pbImageDetail := convertImageDetailToProto(&imageInspect)

	return &pb.GetImageResponse{
		Image: pbImageDetail,
	}, nil
}

// DeleteImage implements the DeleteImage RPC method
func (s *DockerService) DeleteImage(ctx context.Context, req *pb.DeleteImageRequest) (*pb.DeleteImageResponse, error) {
	// Build Docker API options
	options := image.RemoveOptions{
		Force:         req.Force,
		PruneChildren: !req.NoPrune,
	}

	// Call Docker API to remove image
	deleteResponses, err := s.dockerClient.Client().ImageRemove(ctx, req.ImageId, options)
	if err != nil {
		return nil, err
	}

	// Convert to protobuf
	pbDeleted := make([]*pb.ImageDeleteResponse, len(deleteResponses))
	for i, resp := range deleteResponses {
		pbDeleted[i] = &pb.ImageDeleteResponse{
			Untagged: resp.Untagged,
			Deleted:  resp.Deleted,
		}
	}

	return &pb.DeleteImageResponse{
		Deleted: pbDeleted,
	}, nil
}

// PullImage implements the PullImage streaming RPC method
func (s *DockerService) PullImage(req *pb.PullImageRequest, stream pb.DockerService_PullImageServer) error {
	ctx := stream.Context()

	// Build Docker API options
	options := image.PullOptions{}
	if req.RegistryAuth != "" {
		options.RegistryAuth = req.RegistryAuth
	}

	// Call Docker API to pull image
	reader, err := s.dockerClient.Client().ImagePull(ctx, req.Image, options)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Read and stream pull progress
	decoder := json.NewDecoder(reader)
	for {
		var progress PullProgress
		if err := decoder.Decode(&progress); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Convert to protobuf and send
		pbProgress := &pb.PullImageProgress{
			Status:   progress.Status,
			Progress: progress.Progress,
			Id:       progress.ID,
		}

		// Parse current and total bytes if available
		if progress.ProgressDetail.Current > 0 {
			pbProgress.Current = int64(progress.ProgressDetail.Current)
			pbProgress.Total = int64(progress.ProgressDetail.Total)
		}

		if err := stream.Send(pbProgress); err != nil {
			return err
		}
	}
}

// TagImage implements the TagImage RPC method
func (s *DockerService) TagImage(ctx context.Context, req *pb.TagImageRequest) (*pb.TagImageResponse, error) {
	// Call Docker API to tag image
	err := s.dockerClient.Client().ImageTag(ctx, req.Source, req.Target)
	if err != nil {
		return &pb.TagImageResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.TagImageResponse{
		Success: true,
		Message: "Image tagged successfully",
	}, nil
}

// convertImageToProto converts Docker API Image type to protobuf Image
func convertImageToProto(img *image.Summary) *pb.Image {
	return &pb.Image{
		Id:          img.ID[:12], // Short ID (12 chars)
		RepoTags:    img.RepoTags,
		RepoDigests: img.RepoDigests,
		ParentId:    img.ParentID,
		Created:     img.Created,
		Size:        img.Size,
		VirtualSize: img.VirtualSize,
		SharedSize:  img.SharedSize,
		Labels:      img.Labels,
		Containers:  int32(img.Containers),
	}
}

// convertImageDetailToProto converts Docker API ImageInspect to protobuf ImageDetail
func convertImageDetailToProto(img *types.ImageInspect) *pb.ImageDetail {
	detail := &pb.ImageDetail{
		Id:            img.ID,
		RepoTags:      img.RepoTags,
		RepoDigests:   img.RepoDigests,
		Parent:        img.Parent,
		Comment:       img.Comment,
		Created:       0, // Set to 0 if unable to parse timestamp
		Container:     img.Container,
		DockerVersion: img.DockerVersion,
		Author:        img.Author,
		Architecture:  img.Architecture,
		Os:            img.Os,
		Size:          img.Size,
		VirtualSize:   img.VirtualSize,
	}

	// Convert Config
	if img.Config != nil {
		detail.Config = &pb.ImageConfig{
			Hostname:     img.Config.Hostname,
			Domainname:   img.Config.Domainname,
			User:         img.Config.User,
			Env:          img.Config.Env,
			Cmd:          img.Config.Cmd,
			Image:        img.Config.Image,
			WorkingDir:   img.Config.WorkingDir,
			Entrypoint:   img.Config.Entrypoint,
			Labels:       img.Config.Labels,
			Volumes:      make(map[string]bool),
			ExposedPorts: make(map[string]bool),
		}

		// Convert Volumes
		for vol := range img.Config.Volumes {
			detail.Config.Volumes[vol] = true
		}

		// Convert ExposedPorts
		for port := range img.Config.ExposedPorts {
			detail.Config.ExposedPorts[string(port)] = true
		}
	}

	// Convert RootFS
	detail.RootFs = &pb.RootFS{
		Type:      img.RootFS.Type,
		Layers:    img.RootFS.Layers,
		BaseLayer: "", // BaseLayer is not available in Docker SDK types
	}

	return detail
}

// PullProgress represents the progress structure from Docker pull output
type PullProgress struct {
	Status         string `json:"status"`
	Progress       string `json:"progress"`
	ID             string `json:"id"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
}
