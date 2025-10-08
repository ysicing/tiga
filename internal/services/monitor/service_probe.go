package monitor

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// ServiceProbeService handles business logic for service monitoring
type ServiceProbeService struct {
	serviceRepo repository.ServiceRepository
	scheduler   *ServiceProbeScheduler
}

// NewServiceProbeService creates a new service probe service
func NewServiceProbeService(serviceRepo repository.ServiceRepository, scheduler *ServiceProbeScheduler) *ServiceProbeService {
	return &ServiceProbeService{
		serviceRepo: serviceRepo,
		scheduler:   scheduler,
	}
}

// CreateMonitor creates a new service monitor and schedules it
func (s *ServiceProbeService) CreateMonitor(ctx context.Context, monitor *models.ServiceMonitor) error {
	if err := s.serviceRepo.Create(ctx, monitor); err != nil {
		return err
	}

	// Schedule the monitor if enabled
	if monitor.Enabled {
		return s.scheduler.ScheduleMonitor(monitor)
	}
	return nil
}

// UpdateMonitor updates a monitor and reschedules if needed
func (s *ServiceProbeService) UpdateMonitor(ctx context.Context, monitor *models.ServiceMonitor) error {
	if err := s.serviceRepo.Update(ctx, monitor); err != nil {
		return err
	}

	// Update schedule
	if monitor.Enabled {
		return s.scheduler.UpdateMonitorSchedule(monitor)
	} else {
		s.scheduler.UnscheduleMonitor(monitor.ID)
	}
	return nil
}

// DeleteMonitor deletes a monitor and unschedules it
func (s *ServiceProbeService) DeleteMonitor(ctx context.Context, id uuid.UUID) error {
	s.scheduler.UnscheduleMonitor(id)
	return s.serviceRepo.Delete(ctx, id)
}

// GetMonitor retrieves a monitor with latest status
func (s *ServiceProbeService) GetMonitor(ctx context.Context, id uuid.UUID) (*models.ServiceMonitor, error) {
	monitor, err := s.serviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Enrich with latest probe result
	if latest, err := s.serviceRepo.GetLatestProbeResult(ctx, id); err == nil {
		if latest.Success {
			monitor.Status = "online"
		} else {
			monitor.Status = "offline"
		}
		monitor.LastCheckTime = &latest.Timestamp
	}

	return monitor, nil
}

// GetAvailabilityStats calculates availability statistics for a period
func (s *ServiceProbeService) GetAvailabilityStats(ctx context.Context, monitorID uuid.UUID, period string) (*models.ServiceAvailability, error) {
	now := time.Now()
	var start time.Time

	switch period {
	case "1h":
		start = now.Add(-1 * time.Hour)
	case "24h":
		start = now.Add(-24 * time.Hour)
	case "7d":
		start = now.Add(-7 * 24 * time.Hour)
	case "30d":
		start = now.Add(-30 * 24 * time.Hour)
	default:
		start = now.Add(-24 * time.Hour)
		period = "24h"
	}

	// Calculate from probe results
	availability, err := s.serviceRepo.CalculateAvailability(ctx, monitorID, start, now)
	if err != nil {
		return nil, err
	}

	availability.Period = period
	return availability, nil
}

// TriggerManualProbe triggers a manual probe execution
func (s *ServiceProbeService) TriggerManualProbe(ctx context.Context, monitorID uuid.UUID) (*models.ServiceProbeResult, error) {
	return s.scheduler.TriggerManualProbe(ctx, monitorID)
}

// ListMonitors retrieves a list of service monitors with optional filters
func (s *ServiceProbeService) ListMonitors(ctx context.Context) ([]*models.ServiceMonitor, int64, error) {
	// TODO: Add filter support from query parameters
	filter := repository.ServiceFilter{
		Page:     1,
		PageSize: 100,
	}
	return s.serviceRepo.List(ctx, filter)
}

// ServiceHistoryInfo represents probe history info for a single service monitor
// This structure is returned to the frontend for multi-line chart rendering
type ServiceHistoryInfo struct {
	ServiceMonitorID   uuid.UUID `json:"service_monitor_id"`
	ServiceMonitorName string    `json:"service_monitor_name"` // Target name
	HostNodeID         uuid.UUID `json:"host_node_id"`
	HostNodeName       string    `json:"host_node_name"` // Executor name
	Timestamps         []int64   `json:"timestamps"`     // Unix timestamps in milliseconds
	AvgDelays          []float32 `json:"avg_delays"`     // Average delays in milliseconds
	Uptimes            []float64 `json:"uptimes"`        // Uptime percentages
}

// GetHostProbeHistory gets probe history for a specific host, grouped by service monitor
// This is used for the multi-line chart showing one executor host's probes to multiple targets
func (s *ServiceProbeService) GetHostProbeHistory(ctx context.Context, hostNodeID uuid.UUID, start, end time.Time) ([]*ServiceHistoryInfo, error) {
	// Get all service histories for this host within the time range
	histories, err := s.serviceRepo.GetServiceHistoryByHost(ctx, hostNodeID, start, end)
	if err != nil {
		return nil, err
	}

	// Group by service monitor
	historyMap := make(map[uuid.UUID][]*models.ServiceHistory)
	for _, history := range histories {
		historyMap[history.ServiceMonitorID] = append(historyMap[history.ServiceMonitorID], history)
	}

	// Build result
	var result []*ServiceHistoryInfo
	for serviceMonitorID, serviceHistories := range historyMap {
		// Get service monitor details
		monitor, err := s.serviceRepo.GetByID(ctx, serviceMonitorID)
		if err != nil {
			continue // Skip if monitor not found
		}

		info := &ServiceHistoryInfo{
			ServiceMonitorID:   serviceMonitorID,
			ServiceMonitorName: monitor.Name + " (" + monitor.Target + ")",
			HostNodeID:         hostNodeID,
			Timestamps:         make([]int64, 0, len(serviceHistories)),
			AvgDelays:          make([]float32, 0, len(serviceHistories)),
			Uptimes:            make([]float64, 0, len(serviceHistories)),
		}

		// Sort by timestamp and populate arrays
		for _, history := range serviceHistories {
			info.Timestamps = append(info.Timestamps, history.CreatedAt.UnixMilli())
			info.AvgDelays = append(info.AvgDelays, history.AvgDelay)
			info.Uptimes = append(info.Uptimes, history.UptimePercent())
		}

		result = append(result, info)
	}

	return result, nil
}

// GetOverview gets 30-day aggregated statistics for all service monitors
// This is used for the service overview page with 30-day availability heatmap
func (s *ServiceProbeService) GetOverview(ctx context.Context) (*ServiceOverviewResponse, error) {
	// Get statistics from ServiceSentinel
	stats, err := s.scheduler.GetOverviewStats(ctx)
	if err != nil {
		return nil, err
	}

	return &ServiceOverviewResponse{
		Services: stats,
	}, nil
}

// NetworkTopologyNode represents a node in the network topology
type NetworkTopologyNode struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"` // "host" or "service"
	IsOnline bool      `json:"is_online"`
}

// NetworkTopologyEdge represents connectivity between nodes
type NetworkTopologyEdge struct {
	SourceID      uuid.UUID `json:"source_id"`
	TargetID      uuid.UUID `json:"target_id"`
	AvgLatency    float32   `json:"avg_latency"`    // Average latency in ms
	MinLatency    float32   `json:"min_latency"`    // Minimum latency in ms
	MaxLatency    float32   `json:"max_latency"`    // Maximum latency in ms
	PacketLoss    float64   `json:"packet_loss"`    // Packet loss percentage
	SuccessRate   float64   `json:"success_rate"`   // Success rate percentage
	ProbeCount    int       `json:"probe_count"`    // Number of probes
	LastProbeTime time.Time `json:"last_probe_time"`
}

// NetworkTopologyResponse contains the network topology data
type NetworkTopologyResponse struct {
	Nodes []NetworkTopologyNode `json:"nodes"`
	Edges []NetworkTopologyEdge `json:"edges"`
	// Matrix provides direct access as a 2D array [source][target]
	Matrix map[string]map[string]*NetworkTopologyEdge `json:"matrix"`
}

// GetNetworkTopology gets network topology matrix showing connectivity between nodes
func (s *ServiceProbeService) GetNetworkTopology(ctx context.Context, hours int) (*NetworkTopologyResponse, error) {
	start := time.Now().Add(-time.Duration(hours) * time.Hour)
	end := time.Now()

	// Get all hosts that are participating in monitoring
	hosts, err := s.serviceRepo.GetActiveHostNodes(ctx)
	if err != nil {
		return nil, err
	}

	// Get all service monitors
	monitors, _, err := s.serviceRepo.List(ctx, repository.ServiceFilter{
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return nil, err
	}

	// Build nodes list
	nodes := make([]NetworkTopologyNode, 0)
	nodeMap := make(map[uuid.UUID]NetworkTopologyNode)

	// Add host nodes
	for _, host := range hosts {
		node := NetworkTopologyNode{
			ID:       host.ID,
			Name:     host.Name,
			Type:     "host",
			IsOnline: host.Online, // Use the Online field directly
		}
		nodes = append(nodes, node)
		nodeMap[host.ID] = node
	}

	// Add service monitor nodes (targets)
	for _, monitor := range monitors {
		node := NetworkTopologyNode{
			ID:       monitor.ID,
			Name:     monitor.Name,
			Type:     "service",
			IsOnline: monitor.Status == "online",
		}
		nodes = append(nodes, node)
		nodeMap[monitor.ID] = node
	}

	// Get probe results for the time period to build edges
	edges := make([]NetworkTopologyEdge, 0)
	matrix := make(map[string]map[string]*NetworkTopologyEdge)

	// For each host-service pair, calculate statistics
	for _, host := range hosts {
		matrix[host.ID.String()] = make(map[string]*NetworkTopologyEdge)

		for _, monitor := range monitors {
			// Get probe results for this host-service pair
			results, err := s.serviceRepo.GetProbeResultsByHostAndService(ctx, host.ID, monitor.ID, start, end)
			if err != nil || len(results) == 0 {
				continue
			}

			// Calculate statistics
			var totalLatency float32
			var minLatency float32 = float32(^uint32(0) >> 1) // Max float32
			var maxLatency float32
			successCount := 0

			for _, result := range results {
				totalLatency += float32(result.Latency)
				if float32(result.Latency) < minLatency {
					minLatency = float32(result.Latency)
				}
				if float32(result.Latency) > maxLatency {
					maxLatency = float32(result.Latency)
				}
				if result.Success {
					successCount++
				}
			}

			edge := &NetworkTopologyEdge{
				SourceID:      host.ID,
				TargetID:      monitor.ID,
				AvgLatency:    totalLatency / float32(len(results)),
				MinLatency:    minLatency,
				MaxLatency:    maxLatency,
				SuccessRate:   float64(successCount) / float64(len(results)) * 100,
				PacketLoss:    float64(len(results)-successCount) / float64(len(results)) * 100,
				ProbeCount:    len(results),
				LastProbeTime: results[len(results)-1].Timestamp,
			}

			edges = append(edges, *edge)
			matrix[host.ID.String()][monitor.ID.String()] = edge
		}
	}

	return &NetworkTopologyResponse{
		Nodes:  nodes,
		Edges:  edges,
		Matrix: matrix,
	}, nil
}
