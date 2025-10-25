package docker

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services/docker"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// NetworkHandler handles Docker network operations
type NetworkHandler struct {
	agentForwarder *docker.AgentForwarderV2
}

// NewNetworkHandler creates a new NetworkHandler instance
func NewNetworkHandler(agentForwarder *docker.AgentForwarderV2) *NetworkHandler {
	return &NetworkHandler{
		agentForwarder: agentForwarder,
	}
}

// GetNetworks lists Docker networks for a given instance
// @Summary List Docker networks
// @Description Get a list of all networks on the Docker instance
// @Tags Docker Networks
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param filters query string false "Docker filters in JSON format"
// @Success 200 {object} map[string]interface{} "Network list response"
// @Failure 400 {object} map[string]interface{} "Invalid instance ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/networks [get]
func (h *NetworkHandler) GetNetworks(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Get optional filters
	filters := c.DefaultQuery("filters", "")

	// Forward request to agent
	req := &pb.ListNetworksRequest{
		Filters: filters,
	}
	resp, err := h.agentForwarder.ListNetworks(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to list networks: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// GetNetwork gets details of a specific Docker network
// @Summary Get Docker network details
// @Description Get detailed information about a specific network
// @Tags Docker Networks
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param network_id path string true "Network ID or name"
// @Param verbose query bool false "Verbose mode"
// @Param scope query string false "Network scope"
// @Success 200 {object} map[string]interface{} "Network details"
// @Failure 400 {object} map[string]interface{} "Invalid parameters"
// @Failure 404 {object} map[string]interface{} "Network not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/networks/{network_id} [get]
func (h *NetworkHandler) GetNetwork(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Get network ID
	networkID := c.Param("network_id")
	if networkID == "" {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("network ID is required"))
		return
	}

	// Get optional parameters
	verbose := c.DefaultQuery("verbose", "false") == "true"
	scope := c.DefaultQuery("scope", "")

	// Forward request to agent
	req := &pb.GetNetworkRequest{
		NetworkId: networkID,
		Verbose:   verbose,
		Scope:     scope,
	}
	resp, err := h.agentForwarder.GetNetwork(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get network: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// CreateNetworkRequest represents the request body for network creation
type CreateNetworkRequest struct {
	Name           string            `json:"name" binding:"required"`
	CheckDuplicate bool              `json:"check_duplicate"`
	Driver         string            `json:"driver"`
	Internal       bool              `json:"internal"`
	Attachable     bool              `json:"attachable"`
	Ingress        bool              `json:"ingress"`
	EnableIPv6     bool              `json:"enable_ipv6"`
	Options        map[string]string `json:"options"`
	Labels         map[string]string `json:"labels"`
	IPAM           *IPAMConfig       `json:"ipam"`
}

// IPAMConfig represents IPAM configuration
type IPAMConfig struct {
	Driver  string            `json:"driver"`
	Config  []IPAMPool        `json:"config"`
	Options map[string]string `json:"options"`
}

// IPAMPool represents IPAM pool configuration
type IPAMPool struct {
	Subnet       string            `json:"subnet"`
	IPRange      string            `json:"ip_range"`
	Gateway      string            `json:"gateway"`
	AuxAddresses map[string]string `json:"aux_addresses"`
}

// CreateNetwork creates a new Docker network
// @Summary Create Docker network
// @Description Create a new network on the Docker instance
// @Tags Docker Networks
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body CreateNetworkRequest true "Network creation parameters"
// @Success 200 {object} map[string]interface{} "Created network details"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/networks [post]
func (h *NetworkHandler) CreateNetwork(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Parse request body
	var reqBody CreateNetworkRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Convert IPAM config
	var ipamConfig *pb.IPAMConfig
	if reqBody.IPAM != nil {
		ipamPools := make([]*pb.IPAMPool, len(reqBody.IPAM.Config))
		for i, pool := range reqBody.IPAM.Config {
			ipamPools[i] = &pb.IPAMPool{
				Subnet:       pool.Subnet,
				IpRange:      pool.IPRange,
				Gateway:      pool.Gateway,
				AuxAddresses: pool.AuxAddresses,
			}
		}
		ipamConfig = &pb.IPAMConfig{
			Driver:  reqBody.IPAM.Driver,
			Config:  ipamPools,
			Options: reqBody.IPAM.Options,
		}
	}

	// Forward request to agent
	req := &pb.CreateNetworkRequest{
		Name:           reqBody.Name,
		CheckDuplicate: reqBody.CheckDuplicate,
		Driver:         reqBody.Driver,
		Internal:       reqBody.Internal,
		Attachable:     reqBody.Attachable,
		Ingress:        reqBody.Ingress,
		Ipam:           ipamConfig,
		EnableIpv6:     reqBody.EnableIPv6,
		Options:        reqBody.Options,
		Labels:         reqBody.Labels,
	}
	resp, err := h.agentForwarder.CreateNetwork(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to create network: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// DeleteNetworkRequest represents the request body for network deletion
type DeleteNetworkRequest struct {
	NetworkID string `json:"network_id" binding:"required"`
}

// DeleteNetwork deletes a Docker network
// @Summary Delete Docker network
// @Description Delete a network from the Docker instance
// @Tags Docker Networks
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body DeleteNetworkRequest true "Network deletion parameters"
// @Success 200 {object} map[string]interface{} "Deletion result"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/networks/delete [post]
func (h *NetworkHandler) DeleteNetwork(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Parse request body
	var reqBody DeleteNetworkRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Forward request to agent
	req := &pb.DeleteNetworkRequest{
		NetworkId: reqBody.NetworkID,
	}
	resp, err := h.agentForwarder.DeleteNetwork(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to delete network: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// ConnectNetworkRequest represents the request body for connecting a container to a network
type ConnectNetworkRequest struct {
	NetworkID      string               `json:"network_id" binding:"required"`
	ContainerID    string               `json:"container_id" binding:"required"`
	EndpointConfig *EndpointConfigInput `json:"endpoint_config"`
}

// EndpointConfigInput represents endpoint configuration
type EndpointConfigInput struct {
	IPAMConfig map[string]string `json:"ipam_config"`
	Links      []string          `json:"links"`
	Aliases    []string          `json:"aliases"`
}

// ConnectNetwork connects a container to a Docker network
// @Summary Connect container to network
// @Description Connect a container to a Docker network
// @Tags Docker Networks
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body ConnectNetworkRequest true "Network connection parameters"
// @Success 200 {object} map[string]interface{} "Connection result"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/networks/connect [post]
func (h *NetworkHandler) ConnectNetwork(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Parse request body
	var reqBody ConnectNetworkRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Convert endpoint config
	var endpointConfig *pb.EndpointConfig
	if reqBody.EndpointConfig != nil {
		endpointConfig = &pb.EndpointConfig{
			IpamConfig: reqBody.EndpointConfig.IPAMConfig,
			Links:      reqBody.EndpointConfig.Links,
			Aliases:    reqBody.EndpointConfig.Aliases,
		}
	}

	// Forward request to agent
	req := &pb.ConnectNetworkRequest{
		NetworkId:      reqBody.NetworkID,
		ContainerId:    reqBody.ContainerID,
		EndpointConfig: endpointConfig,
	}
	resp, err := h.agentForwarder.ConnectNetwork(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to connect network: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// DisconnectNetworkRequest represents the request body for disconnecting a container from a network
type DisconnectNetworkRequest struct {
	NetworkID   string `json:"network_id" binding:"required"`
	ContainerID string `json:"container_id" binding:"required"`
	Force       bool   `json:"force"`
}

// DisconnectNetwork disconnects a container from a Docker network
// @Summary Disconnect container from network
// @Description Disconnect a container from a Docker network
// @Tags Docker Networks
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body DisconnectNetworkRequest true "Network disconnection parameters"
// @Success 200 {object} map[string]interface{} "Disconnection result"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/networks/disconnect [post]
func (h *NetworkHandler) DisconnectNetwork(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Parse request body
	var reqBody DisconnectNetworkRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Forward request to agent
	req := &pb.DisconnectNetworkRequest{
		NetworkId:   reqBody.NetworkID,
		ContainerId: reqBody.ContainerID,
		Force:       reqBody.Force,
	}
	resp, err := h.agentForwarder.DisconnectNetwork(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to disconnect network: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}
