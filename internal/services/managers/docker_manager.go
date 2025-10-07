package managers

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"

	"github.com/ysicing/tiga/internal/models"
)

// DockerManager manages Docker instances
type DockerManager struct {
	*BaseManager
	client *client.Client
}

// NewDockerManager creates a new Docker manager
func NewDockerManager() *DockerManager {
	return &DockerManager{
		BaseManager: NewBaseManager(),
	}
}

// Initialize initializes the Docker manager
func (m *DockerManager) Initialize(ctx context.Context, instance *models.Instance) error {
	if err := m.BaseManager.Initialize(ctx, instance); err != nil {
		return err
	}
	return nil
}

// Connect establishes connection to Docker
func (m *DockerManager) Connect(ctx context.Context) error {
	connection := m.instance.Connection
	host, _ := connection["host"].(string)
	port, _ := connection["port"].(float64)

	// Docker API endpoint
	dockerHost := m.GetConfigValue("docker_host", "").(string)
	if dockerHost == "" {
		// Build from host and port if docker_host not specified
		dockerHost = fmt.Sprintf("tcp://%s:%d", host, int(port))
	}

	apiVersion := m.GetConfigValue("api_version", "").(string)

	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithVersion(apiVersion),
		client.WithTimeout(10*time.Second),
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Test connection
	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	m.client = cli
	return nil
}

// Disconnect closes connection to Docker
func (m *DockerManager) Disconnect(ctx context.Context) error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// HealthCheck checks Docker health
func (m *DockerManager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	start := time.Now()
	status := &HealthStatus{
		Healthy:       false,
		LastCheckTime: time.Now().Format(time.RFC3339),
		Details:       make(map[string]interface{}),
	}

	// Ping Docker daemon
	ping, err := m.client.Ping(ctx)
	if err != nil {
		status.Message = fmt.Sprintf("Health check failed: %v", err)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status, nil
	}

	status.Details["api_version"] = ping.APIVersion
	status.Details["os_type"] = ping.OSType

	// Get Docker info
	info, err := m.client.Info(ctx)
	if err == nil {
		status.Details["containers"] = info.Containers
		status.Details["containers_running"] = info.ContainersRunning
		status.Details["containers_paused"] = info.ContainersPaused
		status.Details["containers_stopped"] = info.ContainersStopped
		status.Details["images"] = info.Images
	}

	status.Healthy = true
	status.Message = "Docker is healthy"
	status.ResponseTime = time.Since(start).Milliseconds()

	return status, nil
}

// CollectMetrics collects metrics from Docker
func (m *DockerManager) CollectMetrics(ctx context.Context) (*ServiceMetrics, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	metrics := &ServiceMetrics{
		InstanceID: m.instance.ID,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metrics:    make(map[string]interface{}),
	}

	// Get Docker info
	info, err := m.client.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricsCollectionFailed, err)
	}

	metrics.Metrics["containers_total"] = info.Containers
	metrics.Metrics["containers_running"] = info.ContainersRunning
	metrics.Metrics["containers_paused"] = info.ContainersPaused
	metrics.Metrics["containers_stopped"] = info.ContainersStopped
	metrics.Metrics["images"] = info.Images
	metrics.Metrics["driver"] = info.Driver
	metrics.Metrics["memory_total"] = info.MemTotal
	metrics.Metrics["cpus"] = info.NCPU

	// Get container stats
	containers, err := m.client.ContainerList(ctx, container.ListOptions{All: true})
	if err == nil {
		var totalCPUPercent, totalMemoryUsage float64
		containerMetrics := make([]map[string]interface{}, 0, len(containers))

		for _, c := range containers {
			stats, err := m.client.ContainerStats(ctx, c.ID, false)
			if err != nil {
				continue
			}

			if err := stats.Body.Close(); err == nil {
				containerMetrics = append(containerMetrics, map[string]interface{}{
					"id":     c.ID[:12],
					"name":   c.Names[0],
					"image":  c.Image,
					"state":  c.State,
					"status": c.Status,
				})
			}
		}

		metrics.Metrics["container_metrics"] = containerMetrics
		metrics.Metrics["avg_cpu_percent"] = totalCPUPercent / float64(len(containers))
		metrics.Metrics["total_memory_usage"] = totalMemoryUsage
	}

	// Get volume count
	volumes, err := m.client.VolumeList(ctx, volume.ListOptions{})
	if err == nil {
		metrics.Metrics["volumes"] = len(volumes.Volumes)
	}

	// Get network count
	networks, err := m.client.NetworkList(ctx, types.NetworkListOptions{})
	if err == nil {
		metrics.Metrics["networks"] = len(networks)
	}

	return metrics, nil
}

// GetInfo returns Docker service information
func (m *DockerManager) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	info := make(map[string]interface{})

	// Get Docker version
	version, err := m.client.ServerVersion(ctx)
	if err != nil {
		return nil, err
	}

	info["version"] = version.Version
	info["api_version"] = version.APIVersion
	info["os"] = version.Os
	info["arch"] = version.Arch
	info["kernel_version"] = version.KernelVersion
	info["git_commit"] = version.GitCommit
	info["build_time"] = version.BuildTime

	// Get Docker info
	dockerInfo, err := m.client.Info(ctx)
	if err == nil {
		info["name"] = dockerInfo.Name
		info["containers"] = dockerInfo.Containers
		info["images"] = dockerInfo.Images
		info["driver"] = dockerInfo.Driver
		info["memory_total"] = dockerInfo.MemTotal
		info["cpus"] = dockerInfo.NCPU
		info["server_version"] = dockerInfo.ServerVersion
	}

	// List containers
	containers, err := m.client.ContainerList(ctx, container.ListOptions{All: true})
	if err == nil {
		containerList := make([]map[string]interface{}, 0, len(containers))
		for _, c := range containers {
			containerList = append(containerList, map[string]interface{}{
				"id":      c.ID[:12],
				"names":   c.Names,
				"image":   c.Image,
				"state":   c.State,
				"status":  c.Status,
				"created": c.Created,
			})
		}
		info["container_list"] = containerList
	}

	return info, nil
}

// ValidateConfig validates Docker configuration
func (m *DockerManager) ValidateConfig(config map[string]interface{}) error {
	// Docker connection is flexible, no strict requirements
	return nil
}

// Type returns the service type
func (m *DockerManager) Type() string {
	return "docker"
}
