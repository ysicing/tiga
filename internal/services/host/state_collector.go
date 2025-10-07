package host

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// StateSubscriber represents a client subscribed to real-time state updates
type StateSubscriber struct {
	ID       string
	HostIDs  []uuid.UUID // Empty means all hosts
	Channel  chan *models.HostState
	Created  time.Time
	LastSent time.Time
}

// StateCollector collects and distributes host monitoring states
type StateCollector struct {
	hostRepo    repository.HostRepository
	agentMgr    *AgentManager
	subscribers sync.Map // map[string]*StateSubscriber
	mu          sync.RWMutex

	// State cache for quick access
	latestStates sync.Map // map[uint]*models.HostState

	// Data retention policy
	retentionRealtime time.Duration // 24 hours
	retentionShort    time.Duration // 7 days
	retentionLong     time.Duration // 30 days
}

// NewStateCollector creates a new StateCollector
func NewStateCollector(hostRepo repository.HostRepository) *StateCollector {
	sc := &StateCollector{
		hostRepo:          hostRepo,
		agentMgr:          nil, // Will be set later via SetAgentManager
		retentionRealtime: 24 * time.Hour,
		retentionShort:    7 * 24 * time.Hour,
		retentionLong:     30 * 24 * time.Hour,
	}

	// Start background tasks
	go sc.startDataCleanup()
	go sc.monitorSubscribers()

	return sc
}

// SetAgentManager sets the agent manager reference (for breaking circular dependency)
func (sc *StateCollector) SetAgentManager(agentMgr *AgentManager) {
	sc.agentMgr = agentMgr
}

// CollectState processes a new state report from an agent
func (sc *StateCollector) CollectState(ctx context.Context, hostID uuid.UUID, state *models.HostState) error {
	// Save to database
	if err := sc.hostRepo.SaveState(ctx, state); err != nil {
		return err
	}

	// Update cache
	sc.latestStates.Store(hostID, state)

	// Get subscriber count
	subscriberCount := 0
	sc.subscribers.Range(func(key, value interface{}) bool {
		subscriberCount++
		return true
	})

	logrus.Debugf("[StateCollector] State collected for host %s, broadcasting to %d subscribers",
		hostID.String(), subscriberCount)

	// Broadcast to subscribers
	sc.broadcastState(state)

	return nil
}

// GetLatestState retrieves the latest cached state for a host
func (sc *StateCollector) GetLatestState(hostID uuid.UUID) (*models.HostState, bool) {
	if state, ok := sc.latestStates.Load(hostID); ok {
		return state.(*models.HostState), true
	}
	return nil, false
}

// GetLatestStates retrieves latest states for multiple hosts
func (sc *StateCollector) GetLatestStates(hostIDs []uuid.UUID) map[uuid.UUID]*models.HostState {
	result := make(map[uuid.UUID]*models.HostState)
	for _, id := range hostIDs {
		if state, ok := sc.GetLatestState(id); ok {
			result[id] = state
		}
	}
	return result
}

// Subscribe creates a new subscription for real-time state updates
func (sc *StateCollector) Subscribe(subscriberID string, hostIDs []uuid.UUID) *StateSubscriber {
	sub := &StateSubscriber{
		ID:       subscriberID,
		HostIDs:  hostIDs,
		Channel:  make(chan *models.HostState, 100), // Buffered channel
		Created:  time.Now(),
		LastSent: time.Now(),
	}

	sc.subscribers.Store(subscriberID, sub)
	return sub
}

// Unsubscribe removes a subscriber
func (sc *StateCollector) Unsubscribe(subscriberID string) {
	if sub, ok := sc.subscribers.LoadAndDelete(subscriberID); ok {
		subscriber := sub.(*StateSubscriber)
		close(subscriber.Channel)
	}
}

// broadcastState sends state update to all interested subscribers
func (sc *StateCollector) broadcastState(state *models.HostState) {
	sc.subscribers.Range(func(key, value interface{}) bool {
		sub := value.(*StateSubscriber)

		// Check if subscriber is interested in this host
		if len(sub.HostIDs) == 0 || sc.containsHostID(sub.HostIDs, state.HostNodeID) {
			select {
			case sub.Channel <- state:
				sub.LastSent = time.Now()
			default:
				// Channel full, skip this update
			}
		}
		return true
	})
}

// containsHostID checks if a host ID is in the list
func (sc *StateCollector) containsHostID(hostIDs []uuid.UUID, targetID uuid.UUID) bool {
	for _, id := range hostIDs {
		if id == targetID {
			return true
		}
	}
	return false
}

// GetHistoricalStates retrieves historical states for a time range
func (sc *StateCollector) GetHistoricalStates(ctx context.Context, hostID uuid.UUID, start, end time.Time, interval string) ([]*models.HostState, error) {
	states, err := sc.hostRepo.GetStatesByTimeRange(ctx, hostID, start, end)
	if err != nil {
		return nil, err
	}

	// Apply interval aggregation if needed
	if interval != "" && interval != "auto" {
		states = sc.aggregateStates(states, interval)
	}

	return states, nil
}

// aggregateStates aggregates states based on interval (simplified version)
func (sc *StateCollector) aggregateStates(states []*models.HostState, interval string) []*models.HostState {
	// TODO: Implement proper aggregation based on interval (1m, 5m, 1h, 1d)
	// For now, return as-is or apply simple sampling
	return states
}

// GetStateStatistics calculates statistics for a time period
func (sc *StateCollector) GetStateStatistics(ctx context.Context, hostID uuid.UUID, start, end time.Time) (map[string]interface{}, error) {
	states, err := sc.hostRepo.GetStatesByTimeRange(ctx, hostID, start, end)
	if err != nil {
		return nil, err
	}

	if len(states) == 0 {
		return map[string]interface{}{
			"count": 0,
		}, nil
	}

	// Calculate statistics
	stats := map[string]interface{}{
		"count": len(states),
	}

	var totalCPU, totalMem, totalDisk float64
	var maxCPU, maxMem, maxDisk float64
	var minCPU, minMem, minDisk float64 = 100, 100, 100

	for _, state := range states {
		totalCPU += state.CPUUsage
		totalMem += state.MemUsage
		totalDisk += state.DiskUsage

		if state.CPUUsage > maxCPU {
			maxCPU = state.CPUUsage
		}
		if state.CPUUsage < minCPU {
			minCPU = state.CPUUsage
		}
		if state.MemUsage > maxMem {
			maxMem = state.MemUsage
		}
		if state.MemUsage < minMem {
			minMem = state.MemUsage
		}
		if state.DiskUsage > maxDisk {
			maxDisk = state.DiskUsage
		}
		if state.DiskUsage < minDisk {
			minDisk = state.DiskUsage
		}
	}

	count := float64(len(states))
	stats["cpu"] = map[string]float64{
		"avg": totalCPU / count,
		"max": maxCPU,
		"min": minCPU,
	}
	stats["memory"] = map[string]float64{
		"avg": totalMem / count,
		"max": maxMem,
		"min": minMem,
	}
	stats["disk"] = map[string]float64{
		"avg": totalDisk / count,
		"max": maxDisk,
		"min": minDisk,
	}

	return stats, nil
}

// startDataCleanup runs periodic cleanup of old monitoring data
func (sc *StateCollector) startDataCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sc.cleanupOldData()
	}
}

// cleanupOldData removes old states according to retention policy
func (sc *StateCollector) cleanupOldData() {
	// TODO: Implement data cleanup based on retention policy
	// - Delete raw data older than 24 hours
	// - Aggregate and keep 1-minute averages for 7 days
	// - Aggregate and keep hourly averages for 30 days
}

// monitorSubscribers monitors subscriber health and removes stale ones
func (sc *StateCollector) monitorSubscribers() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sc.subscribers.Range(func(key, value interface{}) bool {
			sub := value.(*StateSubscriber)
			// Remove subscribers that haven't received data in 30 minutes
			if time.Since(sub.LastSent) > 30*time.Minute {
				sc.Unsubscribe(sub.ID)
			}
			return true
		})
	}
}

// ExportStates exports states to JSON format
func (sc *StateCollector) ExportStates(ctx context.Context, hostID uuid.UUID, start, end time.Time) ([]byte, error) {
	states, err := sc.hostRepo.GetStatesByTimeRange(ctx, hostID, start, end)
	if err != nil {
		return nil, err
	}

	return json.Marshal(states)
}

// GetRealtimeMetrics returns current metrics for dashboard
func (sc *StateCollector) GetRealtimeMetrics() map[string]interface{} {
	activeAgents := 0
	totalHosts := 0

	// Count active agents (if agentMgr is set)
	if sc.agentMgr != nil {
		connections := sc.agentMgr.GetActiveConnections()
		activeAgents = len(connections)
	}

	// Count total states in cache
	sc.latestStates.Range(func(key, value interface{}) bool {
		totalHosts++
		return true
	})

	// Count subscribers
	subscriberCount := 0
	sc.subscribers.Range(func(key, value interface{}) bool {
		subscriberCount++
		return true
	})

	return map[string]interface{}{
		"active_agents":  activeAgents,
		"total_hosts":    totalHosts,
		"subscribers":    subscriberCount,
		"cache_hit_rate": 0.95, // TODO: Calculate actual cache hit rate
	}
}
