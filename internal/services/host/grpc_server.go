package host

import (
	"context"

	"github.com/ysicing/tiga/proto"
)

// GRPCServer implements the HostMonitor gRPC service
type GRPCServer struct {
	proto.UnimplementedHostMonitorServer
	agentManager    *AgentManager
	terminalManager *TerminalManager
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(agentManager *AgentManager, terminalManager *TerminalManager) *GRPCServer {
	return &GRPCServer{
		agentManager:    agentManager,
		terminalManager: terminalManager,
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
