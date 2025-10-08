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
	ServerAddr     string
	UUID           string
	SecretKey      string
	LogLevel       string
	ReportInterval int  // Report interval in seconds
	DisableWebSSH  bool // Disable WebSSH terminal functionality
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

	// Start agent with reconnection loop
	go runAgentWithReconnect(ctx, config)

	// Wait for shutdown signal
	<-sigCh
	logrus.Info("Shutting down gracefully...")
	cancel()

	// Give some time for cleanup
	time.Sleep(2 * time.Second)
	logrus.Info("Agent stopped")
}

// runAgentWithReconnect runs the agent with automatic reconnection
func runAgentWithReconnect(ctx context.Context, config *Config) {
	retryDelay := 5 * time.Second
	maxRetryDelay := 5 * time.Minute
	backoffFactor := 2.0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Connect to gRPC server
		conn, err := connectToServer(ctx, config)
		if err != nil {
			logrus.Errorf("Failed to connect to server: %v", err)
			logrus.Infof("Retrying in %v...", retryDelay)

			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
				// Exponential backoff
				retryDelay = time.Duration(float64(retryDelay) * backoffFactor)
				if retryDelay > maxRetryDelay {
					retryDelay = maxRetryDelay
				}
				continue
			}
		}

		logrus.Info("Successfully connected to server")
		retryDelay = 5 * time.Second // Reset retry delay on successful connection

		// Create gRPC client
		client := proto.NewHostMonitorClient(conn)

		// Initialize collector
		col := collector.NewCollector(version)

		// Register agent and get host info
		if err := registerAgent(ctx, client, config, col); err != nil {
			logrus.Errorf("Failed to register agent: %v", err)
			conn.Close()

			logrus.Infof("Retrying in %v...", retryDelay)
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
				retryDelay = time.Duration(float64(retryDelay) * backoffFactor)
				if retryDelay > maxRetryDelay {
					retryDelay = maxRetryDelay
				}
				continue
			}
		}

		// Start reporting loop
		runReportingLoop(ctx, client, col, config)

		// If we reach here, the reporting loop ended (connection lost)
		conn.Close()
		logrus.Warn("Connection lost, reconnecting...")

		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
			// Exponential backoff
			retryDelay = time.Duration(float64(retryDelay) * backoffFactor)
			if retryDelay > maxRetryDelay {
				retryDelay = maxRetryDelay
			}
		}
	}
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.ServerAddr, "server", "localhost:12307", "Server gRPC address")
	flag.StringVar(&config.UUID, "uuid", "", "Host UUID")
	flag.StringVar(&config.SecretKey, "key", "", "Secret key for authentication")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.IntVar(&config.ReportInterval, "interval", 30, "Report interval in seconds (default: 30)")
	flag.BoolVar(&config.DisableWebSSH, "disable-webssh", false, "Disable WebSSH terminal functionality")

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

	// Validate report interval
	if config.ReportInterval < 5 {
		logrus.Warn("Report interval too small, setting to minimum: 5 seconds")
		config.ReportInterval = 5
	}
	if config.ReportInterval > 300 {
		logrus.Warn("Report interval too large, setting to maximum: 300 seconds")
		config.ReportInterval = 300
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
		grpc.WithBlock(), // Block until connection established or timeout
	}

	// Use shorter timeout for reconnection attempts
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
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
		SshEnabled:      !config.DisableWebSSH, // WebSSH is enabled unless explicitly disabled
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

	// Channel to signal stream errors
	streamErr := make(chan error, 1)

	// Start receiving responses in a separate goroutine
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				logrus.Info("Server closed the stream")
				streamErr <- io.EOF
				return
			}
			if err != nil {
				logrus.Errorf("Error receiving response: %v", err)
				streamErr <- err
				return
			}
			if !resp.Success {
				logrus.Warnf("Server response: %s", resp.Message)
			}

			// Handle tasks from server
			for _, task := range resp.Tasks {
				go handleTask(client, task, config)
			}
		}
	}()

	// Report state at configured interval
	ticker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
	defer ticker.Stop()

	// Report immediately on start
	if err := reportState(stream, col, config); err != nil {
		logrus.Errorf("Failed to send initial state: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			stream.CloseSend()
			return
		case err := <-streamErr:
			logrus.Warnf("Stream error detected: %v, will reconnect", err)
			stream.CloseSend()
			return
		case <-ticker.C:
			if err := reportState(stream, col, config); err != nil {
				logrus.Errorf("Failed to send state: %v, will reconnect", err)
				stream.CloseSend()
				return
			}
		}
	}
}

// reportState collects current state and sends it to server
func reportState(stream proto.HostMonitor_ReportStateClient, col *collector.Collector, config *Config) error {
	state, err := col.CollectHostState()
	if err != nil {
		return fmt.Errorf("failed to collect host state: %w", err)
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
		return fmt.Errorf("failed to send state: %w", err)
	}

	logrus.Debugf("Reported state: CPU=%.2f%%, Mem=%.2f%%, Disk=%.2f%%, Traffic=%d/%d bytes",
		state.CPUUsage, state.MemUsage, state.DiskUsage,
		state.TrafficDeltaSent, state.TrafficDeltaRecv)

	return nil
}

// handleTask processes tasks sent by the server
func handleTask(client proto.HostMonitorClient, task *proto.AgentTask, config *Config) {
	logrus.Infof("[Task] Received task: id=%s type=%s", task.TaskId, task.TaskType)
	logrus.Debugf("[Task] Task params: %+v", task.Params)

	switch task.TaskType {
	case "terminal":
		// Check if WebSSH is disabled
		if config.DisableWebSSH {
			logrus.Warnf("[Task:Terminal] Terminal task rejected: WebSSH functionality is disabled (--disable-webssh)")
			return
		}
		// Get streamID from task params
		streamID, ok := task.Params["stream_id"]
		if !ok {
			logrus.Errorf("[Task:Terminal] Terminal task missing stream_id parameter")
			return
		}
		logrus.Infof("[Task:Terminal] Starting terminal session: stream_id=%s", streamID)
		// Start terminal session
		handleTerminalSession(client, streamID)
		logrus.Infof("[Task:Terminal] Terminal session finished: stream_id=%s", streamID)

	case "probe":
		// TODO: Handle service probe tasks
		logrus.Warnf("[Task:Probe] Probe tasks not yet implemented")

	case "command":
		// TODO: Handle command execution tasks
		logrus.Warnf("[Task:Command] Command tasks not yet implemented")

	default:
		logrus.Warnf("[Task] Unknown task type: %s", task.TaskType)
	}
}
