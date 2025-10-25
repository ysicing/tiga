package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ysicing/tiga/cmd/tiga-agent/collector"
	"github.com/ysicing/tiga/proto"

	dockerclient "github.com/docker/docker/client"
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

	flag.StringVar(&config.ServerAddr, "server", "localhost:12307", "Server gRPC address (host:port format)")
	flag.StringVar(&config.UUID, "uuid", "", "Host UUID")
	flag.StringVar(&config.SecretKey, "key", "", "Secret key for authentication")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.IntVar(&config.ReportInterval, "interval", 10, "Report interval in seconds (default: 30)")
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

	// Strip protocol prefix from server address if present (common mistake)
	if len(config.ServerAddr) > 0 {
		// Remove http:// or https:// prefix
		if len(config.ServerAddr) > 7 && config.ServerAddr[:7] == "http://" {
			config.ServerAddr = config.ServerAddr[7:]
			logrus.Warnf("Removed 'http://' prefix from server address, using: %s", config.ServerAddr)
		} else if len(config.ServerAddr) > 8 && config.ServerAddr[:8] == "https://" {
			config.ServerAddr = config.ServerAddr[8:]
			logrus.Warnf("Removed 'https://' prefix from server address, using: %s", config.ServerAddr)
		}
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
	// Auto-detect TLS based on port
	useTLS := shouldUseTLS(config.ServerAddr)

	var dialOpts []grpc.DialOption

	if useTLS {
		// Use TLS credentials (skip server certificate verification for self-signed certs)
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Allow self-signed certificates
		}
		creds := credentials.NewTLS(tlsConfig)
		dialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(creds),
			grpc.WithBlock(), // Block until connection established or timeout
		}
		logrus.Infof("Using TLS connection (detected port 443/8443)")
	} else {
		// Use insecure connection for development
		dialOpts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(), // Block until connection established or timeout
		}
		logrus.Info("Using insecure connection (non-TLS port)")
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

// shouldUseTLS determines if TLS should be used based on the port
func shouldUseTLS(serverAddr string) bool {
	// Extract port from address (format: host:port)
	parts := strings.Split(serverAddr, ":")
	if len(parts) < 2 {
		return false
	}

	port := parts[len(parts)-1]
	// Auto-enable TLS for standard HTTPS/gRPC-TLS ports
	return port == "443" || port == "8443"
}

// registerAgent registers the agent with the server and sends host info
func registerAgent(ctx context.Context, client proto.HostMonitorClient, config *Config, col *collector.Collector) error {
	// Collect host info
	hostInfo, err := col.CollectHostInfo()
	if err != nil {
		return fmt.Errorf("failed to collect host info: %w", err)
	}

	// Collect Docker info
	dockerInfo := col.CollectDockerInfo()

	// Convert Docker info to proto
	var protoDockerInfo *proto.DockerInfo
	if dockerInfo.Installed {
		protoDockerInfo = &proto.DockerInfo{
			Installed:         true,
			Version:           dockerInfo.Version,
			ApiVersion:        dockerInfo.APIVersion,
			Os:                dockerInfo.OS,
			Arch:              dockerInfo.Arch,
			KernelVersion:     dockerInfo.KernelVersion,
			StorageDriver:     dockerInfo.StorageDriver,
			Containers:        dockerInfo.Containers,
			ContainersRunning: dockerInfo.ContainersRunning,
			ContainersPaused:  dockerInfo.ContainersPaused,
			ContainersStopped: dockerInfo.ContainersStopped,
			Images:            dockerInfo.Images,
			MemTotal:          dockerInfo.MemTotal,
			Ncpu:              dockerInfo.NCPU,
		}
		logrus.WithFields(logrus.Fields{
			"version":    dockerInfo.Version,
			"containers": dockerInfo.Containers,
			"images":     dockerInfo.Images,
		}).Info("Docker detected and will be reported to server")
	} else {
		protoDockerInfo = &proto.DockerInfo{
			Installed: false,
		}
		logrus.Debug("Docker not detected on this host")
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
		DockerInfo:      protoDockerInfo,
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
	// Create and start probe handler for batch reporting
	probeHandler := NewProbeTaskHandler(config.UUID, client)
	probeHandler.Start()
	defer probeHandler.Stop() // Ensure buffer is flushed on exit

	// Initialize Docker task handler if Docker is available
	var dockerHandler *DockerTaskHandler
	var dockerStreamHandler *DockerStreamHandler

	if dockerInfo := col.CollectDockerInfo(); dockerInfo.Installed {
		var err error
		dockerHandler, err = NewDockerTaskHandler()
		if err != nil {
			logrus.WithError(err).Warn("Failed to initialize Docker task handler at startup, will retry on first Docker task")
		} else {
			defer dockerHandler.Close()
			logrus.WithFields(logrus.Fields{
				"docker_version": dockerInfo.Version,
				"api_version":    dockerInfo.APIVersion,
				"containers":     dockerInfo.Containers,
				"images":         dockerInfo.Images,
			}).Info("Docker task handler initialized successfully")

			// Create Docker stream handler using the same Docker client
			dockerStreamHandler = NewDockerStreamHandler(dockerHandler.dockerClient)
			logrus.Info("Docker stream handler initialized successfully")
		}
	} else {
		logrus.Info("Docker daemon not detected (this is normal if Docker is not installed or not running)")
	}

	// Create the bidirectional stream
	stream, err := client.ReportState(ctx)
	if err != nil {
		logrus.Errorf("Failed to create report stream: %v", err)
		return
	}

	// Channel to signal stream errors
	streamErr := make(chan error, 1)

	// Channel for task results
	taskResults := make(chan *proto.TaskResult, 100)

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
				go handleTask(client, probeHandler, dockerHandler, dockerStreamHandler, taskResults, task, config)
			}
		}
	}()

	// Report state at configured interval
	ticker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
	defer ticker.Stop()

	// Report immediately on start
	if err := reportState(stream, col, config, nil); err != nil {
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
			// Collect any pending task results
			var results []*proto.TaskResult
			for {
				select {
				case result := <-taskResults:
					results = append(results, result)
				default:
					goto sendState
				}
			}
		sendState:
			if err := reportState(stream, col, config, results); err != nil {
				logrus.Errorf("Failed to send state: %v, will reconnect", err)
				stream.CloseSend()
				return
			}
		}
	}
}

// reportState collects current state and sends it to server
func reportState(stream proto.HostMonitor_ReportStateClient, col *collector.Collector, config *Config, taskResults []*proto.TaskResult) error {
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

	// Send to server with task results
	req := &proto.ReportStateRequest{
		Uuid:        config.UUID,
		State:       protoState,
		TaskResults: taskResults,
	}

	if err := stream.Send(req); err != nil {
		return fmt.Errorf("failed to send state: %w", err)
	}

	logrus.Debugf("Reported state: CPU=%.2f%%, Mem=%.2f%%, Disk=%.2f%%, Traffic=%d/%d bytes",
		state.CPUUsage, state.MemUsage, state.DiskUsage,
		state.TrafficDeltaSent, state.TrafficDeltaRecv)

	if len(taskResults) > 0 {
		logrus.Debugf("Sent %d task results with state report", len(taskResults))
	}

	return nil
}

// handleTask processes tasks sent by the server
func handleTask(client proto.HostMonitorClient, probeHandler *ProbeTaskHandler, dockerHandler *DockerTaskHandler, dockerStreamHandler *DockerStreamHandler, taskResults chan<- *proto.TaskResult, task *proto.AgentTask, config *Config) {
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
		// Handle service probe tasks (using shared probe handler for batch reporting)
		probeHandler.HandleProbeTask(task, config)

	case "docker":
		// Handle Docker tasks with lazy initialization
		if dockerHandler == nil {
			// Try to initialize Docker handler on-demand
			logrus.Info("[Task:Docker] Docker handler not initialized, attempting lazy initialization...")

			handler, err := tryInitializeDockerHandler()
			if err != nil {
				// Provide diagnostic information for Docker unavailability
				diagMsg := diagnoseDockerIssue()
				errMsg := fmt.Sprintf("Docker handler initialization failed: %v. Diagnosis: %s", err, diagMsg)

				logrus.Warnf("[Task:Docker] %s", errMsg)

				// Send error result with diagnostic information
				taskResults <- &proto.TaskResult{
					TaskId:    task.TaskId,
					Success:   false,
					Error:     errMsg,
					Timestamp: time.Now().UnixMilli(),
				}
				return
			}

			// Successfully initialized, update the handler
			dockerHandler = handler
			dockerStreamHandler = NewDockerStreamHandler(dockerHandler.dockerClient)
			logrus.Info("[Task:Docker] Docker handler lazy initialization successful")
		}

		logrus.Infof("[Task:Docker] Processing Docker task: %s", task.Params["operation"])
		result := dockerHandler.HandleDockerTask(task)
		taskResults <- result
		if result.Success {
			logrus.Infof("[Task:Docker] Task completed successfully: id=%s", task.TaskId)
		} else {
			logrus.Errorf("[Task:Docker] Task failed: id=%s error=%s", task.TaskId, result.Error)
		}

	case "docker_stream":
		// Handle Docker stream tasks
		if dockerStreamHandler == nil {
			logrus.Warnf("[Task:DockerStream] Docker stream handler not available")
			return
		}
		logrus.Infof("[Task:DockerStream] Starting Docker stream session")
		// Initiate DockerStream connection
		go handleDockerStreamSession(client, dockerStreamHandler, task)

	case "command":
		// TODO: Handle command execution tasks
		logrus.Warnf("[Task:Command] Command tasks not yet implemented")

	default:
		logrus.Warnf("[Task] Unknown task type: %s", task.TaskType)
	}
}

// handleDockerStreamSession initiates and handles a Docker stream connection
func handleDockerStreamSession(client proto.HostMonitorClient, dockerStreamHandler *DockerStreamHandler, task *proto.AgentTask) {
	logrus.Info("[DockerStream] Initiating DockerStream connection to server")

	// Extract session_id from task params
	sessionID := task.Params["session_id"]
	if sessionID == "" {
		logrus.Error("[DockerStream] Missing session_id in task params")
		return
	}

	// Create Docker stream
	stream, err := client.DockerStream(context.Background())
	if err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to create DockerStream")
		return
	}

	logrus.Infof("[DockerStream] DockerStream connection established, session_id: %s", sessionID)

	// Send initial message to server with session_id
	initMsg := &proto.DockerStreamMessage{
		Message: &proto.DockerStreamMessage_Init{
			Init: &proto.DockerStreamInit{
				SessionId: sessionID,
			},
		},
	}

	if err := stream.Send(initMsg); err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to send init message")
		return
	}

	logrus.Info("[DockerStream] Sent init message to server")

	// Handle the stream (this is a blocking call)
	if err := dockerStreamHandler.HandleDockerStream(stream); err != nil {
		logrus.WithError(err).Error("[DockerStream] Stream handling error")
	}

	logrus.Info("[DockerStream] DockerStream connection closed")
}

// tryInitializeDockerHandler attempts to initialize Docker handler
// This is used for lazy initialization when a Docker task is received
func tryInitializeDockerHandler() (*DockerTaskHandler, error) {
	handler, err := NewDockerTaskHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Docker handler: %w", err)
	}
	return handler, nil
}

// diagnoseDockerIssue provides diagnostic information when Docker is unavailable
func diagnoseDockerIssue() string {
	// Try to create a Docker client to get specific error
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Sprintf("Docker client creation failed: %v. Ensure Docker is installed and DOCKER_HOST is correctly configured", err)
	}
	defer cli.Close()

	// Try to ping Docker daemon
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return "Docker daemon connection failed: Permission denied. Ensure the agent user is in the 'docker' group or has access to /var/run/docker.sock"
		}
		if strings.Contains(err.Error(), "connection refused") {
			return "Docker daemon connection failed: Connection refused. Ensure Docker daemon is running"
		}
		if strings.Contains(err.Error(), "no such file") {
			return "Docker socket not found at /var/run/docker.sock. Docker may not be installed"
		}
		return fmt.Sprintf("Docker daemon ping failed: %v", err)
	}

	// If we can ping, try to get version to check API compatibility
	version, err := cli.ServerVersion(ctx)
	if err != nil {
		return fmt.Sprintf("Docker version check failed: %v", err)
	}

	// Check if API version is too old
	if version.APIVersion < "1.41" {
		return fmt.Sprintf("Docker API version %s is too old (minimum required: 1.41 / Docker 20.10+). Please upgrade Docker", version.APIVersion)
	}

	return "Docker appears to be available now. Please retry the operation"
}
