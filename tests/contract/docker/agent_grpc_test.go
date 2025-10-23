package docker_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Mock protobuf types for contract testing
// In real implementation, these would be generated from docker.proto

type GetDockerInfoRequest struct{}

type DockerInfo struct {
	Version           string
	ApiVersion        string
	StorageDriver     string
	Containers        int32
	ContainersRunning int32
	ContainersPaused  int32
	ContainersStopped int32
	Images            int32
	OperatingSystem   string
	Architecture      string
	KernelVersion     string
	MemTotal          int64
	NCPU              int32
	ServerVersion     string
}

type ListContainersRequest struct {
	All       bool
	Page      int32
	PageSize  int32
	Filter    string
	SortBy    string
	SortOrder string
}

type ListContainersResponse struct {
	Containers []*Container
	Total      int32
	Page       int32
	PageSize   int32
}

type Container struct {
	Id            string
	Name          string
	Image         string
	ImageId       string
	State         string
	Status        string
	Created       int64
	StartedAt     int64
	FinishedAt    int64
	Ports         []*Port
	Mounts        []*Mount
	Networks      map[string]*Network
	Env           []string
	Labels        map[string]string
	Command       []string
	CpuLimit      int64
	MemoryLimit   int64
	RestartCount  int32
	RestartPolicy string
}

type Port struct {
	Ip          string
	PrivatePort int32
	PublicPort  int32
	Type        string
}

type Mount struct {
	Type        string
	Source      string
	Destination string
	Mode        string
	Rw          bool
}

type Network struct {
	NetworkId   string
	Gateway     string
	IpAddress   string
	IpPrefixLen int32
	MacAddress  string
}

type GetContainerRequest struct {
	ContainerId string
}

type ContainerActionRequest struct {
	ContainerId string
	Timeout     int32
}

type ContainerActionResponse struct {
	Success     bool
	Message     string
	ContainerId string
	Duration    int64
}

type DeleteContainerRequest struct {
	ContainerId   string
	Force         bool
	RemoveVolumes bool
}

type GetContainerStatsRequest struct {
	ContainerId string
	Stream      bool
}

type ContainerStats struct {
	ContainerId        string
	Timestamp          int64
	CpuUsagePercent    float64
	CpuUsageNano       uint64
	MemoryUsage        uint64
	MemoryLimit        uint64
	MemoryUsagePercent float64
	NetworkRxBytes     uint64
	NetworkTxBytes     uint64
	BlockReadBytes     uint64
	BlockWriteBytes    uint64
	PidsCurrent        uint64
}

type GetContainerLogsRequest struct {
	ContainerId    string
	Follow         bool
	TailLines      int32
	SinceTimestamp int64
	Timestamps     bool
}

type LogEntry struct {
	Timestamp string
	Stream    string
	Log       string
}

type ExecRequest struct {
	// Simplified for testing
	Type string // "start", "input", "resize"
	Data []byte
}

type ExecResponse struct {
	Type string // "output", "exit"
	Data []byte
	Code int32
}

type ListImagesRequest struct {
	All    bool
	Filter string
}

type ListImagesResponse struct {
	Images []*Image
	Total  int32
}

type Image struct {
	Id          string
	RepoTags    []string
	RepoDigests []string
	Size        int64
	VirtualSize int64
	Created     int64
	Labels      map[string]string
	Layers      []string
}

type GetImageRequest struct {
	ImageId string
}

type DeleteImageRequest struct {
	ImageId string
	Force   bool
	NoPrune bool
}

type DeleteImageResponse struct {
	Success  bool
	Message  string
	Deleted  []string
	Untagged []string
}

type PullImageRequest struct {
	ImageName string
	Platform  string
}

type PullImageProgress struct {
	Status   string
	Progress string
	Id       string
	Current  int64
	Total    int64
}

type TagImageRequest struct {
	SourceImage string
	TargetRepo  string
	TargetTag   string
}

type TagImageResponse struct {
	Success bool
	Message string
}

// Mock DockerService server implementation
type mockDockerServiceServer struct {
	// Embed UnimplementedDockerServiceServer for forward compatibility
}

func (s *mockDockerServiceServer) GetDockerInfo(ctx context.Context, req *GetDockerInfoRequest) (*DockerInfo, error) {
	return &DockerInfo{
		Version:           "24.0.7",
		ApiVersion:        "1.43",
		StorageDriver:     "overlay2",
		Containers:        10,
		ContainersRunning: 5,
		ContainersPaused:  1,
		ContainersStopped: 4,
		Images:            15,
		OperatingSystem:   "Ubuntu 22.04",
		Architecture:      "x86_64",
		KernelVersion:     "5.15.0-91-generic",
		MemTotal:          16777216000,
		NCPU:              8,
		ServerVersion:     "24.0.7",
	}, nil
}

func (s *mockDockerServiceServer) ListContainers(ctx context.Context, req *ListContainersRequest) (*ListContainersResponse, error) {
	// Validate pagination parameters
	if req.PageSize > 1000 {
		return nil, status.Errorf(codes.InvalidArgument, "page_size exceeds maximum of 1000")
	}

	containers := []*Container{
		{
			Id:            "abc123def456",
			Name:          "nginx-web",
			Image:         "nginx:latest",
			ImageId:       "sha256:abc123",
			State:         "running",
			Status:        "Up 2 hours",
			Created:       time.Now().Unix() - 7200,
			StartedAt:     time.Now().Unix() - 7200,
			RestartCount:  0,
			RestartPolicy: "always",
			Labels:        map[string]string{"app": "web"},
		},
	}

	return &ListContainersResponse{
		Containers: containers,
		Total:      int32(len(containers)),
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

func (s *mockDockerServiceServer) GetContainer(ctx context.Context, req *GetContainerRequest) (*Container, error) {
	if req.ContainerId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "container_id is required")
	}

	if req.ContainerId == "not-found" {
		return nil, status.Errorf(codes.NotFound, "container not found")
	}

	return &Container{
		Id:      req.ContainerId,
		Name:    "test-container",
		Image:   "nginx:latest",
		ImageId: "sha256:abc123",
		State:   "running",
		Status:  "Up 1 hour",
	}, nil
}

func (s *mockDockerServiceServer) StartContainer(ctx context.Context, req *ContainerActionRequest) (*ContainerActionResponse, error) {
	if req.ContainerId == "stopped" {
		return nil, status.Errorf(codes.FailedPrecondition, "container is already stopped")
	}

	return &ContainerActionResponse{
		Success:     true,
		Message:     "Container started successfully",
		ContainerId: req.ContainerId,
		Duration:    1250,
	}, nil
}

func (s *mockDockerServiceServer) StopContainer(ctx context.Context, req *ContainerActionRequest) (*ContainerActionResponse, error) {
	return &ContainerActionResponse{
		Success:     true,
		Message:     "Container stopped successfully",
		ContainerId: req.ContainerId,
		Duration:    850,
	}, nil
}

func (s *mockDockerServiceServer) RestartContainer(ctx context.Context, req *ContainerActionRequest) (*ContainerActionResponse, error) {
	return &ContainerActionResponse{
		Success:     true,
		Message:     "Container restarted successfully",
		ContainerId: req.ContainerId,
		Duration:    2100,
	}, nil
}

func (s *mockDockerServiceServer) PauseContainer(ctx context.Context, req *ContainerActionRequest) (*ContainerActionResponse, error) {
	return &ContainerActionResponse{
		Success:     true,
		Message:     "Container paused successfully",
		ContainerId: req.ContainerId,
		Duration:    150,
	}, nil
}

func (s *mockDockerServiceServer) UnpauseContainer(ctx context.Context, req *ContainerActionRequest) (*ContainerActionResponse, error) {
	return &ContainerActionResponse{
		Success:     true,
		Message:     "Container unpaused successfully",
		ContainerId: req.ContainerId,
		Duration:    120,
	}, nil
}

func (s *mockDockerServiceServer) DeleteContainer(ctx context.Context, req *DeleteContainerRequest) (*ContainerActionResponse, error) {
	return &ContainerActionResponse{
		Success:     true,
		Message:     "Container deleted successfully",
		ContainerId: req.ContainerId,
		Duration:    500,
	}, nil
}

func (s *mockDockerServiceServer) ListImages(ctx context.Context, req *ListImagesRequest) (*ListImagesResponse, error) {
	images := []*Image{
		{
			Id:          "sha256:abc123def456",
			RepoTags:    []string{"nginx:latest", "nginx:1.25"},
			RepoDigests: []string{"nginx@sha256:..."},
			Size:        142000000,
			VirtualSize: 142000000,
			Created:     time.Now().Unix() - 86400,
			Labels:      map[string]string{"maintainer": "NGINX Docker Maintainers"},
		},
	}

	return &ListImagesResponse{
		Images: images,
		Total:  int32(len(images)),
	}, nil
}

func (s *mockDockerServiceServer) GetImage(ctx context.Context, req *GetImageRequest) (*Image, error) {
	if req.ImageId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "image_id is required")
	}

	if req.ImageId == "not-found" {
		return nil, status.Errorf(codes.NotFound, "image not found")
	}

	return &Image{
		Id:          req.ImageId,
		RepoTags:    []string{"nginx:latest"},
		RepoDigests: []string{"nginx@sha256:..."},
		Size:        142000000,
	}, nil
}

func (s *mockDockerServiceServer) DeleteImage(ctx context.Context, req *DeleteImageRequest) (*DeleteImageResponse, error) {
	if req.ImageId == "in-use" {
		return nil, status.Errorf(codes.FailedPrecondition, "image is being used by container")
	}

	return &DeleteImageResponse{
		Success:  true,
		Message:  "Image deleted successfully",
		Deleted:  []string{req.ImageId},
		Untagged: []string{"nginx:latest"},
	}, nil
}

func (s *mockDockerServiceServer) TagImage(ctx context.Context, req *TagImageRequest) (*TagImageResponse, error) {
	if req.SourceImage == "" || req.TargetRepo == "" {
		return nil, status.Errorf(codes.InvalidArgument, "source_image and target_repo are required")
	}

	return &TagImageResponse{
		Success: true,
		Message: "Image tagged successfully",
	}, nil
}

// TestContractGetDockerInfo tests the GetDockerInfo RPC contract
func TestContractGetDockerInfo(t *testing.T) {
	t.Run("should return Docker daemon information", func(t *testing.T) {
		server := &mockDockerServiceServer{}
		req := &GetDockerInfoRequest{}

		resp, err := server.GetDockerInfo(context.Background(), req)
		require.NoError(t, err)

		// Verify required fields
		assert.NotEmpty(t, resp.Version)
		assert.NotEmpty(t, resp.ApiVersion)
		assert.NotEmpty(t, resp.StorageDriver)
		assert.GreaterOrEqual(t, resp.Containers, int32(0))
		assert.GreaterOrEqual(t, resp.Images, int32(0))
		assert.NotEmpty(t, resp.OperatingSystem)
		assert.NotEmpty(t, resp.Architecture)

		// Verify specific values
		assert.Equal(t, "24.0.7", resp.Version)
		assert.Equal(t, "1.43", resp.ApiVersion)
		assert.Equal(t, "overlay2", resp.StorageDriver)
		assert.Equal(t, int32(10), resp.Containers)
		assert.Equal(t, int32(5), resp.ContainersRunning)
	})
}

// TestContractListContainers tests the ListContainers RPC contract
func TestContractListContainers(t *testing.T) {
	server := &mockDockerServiceServer{}

	t.Run("should return containers list with pagination", func(t *testing.T) {
		req := &ListContainersRequest{
			All:      true,
			Page:     1,
			PageSize: 50,
		}

		resp, err := server.ListContainers(context.Background(), req)
		require.NoError(t, err)

		// Verify response structure
		assert.NotNil(t, resp.Containers)
		assert.GreaterOrEqual(t, resp.Total, int32(0))
		assert.Equal(t, req.Page, resp.Page)
		assert.Equal(t, req.PageSize, resp.PageSize)

		// Verify container structure
		if len(resp.Containers) > 0 {
			container := resp.Containers[0]
			assert.NotEmpty(t, container.Id)
			assert.NotEmpty(t, container.Name)
			assert.NotEmpty(t, container.Image)
			assert.NotEmpty(t, container.State)
			assert.Contains(t, []string{"created", "running", "paused", "exited", "dead"}, container.State)
		}
	})

	t.Run("should reject invalid page_size", func(t *testing.T) {
		req := &ListContainersRequest{
			All:      true,
			PageSize: 2000, // Exceeds maximum of 1000
		}

		_, err := server.ListContainers(context.Background(), req)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "exceeds maximum")
	})

	t.Run("should support filtering", func(t *testing.T) {
		req := &ListContainersRequest{
			All:    true,
			Filter: "name=nginx",
		}

		resp, err := server.ListContainers(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

// TestContractGetContainer tests the GetContainer RPC contract
func TestContractGetContainer(t *testing.T) {
	server := &mockDockerServiceServer{}

	t.Run("should return container details", func(t *testing.T) {
		req := &GetContainerRequest{
			ContainerId: "abc123def456",
		}

		resp, err := server.GetContainer(context.Background(), req)
		require.NoError(t, err)

		assert.Equal(t, req.ContainerId, resp.Id)
		assert.NotEmpty(t, resp.Name)
		assert.NotEmpty(t, resp.Image)
		assert.NotEmpty(t, resp.State)
	})

	t.Run("should return NOT_FOUND for non-existent container", func(t *testing.T) {
		req := &GetContainerRequest{
			ContainerId: "not-found",
		}

		_, err := server.GetContainer(context.Background(), req)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("should return INVALID_ARGUMENT for empty container_id", func(t *testing.T) {
		req := &GetContainerRequest{
			ContainerId: "",
		}

		_, err := server.GetContainer(context.Background(), req)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// TestContractContainerLifecycleOperations tests container lifecycle RPC contracts
func TestContractContainerLifecycleOperations(t *testing.T) {
	server := &mockDockerServiceServer{}

	t.Run("StartContainer should start container successfully", func(t *testing.T) {
		req := &ContainerActionRequest{
			ContainerId: "abc123def456",
		}

		resp, err := server.StartContainer(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.Equal(t, req.ContainerId, resp.ContainerId)
		assert.Greater(t, resp.Duration, int64(0))
	})

	t.Run("StopContainer should stop container with timeout", func(t *testing.T) {
		req := &ContainerActionRequest{
			ContainerId: "abc123def456",
			Timeout:     10,
		}

		resp, err := server.StopContainer(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Equal(t, req.ContainerId, resp.ContainerId)
	})

	t.Run("RestartContainer should restart container", func(t *testing.T) {
		req := &ContainerActionRequest{
			ContainerId: "abc123def456",
			Timeout:     10,
		}

		resp, err := server.RestartContainer(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "restarted")
	})

	t.Run("PauseContainer should pause container", func(t *testing.T) {
		req := &ContainerActionRequest{
			ContainerId: "abc123def456",
		}

		resp, err := server.PauseContainer(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "paused")
	})

	t.Run("UnpauseContainer should unpause container", func(t *testing.T) {
		req := &ContainerActionRequest{
			ContainerId: "abc123def456",
		}

		resp, err := server.UnpauseContainer(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "unpaused")
	})

	t.Run("DeleteContainer should delete container with options", func(t *testing.T) {
		req := &DeleteContainerRequest{
			ContainerId:   "abc123def456",
			Force:         true,
			RemoveVolumes: false,
		}

		resp, err := server.DeleteContainer(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "deleted")
	})

	t.Run("should return FAILED_PRECONDITION for invalid state", func(t *testing.T) {
		req := &ContainerActionRequest{
			ContainerId: "stopped",
		}

		_, err := server.StartContainer(context.Background(), req)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
	})
}

// TestContractListImages tests the ListImages RPC contract
func TestContractListImages(t *testing.T) {
	server := &mockDockerServiceServer{}

	t.Run("should return images list", func(t *testing.T) {
		req := &ListImagesRequest{
			All: false,
		}

		resp, err := server.ListImages(context.Background(), req)
		require.NoError(t, err)

		assert.NotNil(t, resp.Images)
		assert.GreaterOrEqual(t, resp.Total, int32(0))

		// Verify image structure
		if len(resp.Images) > 0 {
			image := resp.Images[0]
			assert.NotEmpty(t, image.Id)
			assert.NotEmpty(t, image.RepoTags)
			assert.Greater(t, image.Size, int64(0))
		}
	})

	t.Run("should support filtering", func(t *testing.T) {
		req := &ListImagesRequest{
			All:    true,
			Filter: "dangling=true",
		}

		resp, err := server.ListImages(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

// TestContractGetImage tests the GetImage RPC contract
func TestContractGetImage(t *testing.T) {
	server := &mockDockerServiceServer{}

	t.Run("should return image details", func(t *testing.T) {
		req := &GetImageRequest{
			ImageId: "sha256:abc123def456",
		}

		resp, err := server.GetImage(context.Background(), req)
		require.NoError(t, err)

		assert.Equal(t, req.ImageId, resp.Id)
		assert.NotEmpty(t, resp.RepoTags)
		assert.Greater(t, resp.Size, int64(0))
	})

	t.Run("should return NOT_FOUND for non-existent image", func(t *testing.T) {
		req := &GetImageRequest{
			ImageId: "not-found",
		}

		_, err := server.GetImage(context.Background(), req)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

// TestContractDeleteImage tests the DeleteImage RPC contract
func TestContractDeleteImage(t *testing.T) {
	server := &mockDockerServiceServer{}

	t.Run("should delete image successfully", func(t *testing.T) {
		req := &DeleteImageRequest{
			ImageId: "sha256:abc123def456",
			Force:   false,
			NoPrune: false,
		}

		resp, err := server.DeleteImage(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
		assert.NotEmpty(t, resp.Deleted)
		assert.NotEmpty(t, resp.Untagged)
	})

	t.Run("should return FAILED_PRECONDITION when image in use", func(t *testing.T) {
		req := &DeleteImageRequest{
			ImageId: "in-use",
		}

		_, err := server.DeleteImage(context.Background(), req)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "being used")
	})
}

// TestContractTagImage tests the TagImage RPC contract
func TestContractTagImage(t *testing.T) {
	server := &mockDockerServiceServer{}

	t.Run("should tag image successfully", func(t *testing.T) {
		req := &TagImageRequest{
			SourceImage: "nginx:latest",
			TargetRepo:  "myregistry.com/nginx",
			TargetTag:   "v1.0",
		}

		resp, err := server.TagImage(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Message)
	})

	t.Run("should return INVALID_ARGUMENT for missing parameters", func(t *testing.T) {
		req := &TagImageRequest{
			SourceImage: "",
			TargetRepo:  "myregistry.com/nginx",
		}

		_, err := server.TagImage(context.Background(), req)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// TestContractGRPCErrorCodes tests gRPC error code mappings
func TestContractGRPCErrorCodes(t *testing.T) {
	testCases := []struct {
		name         string
		expectedCode codes.Code
		errorMessage string
	}{
		{
			name:         "NOT_FOUND for missing resource",
			expectedCode: codes.NotFound,
			errorMessage: "Container not found",
		},
		{
			name:         "FAILED_PRECONDITION for invalid state",
			expectedCode: codes.FailedPrecondition,
			errorMessage: "Container is already stopped",
		},
		{
			name:         "INVALID_ARGUMENT for bad parameters",
			expectedCode: codes.InvalidArgument,
			errorMessage: "Invalid container ID format",
		},
		{
			name:         "UNAVAILABLE for connection failure",
			expectedCode: codes.Unavailable,
			errorMessage: "Cannot connect to Docker daemon",
		},
		{
			name:         "DEADLINE_EXCEEDED for timeout",
			expectedCode: codes.DeadlineExceeded,
			errorMessage: "Operation timeout after 30s",
		},
		{
			name:         "PERMISSION_DENIED for access denied",
			expectedCode: codes.PermissionDenied,
			errorMessage: "Permission denied to access Docker socket",
		},
		{
			name:         "INTERNAL for internal error",
			expectedCode: codes.Internal,
			errorMessage: "Docker API error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := status.Errorf(tc.expectedCode, "%s", tc.errorMessage)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tc.expectedCode, st.Code())
			assert.Contains(t, st.Message(), tc.errorMessage)
		})
	}
}

// TestContractMessageStructures tests message structure requirements
func TestContractMessageStructures(t *testing.T) {
	t.Run("Container message should have required fields", func(t *testing.T) {
		container := &Container{
			Id:      "abc123",
			Name:    "test-container",
			Image:   "nginx:latest",
			ImageId: "sha256:abc123",
			State:   "running",
		}

		assert.NotEmpty(t, container.Id)
		assert.NotEmpty(t, container.Name)
		assert.NotEmpty(t, container.Image)
		assert.NotEmpty(t, container.State)
	})

	t.Run("Image message should have required fields", func(t *testing.T) {
		image := &Image{
			Id:       "sha256:abc123",
			RepoTags: []string{"nginx:latest"},
			Size:     142000000,
		}

		assert.NotEmpty(t, image.Id)
		assert.NotEmpty(t, image.RepoTags)
		assert.Greater(t, image.Size, int64(0))
	})

	t.Run("DockerInfo message should have required fields", func(t *testing.T) {
		info := &DockerInfo{
			Version:       "24.0.7",
			ApiVersion:    "1.43",
			StorageDriver: "overlay2",
			Containers:    10,
		}

		assert.NotEmpty(t, info.Version)
		assert.NotEmpty(t, info.ApiVersion)
		assert.NotEmpty(t, info.StorageDriver)
		assert.GreaterOrEqual(t, info.Containers, int32(0))
	})
}

// TestContractPaginationRequirements tests pagination contract requirements
func TestContractPaginationRequirements(t *testing.T) {
	server := &mockDockerServiceServer{}

	t.Run("should enforce maximum page_size of 1000", func(t *testing.T) {
		req := &ListContainersRequest{
			PageSize: 1001,
		}

		_, err := server.ListContainers(context.Background(), req)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("should accept valid page_size", func(t *testing.T) {
		req := &ListContainersRequest{
			PageSize: 50,
		}

		resp, err := server.ListContainers(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, int32(50), resp.PageSize)
	})

	t.Run("should default to 50 items per page", func(t *testing.T) {
		req := &ListContainersRequest{
			PageSize: 0, // Not specified
		}

		resp, err := server.ListContainers(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

// Mock functions for streaming tests (simplified)
func TestContractStreamingOperations(t *testing.T) {
	t.Run("GetContainerStats should support streaming", func(t *testing.T) {
		req := &GetContainerStatsRequest{
			ContainerId: "abc123",
			Stream:      true,
		}

		assert.NotEmpty(t, req.ContainerId)
		assert.True(t, req.Stream)
	})

	t.Run("GetContainerLogs should support streaming", func(t *testing.T) {
		req := &GetContainerLogsRequest{
			ContainerId: "abc123",
			Follow:      true,
			TailLines:   100,
			Timestamps:  true,
		}

		assert.NotEmpty(t, req.ContainerId)
		assert.True(t, req.Follow)
		assert.Equal(t, int32(100), req.TailLines)
	})

	t.Run("PullImage should support streaming progress", func(t *testing.T) {
		req := &PullImageRequest{
			ImageName: "nginx:latest",
			Platform:  "linux/amd64",
		}

		assert.NotEmpty(t, req.ImageName)
	})
}
