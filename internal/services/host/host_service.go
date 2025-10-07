package host

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// HostService handles business logic for host management
type HostService struct {
	hostRepo     repository.HostRepository
	agentMgr     *AgentManager
	stateCollector *StateCollector
	serverURL    string // Server URL for agent installation
}

// NewHostService creates a new HostService
func NewHostService(hostRepo repository.HostRepository, agentMgr *AgentManager, stateCollector *StateCollector, serverURL string) *HostService {
	return &HostService{
		hostRepo:       hostRepo,
		agentMgr:       agentMgr,
		stateCollector: stateCollector,
		serverURL:      serverURL,
	}
}

// CreateHost creates a new host node with generated UUID and secret key
func (s *HostService) CreateHost(ctx context.Context, host *models.HostNode) error {
	// Generate UUID
	host.UUID = uuid.New().String()

	// Generate secret key
	secretKey, err := s.generateSecretKey()
	if err != nil {
		return fmt.Errorf("failed to generate secret key: %w", err)
	}
	host.SecretKey = secretKey // TODO: Encrypt with AES-256

	// Create the host
	if err := s.hostRepo.Create(ctx, host); err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}

	return nil
}

// GetHost retrieves a host by ID with online status
func (s *HostService) GetHost(ctx context.Context, id uint) (*models.HostNode, error) {
	host, err := s.hostRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Enrich with runtime data
	s.enrichHostWithRuntimeData(host)

	return host, nil
}

// ListHosts retrieves a list of hosts with filters
func (s *HostService) ListHosts(ctx context.Context, filter repository.HostFilter) ([]*models.HostNode, int64, error) {
	hosts, total, err := s.hostRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Enrich each host with runtime data
	for _, host := range hosts {
		s.enrichHostWithRuntimeData(host)
	}

	return hosts, total, nil
}

// UpdateHost updates a host node
func (s *HostService) UpdateHost(ctx context.Context, host *models.HostNode) error {
	// Don't allow updating UUID or secret key through this method
	existingHost, err := s.hostRepo.GetByID(ctx, host.ID)
	if err != nil {
		return err
	}

	// Preserve UUID and secret key
	host.UUID = existingHost.UUID
	host.SecretKey = existingHost.SecretKey

	return s.hostRepo.Update(ctx, host)
}

// DeleteHost deletes a host node
func (s *HostService) DeleteHost(ctx context.Context, id uint) error {
	// Get the host to find its UUID
	host, err := s.hostRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Disconnect the agent if online
	if s.agentMgr.IsAgentOnline(host.UUID) {
		s.agentMgr.DisconnectAgent(host.UUID)
	}

	// Delete the host
	return s.hostRepo.Delete(ctx, id)
}

// GetAgentInstallCommand generates the agent installation command
func (s *HostService) GetAgentInstallCommand(ctx context.Context, id uint) (string, error) {
	host, err := s.hostRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// Generate installation command
	cmd := fmt.Sprintf(
		"curl -fsSL %s/agent/install.sh | bash -s -- --server %s --uuid %s --key %s",
		s.serverURL,
		s.serverURL,
		host.UUID,
		host.SecretKey,
	)

	return cmd, nil
}

// RegenerateSecretKey regenerates the secret key for a host
func (s *HostService) RegenerateSecretKey(ctx context.Context, id uint) (string, error) {
	host, err := s.hostRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// Disconnect current agent if online
	if s.agentMgr.IsAgentOnline(host.UUID) {
		s.agentMgr.DisconnectAgent(host.UUID)
	}

	// Generate new secret key
	newKey, err := s.generateSecretKey()
	if err != nil {
		return "", err
	}

	host.SecretKey = newKey // TODO: Encrypt
	if err := s.hostRepo.Update(ctx, host); err != nil {
		return "", err
	}

	return newKey, nil
}

// GetHostState retrieves the current state of a host
func (s *HostService) GetHostState(ctx context.Context, id uint) (*models.HostState, error) {
	// Try cache first
	if state, ok := s.stateCollector.GetLatestState(id); ok {
		return state, nil
	}

	// Fallback to database
	return s.hostRepo.GetLatestState(ctx, id)
}

// GetHostStateHistory retrieves historical states for a host
func (s *HostService) GetHostStateHistory(ctx context.Context, id uint, start, end string, interval string) ([]*models.HostState, error) {
	// Parse time strings
	// TODO: Implement time parsing
	// For now, simplified version
	return s.hostRepo.GetLatestStates(ctx, id, 100)
}

// enrichHostWithRuntimeData adds online status and latest state to host
func (s *HostService) enrichHostWithRuntimeData(host *models.HostNode) {
	// Check online status
	host.Online = s.agentMgr.IsAgentOnline(host.UUID)

	// Get connection info if online
	if host.Online {
		if conn := s.agentMgr.GetConnectionByHostID(host.ID); conn != nil {
			host.LastActive = &conn.LastSeen
		}
	}
}

// generateSecretKey generates a random secret key
func (s *HostService) generateSecretKey() (string, error) {
	// Generate 32 bytes (256 bits) of random data
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	// Encode to base64 for easy transmission
	return base64.URLEncoding.EncodeToString(key), nil
}

// BatchUpdateDisplayIndex updates display indexes for multiple hosts
func (s *HostService) BatchUpdateDisplayIndex(ctx context.Context, updates map[uint]int) error {
	for id, index := range updates {
		host, err := s.hostRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		host.DisplayIndex = index
		if err := s.hostRepo.Update(ctx, host); err != nil {
			return err
		}
	}
	return nil
}

// GetOnlineHosts returns all currently online hosts
func (s *HostService) GetOnlineHosts(ctx context.Context) ([]*models.HostNode, error) {
	connections := s.agentMgr.GetActiveConnections()
	var hosts []*models.HostNode

	for _, conn := range connections {
		host, err := s.hostRepo.GetByID(ctx, conn.HostNodeID)
		if err != nil {
			continue
		}
		s.enrichHostWithRuntimeData(host)
		hosts = append(hosts, host)
	}

	return hosts, nil
}

// GetHostMetricsSummary returns a summary of key metrics for a host
func (s *HostService) GetHostMetricsSummary(ctx context.Context, id uint) (map[string]interface{}, error) {
	state, err := s.GetHostState(ctx, id)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"cpu_usage":    state.CPUUsage,
		"mem_usage":    state.MemUsage,
		"disk_usage":   state.DiskUsage,
		"uptime":       state.Uptime,
		"last_updated": state.Timestamp,
	}

	return summary, nil
}
