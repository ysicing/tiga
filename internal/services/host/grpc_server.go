package host

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/services/monitor"
	"github.com/ysicing/tiga/proto"
)

// GRPCServer implements the HostMonitor gRPC service
type GRPCServer struct {
	proto.UnimplementedHostMonitorServer
	agentManager        *AgentManager
	terminalManager     *TerminalManager
	dockerStreamManager *DockerStreamManager
	probeScheduler      *monitor.ServiceProbeScheduler
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(agentManager *AgentManager, terminalManager *TerminalManager, dockerStreamManager *DockerStreamManager, probeScheduler *monitor.ServiceProbeScheduler) *GRPCServer {
	return &GRPCServer{
		agentManager:        agentManager,
		terminalManager:     terminalManager,
		dockerStreamManager: dockerStreamManager,
		probeScheduler:      probeScheduler,
	}
}

// RegisterAgent handles agent registration
func (s *GRPCServer) RegisterAgent(ctx context.Context, req *proto.RegisterAgentRequest) (*proto.RegisterAgentResponse, error) {
	return s.agentManager.RegisterAgent(ctx, req)
}

// ReportState handles state reporting stream
func (s *GRPCServer) ReportState(stream proto.HostMonitor_ReportStateServer) error {
	return s.agentManager.HandleReportState(stream)
}

// Heartbeat handles heartbeat requests
func (s *GRPCServer) Heartbeat(ctx context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	return s.agentManager.Heartbeat(ctx, req)
}

// IOStream handles terminal I/O stream
func (s *GRPCServer) IOStream(stream proto.HostMonitor_IOStreamServer) error {
	return s.terminalManager.HandleIOStream(stream)
}

// DockerStream handles Docker streaming operations
func (s *GRPCServer) DockerStream(stream proto.HostMonitor_DockerStreamServer) error {
	return s.dockerStreamManager.HandleDockerStream(stream)
}

// ReportProbeResultBatch handles batch probe result reporting from Agents
func (s *GRPCServer) ReportProbeResultBatch(ctx context.Context, req *proto.ReportProbeResultBatchRequest) (*proto.ReportProbeResultBatchResponse, error) {
	// Validate request
	if req.Uuid == "" {
		return &proto.ReportProbeResultBatchResponse{
			Success:   false,
			Message:   "UUID is required",
			Processed: 0,
			Failed:    0,
		}, nil
	}

	if len(req.Results) == 0 {
		return &proto.ReportProbeResultBatchResponse{
			Success:   false,
			Message:   "No results to process",
			Processed: 0,
			Failed:    0,
		}, nil
	}

	// Parse host node UUID once
	hostNodeID, err := uuid.Parse(req.Uuid)
	if err != nil {
		logrus.Errorf("Invalid host node UUID: %v", err)
		return &proto.ReportProbeResultBatchResponse{
			Success:   false,
			Message:   fmt.Sprintf("Invalid host node UUID: %v", err),
			Processed: 0,
			Failed:    int32(len(req.Results)),
		}, nil
	}

	// Convert batch results
	reports := make([]*monitor.ProbeReport, 0, len(req.Results))
	var processedCount, failedCount int32

	for _, item := range req.Results {
		// Validate individual result
		if item.ServiceMonitorId == "" {
			failedCount++
			logrus.Warnf("Skipping result with empty service monitor ID")
			continue
		}

		if item.Result == nil {
			failedCount++
			logrus.Warnf("Skipping result with nil probe result")
			continue
		}

		// Parse service monitor ID
		serviceMonitorID, err := uuid.Parse(item.ServiceMonitorId)
		if err != nil {
			failedCount++
			logrus.Warnf("Invalid service monitor ID: %v", err)
			continue
		}

		// Convert to monitor.ProbeReport
		report := &monitor.ProbeReport{
			ServiceMonitorID: serviceMonitorID,
			HostNodeID:       hostNodeID,
			Success:          item.Result.Success,
			Latency:          float32(item.Result.Latency),
			Timestamp:        time.UnixMilli(item.Result.Timestamp),
			ErrorMessage:     item.Result.ErrorMessage,
			Data:             item.Result.HttpResponseBody,
		}

		reports = append(reports, report)
		processedCount++
	}

	// Report batch to ServiceSentinel for aggregation
	if s.probeScheduler != nil && len(reports) > 0 {
		s.probeScheduler.ReportAgentProbeResultBatch(reports)
	}

	logrus.Debugf("Received batch probe results from Agent %s: total=%d, processed=%d, failed=%d",
		req.Uuid, len(req.Results), processedCount, failedCount)

	return &proto.ReportProbeResultBatchResponse{
		Success:   true,
		Message:   fmt.Sprintf("Batch processed: %d succeeded, %d failed", processedCount, failedCount),
		Processed: processedCount,
		Failed:    failedCount,
	}, nil
}
