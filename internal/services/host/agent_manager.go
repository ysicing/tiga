package host

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// AgentConnection represents an active agent connection
type AgentConnection struct {
	UUID       string
	HostNodeID uuid.UUID
	Stream     proto.HostMonitor_ReportStateServer
	Connected  time.Time
	LastSeen   time.Time
	cancel     context.CancelFunc

	// Task queue for pending tasks
	taskQueue chan *proto.AgentTask
	taskMu    sync.Mutex
}

// AgentManager manages agent connections and gRPC streams
type AgentManager struct {
	hostRepo       repository.HostRepository
	stateCollector *StateCollector
	db             *gorm.DB

	// Active connections map: UUID -> Connection
	connections sync.Map
	mu          sync.RWMutex

	// Heartbeat monitoring
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
}

// NewAgentManager creates a new AgentManager
func NewAgentManager(hostRepo repository.HostRepository, stateCollector *StateCollector, db *gorm.DB) *AgentManager {
	return &AgentManager{
		hostRepo:          hostRepo,
		stateCollector:    stateCollector,
		db:                db,
		heartbeatInterval: 30 * time.Second,
		heartbeatTimeout:  90 * time.Second,
	}
}

// RegisterAgent handles agent registration
func (m *AgentManager) RegisterAgent(ctx context.Context, req *proto.RegisterAgentRequest) (*proto.RegisterAgentResponse, error) {
	// Validate UUID and secret key
	host, err := m.hostRepo.GetByUUID(ctx, req.Uuid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &proto.RegisterAgentResponse{
				Success: false,
				Message: "Host not found or invalid credentials",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to get host: %v", err)
	}

	// TODO: Verify secret key (should be encrypted comparison)
	// For now, simplified verification
	if host.SecretKey == "" {
		return &proto.RegisterAgentResponse{
			Success: false,
			Message: "Invalid credentials",
		}, nil
	}

	// Update or create HostInfo
	hostInfo := &models.HostInfo{
		HostNodeID:      host.ID,
		Platform:        req.HostInfo.Platform,
		PlatformVersion: req.HostInfo.PlatformVersion,
		Arch:            req.HostInfo.Arch,
		Virtualization:  req.HostInfo.Virtualization,
		CPUModel:        req.HostInfo.CpuModel,
		CPUCores:        int(req.HostInfo.CpuCores),
		MemTotal:        req.HostInfo.MemTotal,
		DiskTotal:       req.HostInfo.DiskTotal,
		SwapTotal:       req.HostInfo.SwapTotal,
		AgentVersion:    req.HostInfo.AgentVersion,
		BootTime:        uint64(req.HostInfo.BootTime),
		SSHEnabled:      req.HostInfo.SshEnabled,
		SSHPort:         int(req.HostInfo.SshPort),
		SSHUser:         req.HostInfo.SshUser,
	}

	// Upsert HostInfo
	if err := m.db.WithContext(ctx).
		Where("host_node_id = ?", host.ID).
		Assign(hostInfo).
		FirstOrCreate(&hostInfo).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save host info: %v", err)
	}

	// Update or create AgentConnection record
	agentConn := &models.AgentConnection{
		HostNodeID:   host.ID,
		Status:       models.AgentStatusOnline,
		AgentVersion: req.HostInfo.AgentVersion,
	}
	now := time.Now()
	agentConn.ConnectedAt = &now
	agentConn.LastHeartbeat = now

	if err := m.db.WithContext(ctx).
		Where("host_node_id = ?", host.ID).
		Assign(agentConn).
		FirstOrCreate(&agentConn).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save agent connection: %v", err)
	}

	return &proto.RegisterAgentResponse{
		Success:    true,
		Message:    "Registration successful",
		ServerTime: time.Now().Unix(),
	}, nil
}

// HandleReportState handles the bidirectional streaming of host states
func (m *AgentManager) HandleReportState(stream proto.HostMonitor_ReportStateServer) error {
	ctx := stream.Context()
	var currentUUID string
	var hostNodeID uuid.UUID

	// Cleanup on disconnect
	defer func() {
		if currentUUID != "" {
			m.DisconnectAgent(currentUUID)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			req, err := stream.Recv()
			if err != nil {
				return err
			}

			// First message should establish the connection
			if currentUUID == "" {
				currentUUID = req.Uuid

				// Verify the host exists
				host, err := m.hostRepo.GetByUUID(context.Background(), currentUUID)
				if err != nil {
					return status.Errorf(codes.Unauthenticated, "invalid UUID")
				}
				hostNodeID = host.ID

				// Register the active connection
				m.RegisterConnection(currentUUID, hostNodeID, stream)
			}

			// Process the state report
			if req.State != nil {
				if err := m.processStateReport(hostNodeID, req.State); err != nil {
					// Log error but don't disconnect
					fmt.Printf("Error processing state: %v\n", err)
				}

				// Update LastSeen to treat state report as heartbeat
				if conn, ok := m.connections.Load(currentUUID); ok {
					agentConn := conn.(*AgentConnection)
					agentConn.LastSeen = time.Now()

					// Also update database heartbeat record
					var dbConn models.AgentConnection
					if err := m.db.Where("host_node_id = ?", hostNodeID).First(&dbConn).Error; err == nil {
						dbConn.UpdateHeartbeat()
						m.db.Save(&dbConn)
					}

					logrus.Debugf("[AgentManager] Updated LastSeen and DB heartbeat for agent %s", currentUUID)
				}
			}

			// Send acknowledgment with any pending tasks
			resp := &proto.ReportStateResponse{
				Success: true,
				Message: "State received",
				Tasks:   m.getPendingTasks(currentUUID),
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

// processStateReport saves the host state to database
func (m *AgentManager) processStateReport(hostNodeID uuid.UUID, state *proto.HostState) error {
	hostState := &models.HostState{
		HostNodeID:       hostNodeID,
		Timestamp:        time.UnixMilli(state.Timestamp),
		CPUUsage:         state.CpuUsage,
		Load1:            state.Load_1,
		Load5:            state.Load_5,
		Load15:           state.Load_15,
		MemUsed:          state.MemUsed,
		MemUsage:         state.MemUsage,
		SwapUsed:         state.SwapUsed,
		DiskUsed:         state.DiskUsed,
		DiskUsage:        state.DiskUsage,
		NetInTransfer:    state.NetInTransfer,
		NetOutTransfer:   state.NetOutTransfer,
		NetInSpeed:       state.NetInSpeed,
		NetOutSpeed:      state.NetOutSpeed,
		TCPConnCount:     uint64(state.TcpConnCount),
		UDPConnCount:     uint64(state.UdpConnCount),
		ProcessCount:     uint64(state.ProcessCount),
		Uptime:           uint64(state.Uptime),
		GPUUsage:         state.GpuUsage,
		TrafficSent:      state.TrafficSent,
		TrafficRecv:      state.TrafficRecv,
		TrafficDeltaSent: state.TrafficDeltaSent,
		TrafficDeltaRecv: state.TrafficDeltaRecv,
	}

	// Convert temperatures to JSON if present
	if len(state.Temperatures) > 0 {
		// TODO: Marshal temperatures to JSON
	}

	// Save the state to database and broadcast to subscribers
	if err := m.stateCollector.CollectState(context.Background(), hostNodeID, hostState); err != nil {
		return err
	}

	// Accumulate traffic delta into host_node.traffic_used
	if state.TrafficDeltaSent > 0 || state.TrafficDeltaRecv > 0 {
		totalDelta := state.TrafficDeltaSent + state.TrafficDeltaRecv
		if err := m.accumulateTraffic(hostNodeID, int64(totalDelta)); err != nil {
			// Log error but don't fail the entire operation
			fmt.Printf("Failed to accumulate traffic for host %d: %v\n", hostNodeID, err)
		}
	}

	return nil
}

// accumulateTraffic adds delta traffic to the host's total traffic_used
func (m *AgentManager) accumulateTraffic(hostNodeID uuid.UUID, delta int64) error {
	return m.db.Model(&models.HostNode{}).
		Where("id = ?", hostNodeID).
		UpdateColumn("traffic_used", gorm.Expr("traffic_used + ?", delta)).
		Error
}

// RegisterConnection registers an active agent connection
func (m *AgentManager) RegisterConnection(uuid string, hostNodeID uuid.UUID, stream proto.HostMonitor_ReportStateServer) {
	ctx, cancel := context.WithCancel(context.Background())

	conn := &AgentConnection{
		UUID:       uuid,
		HostNodeID: hostNodeID,
		Stream:     stream,
		Connected:  time.Now(),
		LastSeen:   time.Now(),
		cancel:     cancel,
		taskQueue:  make(chan *proto.AgentTask, 100), // Buffered channel
	}

	m.connections.Store(uuid, conn)

	// Update agent connection status in database
	m.updateConnectionStatus(hostNodeID, models.AgentStatusOnline)

	// Update last active time (online status is runtime only)
	now := time.Now()
	m.db.Model(&models.HostNode{}).Where("id = ?", hostNodeID).Update("last_active", now)

	logrus.Infof("[AgentManager] Registered connection: uuid=%s, hostID=%s, connections=%d",
		uuid, hostNodeID.String(), m.GetActiveConnectionCount())

	// Start heartbeat monitoring for this connection
	go m.monitorConnection(ctx, uuid)
}

// DisconnectAgent removes an agent connection
func (m *AgentManager) DisconnectAgent(uuid string) {
	if conn, ok := m.connections.LoadAndDelete(uuid); ok {
		agentConn := conn.(*AgentConnection)
		if agentConn.cancel != nil {
			agentConn.cancel()
		}

		// Update database status
		m.updateConnectionStatus(agentConn.HostNodeID, models.AgentStatusOffline)

		// Update last active time (online status is runtime only)
		now := time.Now()
		m.db.Model(&models.HostNode{}).Where("id = ?", agentConn.HostNodeID).Update("last_active", now)
	}
}

// updateConnectionStatus updates the agent connection status in database
func (m *AgentManager) updateConnectionStatus(hostNodeID uuid.UUID, status models.AgentConnectionStatus) {
	var agentConn models.AgentConnection
	if err := m.db.Where("host_node_id = ?", hostNodeID).First(&agentConn).Error; err != nil {
		return
	}

	agentConn.Status = status
	if status == models.AgentStatusOffline {
		now := time.Now()
		agentConn.LastDisconnectAt = &now
		agentConn.DisconnectReason = "Connection lost"
	}

	m.db.Save(&agentConn)
}

// monitorConnection monitors a connection for heartbeat timeout
func (m *AgentManager) monitorConnection(ctx context.Context, uuid string) {
	ticker := time.NewTicker(m.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conn, ok := m.connections.Load(uuid)
			if !ok {
				return
			}

			agentConn := conn.(*AgentConnection)
			if time.Since(agentConn.LastSeen) > m.heartbeatTimeout {
				// Timeout - disconnect the agent
				m.DisconnectAgent(uuid)
				return
			}
		}
	}
}

// Heartbeat handles heartbeat requests from agents
func (m *AgentManager) Heartbeat(ctx context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	// Update last seen time
	if conn, ok := m.connections.Load(req.Uuid); ok {
		agentConn := conn.(*AgentConnection)
		agentConn.LastSeen = time.Now()

		// Update heartbeat in database
		var dbConn models.AgentConnection
		if err := m.db.WithContext(ctx).Where("host_node_id = ?", agentConn.HostNodeID).First(&dbConn).Error; err == nil {
			dbConn.UpdateHeartbeat()
			m.db.Save(&dbConn)
		}
	}

	return &proto.HeartbeatResponse{
		Success:    true,
		ServerTime: time.Now().Unix(),
	}, nil
}

// GetActiveConnections returns all active agent connections
func (m *AgentManager) GetActiveConnections() map[string]*AgentConnection {
	result := make(map[string]*AgentConnection)
	m.connections.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*AgentConnection)
		return true
	})
	return result
}

// IsAgentOnline checks if an agent is currently connected
func (m *AgentManager) IsAgentOnline(uuid string) bool {
	_, ok := m.connections.Load(uuid)
	return ok
}

// GetConnectionByHostID returns the connection for a given host ID
func (m *AgentManager) GetConnectionByHostID(hostID uuid.UUID) *AgentConnection {
	var result *AgentConnection
	m.connections.Range(func(key, value interface{}) bool {
		conn := value.(*AgentConnection)
		if conn.HostNodeID == hostID {
			result = conn
			return false
		}
		return true
	})
	return result
}

// GetActiveConnectionCount returns the number of active connections
func (m *AgentManager) GetActiveConnectionCount() int {
	count := 0
	m.connections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// getPendingTasks retrieves and clears pending tasks for an agent
func (m *AgentManager) getPendingTasks(uuid string) []*proto.AgentTask {
	conn, ok := m.connections.Load(uuid)
	if !ok {
		return nil
	}

	agentConn := conn.(*AgentConnection)
	var tasks []*proto.AgentTask

	// Drain all tasks from the queue without blocking
	for {
		select {
		case task := <-agentConn.taskQueue:
			tasks = append(tasks, task)
		default:
			return tasks
		}
	}
}

// QueueTask adds a task to the agent's queue
func (m *AgentManager) QueueTask(uuid string, task *proto.AgentTask) error {
	conn, ok := m.connections.Load(uuid)
	if !ok {
		return fmt.Errorf("agent not connected: %s", uuid)
	}

	agentConn := conn.(*AgentConnection)

	// Non-blocking send to avoid deadlock
	select {
	case agentConn.taskQueue <- task:
		return nil
	default:
		return fmt.Errorf("task queue full for agent: %s", uuid)
	}
}
