package host

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/docker"
	"github.com/ysicing/tiga/proto"
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

	// Task results waiting channels
	pendingResults map[string]chan *proto.TaskResult
	resultsMu      sync.RWMutex
}

// AgentManager manages agent connections and gRPC streams
type AgentManager struct {
	hostRepo              repository.HostRepository
	stateCollector        *StateCollector
	db                    *gorm.DB
	auditLogger           *AuditLogger                  // T038: 统一审计
	dockerInstanceService *docker.DockerInstanceService // T032: Docker实例集成

	// Active connections map: UUID -> Connection
	connections sync.Map
	mu          sync.RWMutex

	// Heartbeat monitoring
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
}

// NewAgentManager creates a new AgentManager
// T032: 添加Docker服务集成
func NewAgentManager(
	hostRepo repository.HostRepository,
	stateCollector *StateCollector,
	db *gorm.DB,
	auditLogger *AuditLogger,
	dockerInstanceService *docker.DockerInstanceService,
) *AgentManager {
	return &AgentManager{
		hostRepo:              hostRepo,
		stateCollector:        stateCollector,
		db:                    db,
		auditLogger:           auditLogger,
		dockerInstanceService: dockerInstanceService,
		heartbeatInterval:     30 * time.Second,
		heartbeatTimeout:      90 * time.Second,
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
	}
	spew.Dump("RegisterAgent HostInfo:", req.HostInfo)

	// Upsert HostInfo - query existing record to preserve ID and timestamps
	var existingInfo models.HostInfo
	err = m.db.WithContext(ctx).Where("host_node_id = ?", host.ID).First(&existingInfo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, status.Errorf(codes.Internal, "failed to query host info: %v", err)
	}

	if err == gorm.ErrRecordNotFound {
		// Create new record
		if err = m.db.WithContext(ctx).Create(hostInfo).Error; err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create host info: %v", err)
		}
	} else {
		// Update existing record - preserve ID and CreatedAt
		hostInfo.ID = existingInfo.ID
		hostInfo.CreatedAt = existingInfo.CreatedAt
		if err = m.db.WithContext(ctx).Save(hostInfo).Error; err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update host info: %v", err)
		}
	}

	// Get client IP address from gRPC context
	var clientIP string
	if p, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			clientIP = tcpAddr.IP.String()
		}
	}

	// Update or create AgentConnection record
	agentConn := &models.AgentConnection{
		HostNodeID:   host.ID,
		Status:       models.AgentStatusOnline,
		AgentVersion: req.HostInfo.AgentVersion,
		IPAddress:    clientIP, // Save client IP for AgentForwarder
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

	// T032: Docker实例自动发现
	// 从Agent上报的HostInfo中检测Docker信息
	if req.HostInfo.DockerInfo != nil && req.HostInfo.DockerInfo.Installed {
		go m.discoverDockerInstanceFromProto(agentConn.ID, host.ID, req.HostInfo.DockerInfo)
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

			// Process task results
			if len(req.TaskResults) > 0 {
				if err := m.processTaskResults(currentUUID, req.TaskResults); err != nil {
					logrus.WithError(err).Warn("[AgentManager] Failed to process task results")
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
	// Log agent version info if available (backward compatible)
	if state.VersionInfo != nil {
		logrus.WithFields(logrus.Fields{
			"agent_version": state.VersionInfo.Version,
			"build_time":    state.VersionInfo.BuildTime,
			"commit_id":     state.VersionInfo.CommitId,
			"host_id":       hostNodeID.String(),
		}).Debug("Agent version info received")
	}

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

	// Check if this is a reconnection
	var agentConn models.AgentConnection
	isReconnection := false
	if err := m.db.Where("host_node_id = ?", hostNodeID).First(&agentConn).Error; err == nil {
		if agentConn.LastDisconnectAt != nil {
			isReconnection = true
		}
	}

	conn := &AgentConnection{
		UUID:           uuid,
		HostNodeID:     hostNodeID,
		Stream:         stream,
		Connected:      time.Now(),
		LastSeen:       time.Now(),
		cancel:         cancel,
		taskQueue:      make(chan *proto.AgentTask, 100), // Buffered channel
		pendingResults: make(map[string]chan *proto.TaskResult),
	}

	m.connections.Store(uuid, conn)

	// Update agent connection status in database
	m.updateConnectionStatus(hostNodeID, models.AgentStatusOnline)

	// Update last active time (online status is runtime only)
	now := time.Now()
	m.db.Model(&models.HostNode{}).Where("id = ?", hostNodeID).Update("last_active", now)

	logrus.Infof("[AgentManager] Registered connection: uuid=%s, hostID=%s, connections=%d",
		uuid, hostNodeID.String(), m.GetActiveConnectionCount())

	// Record activity log
	action := models.ActivityAgentConnected
	description := "Agent connected"
	metadata := fmt.Sprintf(`{"uuid":"%s","connection_count":%d}`, uuid, m.GetActiveConnectionCount())

	if isReconnection && agentConn.LastDisconnectAt != nil {
		action = models.ActivityAgentReconnected
		offlineDuration := time.Since(*agentConn.LastDisconnectAt)

		// Format offline duration in human-readable format
		var durationStr string
		if offlineDuration < time.Minute {
			durationStr = fmt.Sprintf("%.0f秒", offlineDuration.Seconds())
		} else if offlineDuration < time.Hour {
			durationStr = fmt.Sprintf("%.0f分钟", offlineDuration.Minutes())
		} else if offlineDuration < 24*time.Hour {
			durationStr = fmt.Sprintf("%.1f小时", offlineDuration.Hours())
		} else {
			durationStr = fmt.Sprintf("%.1f天", offlineDuration.Hours()/24)
		}

		description = fmt.Sprintf("Agent reconnected (离线 %s)", durationStr)
		metadata = fmt.Sprintf(`{"uuid":"%s","connection_count":%d,"offline_duration":"%s","offline_seconds":%.0f}`,
			uuid, m.GetActiveConnectionCount(), durationStr, offlineDuration.Seconds())
	}

	// Record agent connection activity (T038: 使用统一审计)
	if err := m.auditLogger.LogActivity(context.Background(), AuditEntry{
		HostNodeID:  hostNodeID,
		UserID:      nil, // System action
		Username:    "",
		Action:      action,
		ActionType:  models.ActivityTypeAgent,
		Description: description,
		Metadata:    metadata,
		ClientIP:    "",
		UserAgent:   "",
	}); err != nil {
		logrus.Warnf("Failed to record agent connection activity: %v", err)
	}

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

		// Record agent disconnection activity (T038: 使用统一审计)
		if err := m.auditLogger.LogActivity(context.Background(), AuditEntry{
			HostNodeID:  agentConn.HostNodeID,
			UserID:      nil, // System action
			Username:    "",
			Action:      models.ActivityAgentDisconnected,
			ActionType:  models.ActivityTypeAgent,
			Description: "Agent disconnected",
			Metadata:    fmt.Sprintf(`{"uuid":"%s","connected_duration":"%s"}`, uuid, time.Since(agentConn.Connected).String()),
			ClientIP:    "",
			UserAgent:   "",
		}); err != nil {
			logrus.Warnf("Failed to record agent disconnection activity: %v", err)
		}

		// T032: 标记关联的Docker实例为离线
		if m.dockerInstanceService != nil {
			// 获取AgentConnection记录以获取agent ID
			var dbConn models.AgentConnection
			if err := m.db.Where("host_node_id = ?", agentConn.HostNodeID).First(&dbConn).Error; err == nil {
				if err := m.dockerInstanceService.MarkOfflineByAgentID(context.Background(), dbConn.ID); err != nil {
					logrus.WithError(err).Warn("Failed to mark Docker instances offline")
				}
			}
		}

		logrus.Infof("[AgentManager] Agent disconnected: uuid=%s, hostID=%s, duration=%s",
			uuid, agentConn.HostNodeID.String(), time.Since(agentConn.Connected))
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
	} else {
		agentConn.LastDisconnectAt = nil
		agentConn.DisconnectReason = ""
		if status == models.AgentStatusOnline {
			now := time.Now()
			agentConn.ConnectedAt = &now
			agentConn.LastHeartbeat = now
		}
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
			// Log tasks being sent to agent
			if len(tasks) > 0 {
				logrus.Debugf("[TaskDispatch] 服务端发送 %d 个任务给Agent %s", len(tasks), uuid)
				for _, task := range tasks {
					taskType := "unknown"
					if t, ok := task.Params["type"]; ok {
						taskType = t
					}
					target := "unknown"
					if tgt, ok := task.Params["target"]; ok {
						target = tgt
					}
					logrus.Debugf("[TaskDispatch] -> 任务ID: %s, 类型: %s, 目标: %s", task.TaskId, taskType, target)
				}
			}
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

	// Log task queuing
	taskType := "unknown"
	if t, ok := task.Params["type"]; ok {
		taskType = t
	}
	target := "unknown"
	if tgt, ok := task.Params["target"]; ok {
		target = tgt
	}
	logrus.Debugf("[TaskQueue] 服务端将任务加入队列 - Agent: %s, 任务ID: %s, 类型: %s, 目标: %s",
		uuid, task.TaskId, taskType, target)

	// Non-blocking send to avoid deadlock
	select {
	case agentConn.taskQueue <- task:
		logrus.Debugf("[TaskQueue] 任务入队成功 - Agent: %s, 任务ID: %s", uuid, task.TaskId)
		return nil
	default:
		logrus.Errorf("[TaskQueue] 任务队列已满 - Agent: %s, 任务ID: %s", uuid, task.TaskId)
		return fmt.Errorf("task queue full for agent: %s", uuid)
	}
}

// GetAllAgentUUIDs returns all connected agent UUIDs
func (m *AgentManager) GetAllAgentUUIDs() []string {
	var uuids []string
	m.connections.Range(func(key, value interface{}) bool {
		uuids = append(uuids, key.(string))
		return true
	})
	return uuids
}

// discoverDockerInstanceFromProto creates/updates Docker instance from agent's reported Docker info
// T032: Docker实例自动发现（集成Agent架构）
func (m *AgentManager) discoverDockerInstanceFromProto(agentID uuid.UUID, hostID uuid.UUID, dockerInfo *proto.DockerInfo) {
	if m.dockerInstanceService == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get host node to retrieve hostname
	host, err := m.hostRepo.GetByID(ctx, hostID)
	if err != nil {
		logrus.WithError(err).Error("Failed to get host node for Docker instance naming")
		return
	}

	logrus.WithFields(logrus.Fields{
		"agent_id":       agentID,
		"host_id":        hostID,
		"host_name":      host.Name,
		"docker_version": dockerInfo.Version,
	}).Info("Starting Docker instance auto-discovery from agent report")

	// Create info map from proto DockerInfo
	infoMap := map[string]interface{}{
		"version":          dockerInfo.Version,
		"api_version":      dockerInfo.ApiVersion,
		"storage_driver":   dockerInfo.StorageDriver,
		"operating_system": dockerInfo.Os,
		"architecture":     dockerInfo.Arch,
		"kernel_version":   dockerInfo.KernelVersion,
		"mem_total":        dockerInfo.MemTotal,
		"n_cpu":            dockerInfo.Ncpu,
		"containers":       dockerInfo.Containers,
		"images":           dockerInfo.Images,
	}

	// Auto-discover or update Docker instance (use hostname as instance name)
	instance, err := m.dockerInstanceService.AutoDiscoverOrUpdate(ctx, agentID, host.Name, infoMap)
	if err != nil {
		logrus.WithError(err).Error("Failed to auto-discover Docker instance")
		return
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":    instance.ID,
		"instance_name":  instance.Name,
		"agent_id":       agentID,
		"docker_version": dockerInfo.Version,
		"containers":     dockerInfo.Containers,
		"images":         dockerInfo.Images,
	}).Info("Docker instance auto-discovered/updated successfully")
}

// processTaskResults handles task results received from agent
func (m *AgentManager) processTaskResults(uuid string, results []*proto.TaskResult) error {
	conn, ok := m.connections.Load(uuid)
	if !ok {
		return fmt.Errorf("agent connection not found: %s", uuid)
	}

	agentConn := conn.(*AgentConnection)

	for _, result := range results {
		logrus.WithFields(logrus.Fields{
			"agent_uuid": uuid,
			"task_id":    result.TaskId,
			"success":    result.Success,
		}).Debug("[AgentManager] Processing task result")

		// Deliver result to waiting channel if exists
		agentConn.resultsMu.RLock()
		resultChan, exists := agentConn.pendingResults[result.TaskId]
		agentConn.resultsMu.RUnlock()

		if exists {
			// Non-blocking send to avoid deadlock
			select {
			case resultChan <- result:
				logrus.Debugf("[AgentManager] Task result delivered: task_id=%s", result.TaskId)
			default:
				logrus.Warnf("[AgentManager] Result channel full or closed: task_id=%s", result.TaskId)
			}
		} else {
			logrus.Warnf("[AgentManager] No waiting channel for task result: task_id=%s", result.TaskId)
		}
	}

	return nil
}

// QueueTaskAndWait queues a task to the agent and waits for the result
func (m *AgentManager) QueueTaskAndWait(ctx context.Context, uuid string, task *proto.AgentTask, timeout time.Duration) (*proto.TaskResult, error) {
	conn, ok := m.connections.Load(uuid)
	if !ok {
		return nil, fmt.Errorf("agent not connected: %s", uuid)
	}

	agentConn := conn.(*AgentConnection)

	// Create result channel
	resultChan := make(chan *proto.TaskResult, 1)

	// Register the channel
	agentConn.resultsMu.Lock()
	agentConn.pendingResults[task.TaskId] = resultChan
	agentConn.resultsMu.Unlock()

	// Cleanup on exit
	defer func() {
		agentConn.resultsMu.Lock()
		delete(agentConn.pendingResults, task.TaskId)
		agentConn.resultsMu.Unlock()
		close(resultChan)
	}()

	// Queue the task
	if err := m.QueueTask(uuid, task); err != nil {
		return nil, err
	}

	logrus.WithFields(logrus.Fields{
		"agent_uuid": uuid,
		"task_id":    task.TaskId,
		"task_type":  task.TaskType,
		"timeout":    timeout,
	}).Debug("[AgentManager] Waiting for task result")

	// Wait for result with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case result := <-resultChan:
		if result == nil {
			return nil, fmt.Errorf("received nil result for task: %s", task.TaskId)
		}
		logrus.WithFields(logrus.Fields{
			"agent_uuid": uuid,
			"task_id":    task.TaskId,
			"success":    result.Success,
		}).Debug("[AgentManager] Received task result")
		return result, nil
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("timeout waiting for task result: task_id=%s, timeout=%v", task.TaskId, timeout)
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for task result: task_id=%s", task.TaskId)
	}
}

// ArchiveDockerInstancesByHostID archives all Docker instances for a host when it's deleted
// T032: Docker实例生命周期管理
func (m *AgentManager) ArchiveDockerInstancesByHostID(ctx context.Context, hostID uuid.UUID) error {
	if m.dockerInstanceService == nil {
		return nil
	}

	// Get agent connection for this host
	var agentConn models.AgentConnection
	if err := m.db.Where("host_node_id = ?", hostID).First(&agentConn).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// No agent connection found, nothing to archive
			logrus.WithField("host_id", hostID).Debug("No agent connection found for host deletion")
			return nil
		}
		return fmt.Errorf("failed to get agent connection: %w", err)
	}

	// Mark all Docker instances associated with this agent as archived
	if err := m.dockerInstanceService.MarkArchivedByAgentID(ctx, agentConn.ID); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"host_id":  hostID,
			"agent_id": agentConn.ID,
		}).Error("Failed to archive Docker instances")
		return fmt.Errorf("failed to archive Docker instances: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"host_id":  hostID,
		"agent_id": agentConn.ID,
	}).Info("Docker instances archived for deleted host")

	return nil
}
