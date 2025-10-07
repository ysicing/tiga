package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ysicing/tiga/cmd/tiga-agent/collector"
	"github.com/ysicing/tiga/proto"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

type Config struct {
	ServerAddr string
	UUID       string
	SecretKey  string
	LogLevel   string
}

func main() {
	config := parseFlags()
	setupLogger(config.LogLevel)

	logrus.Infof("Tiga Agent %s (built at %s)", version, buildTime)
	logrus.Infof("Connecting to server: %s", config.ServerAddr)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Connect to gRPC server
	conn, err := connectToServer(ctx, config)
	if err != nil {
		logrus.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	logrus.Info("Successfully connected to server")

	// Create gRPC client
	client := proto.NewHostMonitorClient(conn)

	// Initialize collector
	col := collector.NewCollector(version)

	// Register agent and get host info
	if err := registerAgent(ctx, client, config, col); err != nil {
		logrus.Fatalf("Failed to register agent: %v", err)
	}

	// Start reporting loop
	go runReportingLoop(ctx, client, col, config)

	// Wait for shutdown signal
	<-sigCh
	logrus.Info("Shutting down gracefully...")
	cancel()

	// Give some time for cleanup
	time.Sleep(2 * time.Second)
	logrus.Info("Agent stopped")
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.ServerAddr, "server", "localhost:12307", "Server gRPC address")
	flag.StringVar(&config.UUID, "uuid", "", "Host UUID")
	flag.StringVar(&config.SecretKey, "key", "", "Secret key for authentication")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Tiga Agent %s\nBuild time: %s\n", version, buildTime)
		os.Exit(0)
	}

	if config.UUID == "" {
		logrus.Fatal("UUID is required (use --uuid flag)")
	}

	if config.SecretKey == "" {
		logrus.Fatal("Secret key is required (use --key flag)")
	}

	return config
}

func setupLogger(level string) {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Warn("Invalid log level, using info")
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
}

func connectToServer(ctx context.Context, config *Config) (*grpc.ClientConn, error) {
	// TODO: Add TLS support
	// For now, use insecure connection for development
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.ServerAddr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return conn, nil
}

// registerAgent registers the agent with the server and sends host info
func registerAgent(ctx context.Context, client proto.HostMonitorClient, config *Config, col *collector.Collector) error {
	// Collect host info
	hostInfo, err := col.CollectHostInfo()
	if err != nil {
		return fmt.Errorf("failed to collect host info: %w", err)
	}

	// Convert to proto
	protoHostInfo := &proto.HostInfo{
		Platform:        hostInfo.Platform,
		PlatformVersion: hostInfo.PlatformVersion,
		Arch:            hostInfo.Arch,
		Virtualization:  hostInfo.Virtualization,
		CpuModel:        hostInfo.CPUModel,
		CpuCores:        int32(hostInfo.CPUCores),
		MemTotal:        hostInfo.MemTotal,
		DiskTotal:       hostInfo.DiskTotal,
		SwapTotal:       hostInfo.SwapTotal,
		AgentVersion:    hostInfo.AgentVersion,
		BootTime:        int64(hostInfo.BootTime),
		SshEnabled:      hostInfo.SSHEnabled,
		SshPort:         int32(hostInfo.SSHPort),
		SshUser:         hostInfo.SSHUser,
	}

	// Register with server
	resp, err := client.RegisterAgent(ctx, &proto.RegisterAgentRequest{
		Uuid:      config.UUID,
		SecretKey: config.SecretKey,
		HostInfo:  protoHostInfo,
	})

	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration rejected: %s", resp.Message)
	}

	logrus.Infof("Agent registered successfully: %s", resp.Message)
	return nil
}

// runReportingLoop continuously collects and reports host state
func runReportingLoop(ctx context.Context, client proto.HostMonitorClient, col *collector.Collector, config *Config) {
	// Create the bidirectional stream
	stream, err := client.ReportState(ctx)
	if err != nil {
		logrus.Errorf("Failed to create report stream: %v", err)
		return
	}

	// Start receiving responses in a separate goroutine
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				logrus.Info("Server closed the stream")
				return
			}
			if err != nil {
				logrus.Errorf("Error receiving response: %v", err)
				return
			}
			if !resp.Success {
				logrus.Warnf("Server response: %s", resp.Message)
			}

			// Handle tasks from server
			for _, task := range resp.Tasks {
				go handleTask(client, task)
			}
		}
	}()

	// Report state every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Report immediately on start
	reportState(stream, col, config)

	for {
		select {
		case <-ctx.Done():
			stream.CloseSend()
			return
		case <-ticker.C:
			reportState(stream, col, config)
		}
	}
}

// reportState collects current state and sends it to server
func reportState(stream proto.HostMonitor_ReportStateClient, col *collector.Collector, config *Config) {
	state, err := col.CollectHostState()
	if err != nil {
		logrus.Errorf("Failed to collect host state: %v", err)
		return
	}

	// Convert to proto
	protoState := &proto.HostState{
		Timestamp:        time.Now().UnixMilli(),
		CpuUsage:         state.CPUUsage,
		Load_1:           state.Load1,
		Load_5:           state.Load5,
		Load_15:          state.Load15,
		MemUsed:          state.MemUsed,
		MemUsage:         state.MemUsage,
		SwapUsed:         state.SwapUsed,
		DiskUsed:         state.DiskUsed,
		DiskUsage:        state.DiskUsage,
		NetInTransfer:    state.NetInTransfer,
		NetOutTransfer:   state.NetOutTransfer,
		NetInSpeed:       state.NetInSpeed,
		NetOutSpeed:      state.NetOutSpeed,
		TcpConnCount:     int32(state.TCPConnCount),
		UdpConnCount:     int32(state.UDPConnCount),
		ProcessCount:     int32(state.ProcessCount),
		Uptime:           int64(state.Uptime),
		GpuUsage:         state.GPUUsage,
		TrafficSent:      state.TrafficSent,
		TrafficRecv:      state.TrafficRecv,
		TrafficDeltaSent: state.TrafficDeltaSent,
		TrafficDeltaRecv: state.TrafficDeltaRecv,
	}

	// Send to server
	req := &proto.ReportStateRequest{
		Uuid:  config.UUID,
		State: protoState,
	}

	if err := stream.Send(req); err != nil {
		logrus.Errorf("Failed to send state: %v", err)
		return
	}

	logrus.Debugf("Reported state: CPU=%.2f%%, Mem=%.2f%%, Disk=%.2f%%, Traffic=%d/%d bytes",
		state.CPUUsage, state.MemUsage, state.DiskUsage,
		state.TrafficDeltaSent, state.TrafficDeltaRecv)
}

// handleTask processes tasks sent by the server
func handleTask(client proto.HostMonitorClient, task *proto.AgentTask) {
	logrus.Infof("Received task: id=%s type=%s", task.TaskId, task.TaskType)

	switch task.TaskType {
	case "terminal":
		// Get streamID from task params
		streamID, ok := task.Params["stream_id"]
		if !ok {
			logrus.Errorf("Terminal task missing stream_id parameter")
			return
		}
		// Start terminal session
		handleTerminalSession(client, streamID)

	case "probe":
		// TODO: Handle service probe tasks
		logrus.Warnf("Probe tasks not yet implemented")

	case "command":
		// TODO: Handle command execution tasks
		logrus.Warnf("Command tasks not yet implemented")

	default:
		logrus.Warnf("Unknown task type: %s", task.TaskType)
	}
}
