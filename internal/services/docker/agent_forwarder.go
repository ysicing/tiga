package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

const (
	defaultDialTimeout    = 10 * time.Second
	defaultRequestTimeout = 30 * time.Second
	defaultAgentPort      = "50051"
)

// AgentConnection represents a gRPC connection to a Docker Agent
type agentConnection struct {
	client pb.DockerServiceClient
	conn   *grpc.ClientConn
	addr   string
}

// AgentForwarder manages gRPC connections to Docker Agents and forwards requests
type AgentForwarder struct {
	db          *gorm.DB
	connections map[uuid.UUID]*agentConnection // agentID -> connection
	mu          sync.RWMutex
}

// NewAgentForwarder creates a new AgentForwarder instance
func NewAgentForwarder(db *gorm.DB) *AgentForwarder {
	return &AgentForwarder{
		db:          db,
		connections: make(map[uuid.UUID]*agentConnection),
	}
}

// getClient gets or creates a gRPC client for the given agent ID
func (f *AgentForwarder) getClient(agentID uuid.UUID) (pb.DockerServiceClient, error) {
	// Check if connection exists
	f.mu.RLock()
	conn, exists := f.connections[agentID]
	f.mu.RUnlock()

	if exists && conn.conn.GetState().String() == "READY" {
		return conn.client, nil
	}

	// Need to create new connection
	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	conn, exists = f.connections[agentID]
	if exists && conn.conn.GetState().String() == "READY" {
		return conn.client, nil
	}

	// Get agent address from database
	addr, err := f.getAgentAddress(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent address: %w", err)
	}

	// Create new gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
	defer cancel()

	grpcConn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial agent at %s: %w", addr, err)
	}

	client := pb.NewDockerServiceClient(grpcConn)

	// Store connection
	f.connections[agentID] = &agentConnection{
		client: client,
		conn:   grpcConn,
		addr:   addr,
	}

	return client, nil
}

// getAgentAddress retrieves the gRPC address for an agent from the database
func (f *AgentForwarder) getAgentAddress(agentID uuid.UUID) (string, error) {
	// Query AgentConnection to get IP address
	// For now, we'll construct the address using IP + default port
	// TODO: Support custom agent ports
	var agentConn struct {
		IPAddress string
	}

	err := f.db.Table("agent_connections").
		Select("ip_address").
		Where("id = ? AND status = ?", agentID, "online").
		First(&agentConn).Error

	if err != nil {
		return "", fmt.Errorf("agent not found or offline: %w", err)
	}

	if agentConn.IPAddress == "" {
		return "", fmt.Errorf("agent IP address is empty")
	}

	// Construct gRPC address
	return fmt.Sprintf("%s:%s", agentConn.IPAddress, defaultAgentPort), nil
}

// Close closes all gRPC connections
func (f *AgentForwarder) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, conn := range f.connections {
		if err := conn.conn.Close(); err != nil {
			return err
		}
	}

	f.connections = make(map[uuid.UUID]*agentConnection)
	return nil
}

// CloseAgent closes the connection to a specific agent
func (f *AgentForwarder) CloseAgent(agentID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	conn, exists := f.connections[agentID]
	if !exists {
		return nil
	}

	if err := conn.conn.Close(); err != nil {
		return err
	}

	delete(f.connections, agentID)
	return nil
}

// GetDockerInfo forwards GetDockerInfo request to the agent
func (f *AgentForwarder) GetDockerInfo(agentID uuid.UUID) (*pb.GetDockerInfoResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.GetDockerInfo(ctx, &pb.GetDockerInfoRequest{})
}

// ListContainers forwards ListContainers request to the agent
func (f *AgentForwarder) ListContainers(agentID uuid.UUID, req *pb.ListContainersRequest) (*pb.ListContainersResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.ListContainers(ctx, req)
}

// GetContainer forwards GetContainer request to the agent
func (f *AgentForwarder) GetContainer(agentID uuid.UUID, req *pb.GetContainerRequest) (*pb.GetContainerResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.GetContainer(ctx, req)
}

// StartContainer forwards StartContainer request to the agent
func (f *AgentForwarder) StartContainer(agentID uuid.UUID, req *pb.StartContainerRequest) (*pb.StartContainerResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.StartContainer(ctx, req)
}

// StopContainer forwards StopContainer request to the agent
func (f *AgentForwarder) StopContainer(agentID uuid.UUID, req *pb.StopContainerRequest) (*pb.StopContainerResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.StopContainer(ctx, req)
}

// RestartContainer forwards RestartContainer request to the agent
func (f *AgentForwarder) RestartContainer(agentID uuid.UUID, req *pb.RestartContainerRequest) (*pb.RestartContainerResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.RestartContainer(ctx, req)
}

// PauseContainer forwards PauseContainer request to the agent
func (f *AgentForwarder) PauseContainer(agentID uuid.UUID, req *pb.PauseContainerRequest) (*pb.PauseContainerResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.PauseContainer(ctx, req)
}

// UnpauseContainer forwards UnpauseContainer request to the agent
func (f *AgentForwarder) UnpauseContainer(agentID uuid.UUID, req *pb.UnpauseContainerRequest) (*pb.UnpauseContainerResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.UnpauseContainer(ctx, req)
}

// DeleteContainer forwards DeleteContainer request to the agent
func (f *AgentForwarder) DeleteContainer(agentID uuid.UUID, req *pb.DeleteContainerRequest) (*pb.DeleteContainerResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.DeleteContainer(ctx, req)
}

// ListImages forwards ListImages request to the agent
func (f *AgentForwarder) ListImages(agentID uuid.UUID, req *pb.ListImagesRequest) (*pb.ListImagesResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.ListImages(ctx, req)
}

// GetImage forwards GetImage request to the agent
func (f *AgentForwarder) GetImage(agentID uuid.UUID, req *pb.GetImageRequest) (*pb.GetImageResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.GetImage(ctx, req)
}

// DeleteImage forwards DeleteImage request to the agent
func (f *AgentForwarder) DeleteImage(agentID uuid.UUID, req *pb.DeleteImageRequest) (*pb.DeleteImageResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.DeleteImage(ctx, req)
}

// TagImage forwards TagImage request to the agent
func (f *AgentForwarder) TagImage(agentID uuid.UUID, req *pb.TagImageRequest) (*pb.TagImageResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.TagImage(ctx, req)
}

// GetContainerStats forwards GetContainerStats streaming request to the agent
// Returns the stream client for the caller to read from
func (f *AgentForwarder) GetContainerStats(agentID uuid.UUID, req *pb.GetContainerStatsRequest) (pb.DockerService_GetContainerStatsClient, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // Longer timeout for streaming
	_ = cancel                                                              // Will be called by the stream client when done

	return client.GetContainerStats(ctx, req)
}

// GetContainerLogs forwards GetContainerLogs streaming request to the agent
// Returns the stream client for the caller to read from
func (f *AgentForwarder) GetContainerLogs(agentID uuid.UUID, req *pb.GetContainerLogsRequest) (pb.DockerService_GetContainerLogsClient, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // Longer timeout for streaming
	_ = cancel                                                              // Will be called by the stream client when done

	return client.GetContainerLogs(ctx, req)
}

// PullImage forwards PullImage streaming request to the agent
// Returns the stream client for the caller to read progress from
func (f *AgentForwarder) PullImage(agentID uuid.UUID, req *pb.PullImageRequest) (pb.DockerService_PullImageClient, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute) // Very long timeout for image pulls
	_ = cancel                                                               // Will be called by the stream client when done

	return client.PullImage(ctx, req)
}

// ExecContainer forwards ExecContainer bidirectional streaming request to the agent
// Returns the stream client for the caller to send/receive messages
func (f *AgentForwarder) ExecContainer(agentID uuid.UUID) (pb.DockerService_ExecContainerClient, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background() // No timeout for interactive sessions

	return client.ExecContainer(ctx)
}

// ==================== Volume Operations ====================

// ListVolumes forwards ListVolumes request to the agent
func (f *AgentForwarder) ListVolumes(agentID uuid.UUID, req *pb.ListVolumesRequest) (*pb.ListVolumesResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.ListVolumes(ctx, req)
}

// GetVolume forwards GetVolume request to the agent
func (f *AgentForwarder) GetVolume(agentID uuid.UUID, req *pb.GetVolumeRequest) (*pb.GetVolumeResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.GetVolume(ctx, req)
}

// CreateVolume forwards CreateVolume request to the agent
func (f *AgentForwarder) CreateVolume(agentID uuid.UUID, req *pb.CreateVolumeRequest) (*pb.CreateVolumeResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.CreateVolume(ctx, req)
}

// DeleteVolume forwards DeleteVolume request to the agent
func (f *AgentForwarder) DeleteVolume(agentID uuid.UUID, req *pb.DeleteVolumeRequest) (*pb.DeleteVolumeResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.DeleteVolume(ctx, req)
}

// PruneVolumes forwards PruneVolumes request to the agent
func (f *AgentForwarder) PruneVolumes(agentID uuid.UUID, req *pb.PruneVolumesRequest) (*pb.PruneVolumesResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.PruneVolumes(ctx, req)
}

// ==================== Network Operations ====================

// ListNetworks forwards ListNetworks request to the agent
func (f *AgentForwarder) ListNetworks(agentID uuid.UUID, req *pb.ListNetworksRequest) (*pb.ListNetworksResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.ListNetworks(ctx, req)
}

// GetNetwork forwards GetNetwork request to the agent
func (f *AgentForwarder) GetNetwork(agentID uuid.UUID, req *pb.GetNetworkRequest) (*pb.GetNetworkResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.GetNetwork(ctx, req)
}

// CreateNetwork forwards CreateNetwork request to the agent
func (f *AgentForwarder) CreateNetwork(agentID uuid.UUID, req *pb.CreateNetworkRequest) (*pb.CreateNetworkResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.CreateNetwork(ctx, req)
}

// DeleteNetwork forwards DeleteNetwork request to the agent
func (f *AgentForwarder) DeleteNetwork(agentID uuid.UUID, req *pb.DeleteNetworkRequest) (*pb.DeleteNetworkResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.DeleteNetwork(ctx, req)
}

// ConnectNetwork forwards ConnectNetwork request to the agent
func (f *AgentForwarder) ConnectNetwork(agentID uuid.UUID, req *pb.ConnectNetworkRequest) (*pb.ConnectNetworkResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.ConnectNetwork(ctx, req)
}

// DisconnectNetwork forwards DisconnectNetwork request to the agent
func (f *AgentForwarder) DisconnectNetwork(agentID uuid.UUID, req *pb.DisconnectNetworkRequest) (*pb.DisconnectNetworkResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.DisconnectNetwork(ctx, req)
}

// ==================== System Operations ====================

// GetSystemInfo forwards GetSystemInfo request to the agent
func (f *AgentForwarder) GetSystemInfo(agentID uuid.UUID, req *pb.GetSystemInfoRequest) (*pb.GetSystemInfoResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.GetSystemInfo(ctx, req)
}

// GetVersion forwards GetVersion request to the agent
func (f *AgentForwarder) GetVersion(agentID uuid.UUID, req *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.GetVersion(ctx, req)
}

// GetDiskUsage forwards GetDiskUsage request to the agent
func (f *AgentForwarder) GetDiskUsage(agentID uuid.UUID, req *pb.GetDiskUsageRequest) (*pb.GetDiskUsageResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.GetDiskUsage(ctx, req)
}

// Ping forwards Ping request to the agent
func (f *AgentForwarder) Ping(agentID uuid.UUID, req *pb.PingRequest) (*pb.PingResponse, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	return client.Ping(ctx, req)
}

// GetEvents forwards GetEvents streaming request to the agent
// Returns the stream client for the caller to receive messages
func (f *AgentForwarder) GetEvents(agentID uuid.UUID, req *pb.GetEventsRequest) (pb.DockerService_GetEventsClient, error) {
	client, err := f.getClient(agentID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background() // No timeout for event streams

	return client.GetEvents(ctx, req)
}
