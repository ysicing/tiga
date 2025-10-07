package managers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
)

// ListContainers lists Docker containers
func (m *DockerManager) ListContainers(ctx context.Context, all bool) ([]types.Container, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	containers, err := m.client.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}

// GetContainer gets container details
func (m *DockerManager) GetContainer(ctx context.Context, containerID string) (*types.ContainerJSON, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	containerJSON, err := m.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return &containerJSON, nil
}

// CreateContainer creates a new container
func (m *DockerManager) CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, containerName string) (string, error) {
	if m.client == nil {
		return "", ErrNotConnected
	}

	resp, err := m.client.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// StartContainer starts a container
func (m *DockerManager) StartContainer(ctx context.Context, containerID string) error {
	if m.client == nil {
		return ErrNotConnected
	}

	if err := m.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

// StopContainer stops a container
func (m *DockerManager) StopContainer(ctx context.Context, containerID string, timeout *int) error {
	if m.client == nil {
		return ErrNotConnected
	}

	var stopOptions container.StopOptions
	if timeout != nil {
		stopOptions.Timeout = timeout
	}

	if err := m.client.ContainerStop(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

// RestartContainer restarts a container
func (m *DockerManager) RestartContainer(ctx context.Context, containerID string, timeout *int) error {
	if m.client == nil {
		return ErrNotConnected
	}

	var stopOptions container.StopOptions
	if timeout != nil {
		stopOptions.Timeout = timeout
	}

	if err := m.client.ContainerRestart(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	return nil
}

// RemoveContainer removes a container
func (m *DockerManager) RemoveContainer(ctx context.Context, containerID string, force bool, removeVolumes bool) error {
	if m.client == nil {
		return ErrNotConnected
	}

	options := container.RemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	}

	if err := m.client.ContainerRemove(ctx, containerID, options); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

// GetContainerLogs gets container logs
func (m *DockerManager) GetContainerLogs(ctx context.Context, containerID string, showStdout, showStderr bool, tail string, since string) (io.ReadCloser, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	options := container.LogsOptions{
		ShowStdout: showStdout,
		ShowStderr: showStderr,
		Tail:       tail,
		Since:      since,
		Timestamps: true,
	}

	logs, err := m.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	return logs, nil
}

// GetContainerStats gets container statistics
func (m *DockerManager) GetContainerStats(ctx context.Context, containerID string) (types.StatsJSON, error) {
	if m.client == nil {
		return types.StatsJSON{}, ErrNotConnected
	}

	stats, err := m.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return types.StatsJSON{}, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	var statsJSON types.StatsJSON
	decoder := json.NewDecoder(stats.Body)
	if err := decoder.Decode(&statsJSON); err != nil {
		return types.StatsJSON{}, fmt.Errorf("failed to decode stats: %w", err)
	}

	return statsJSON, nil
}

// ListImages lists Docker images
func (m *DockerManager) ListImages(ctx context.Context, all bool) ([]image.Summary, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	images, err := m.client.ImageList(ctx, image.ListOptions{All: all})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return images, nil
}

// GetImage gets image details
func (m *DockerManager) GetImage(ctx context.Context, imageID string) (*types.ImageInspect, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	imageInspect, _, err := m.client.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	return &imageInspect, nil
}

// PullImage pulls a Docker image
func (m *DockerManager) PullImage(ctx context.Context, imageName string) (io.ReadCloser, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	reader, err := m.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	return reader, nil
}

// RemoveImage removes a Docker image
func (m *DockerManager) RemoveImage(ctx context.Context, imageID string, force bool, pruneChildren bool) ([]image.DeleteResponse, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	options := image.RemoveOptions{
		Force:         force,
		PruneChildren: pruneChildren,
	}

	responses, err := m.client.ImageRemove(ctx, imageID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to remove image: %w", err)
	}

	return responses, nil
}

// PruneImages prunes unused images
func (m *DockerManager) PruneImages(ctx context.Context, pruneAll bool) (*image.PruneReport, error) {
	if m.client == nil {
		return nil, ErrNotConnected
	}

	pruneFilters := filters.NewArgs()
	if !pruneAll {
		pruneFilters.Add("dangling", "true")
	}

	report, err := m.client.ImagesPrune(ctx, pruneFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to prune images: %w", err)
	}

	return &report, nil
}

// ExecContainer executes a command in a running container
func (m *DockerManager) ExecContainer(ctx context.Context, containerID string, cmd []string) (string, error) {
	if m.client == nil {
		return "", ErrNotConnected
	}

	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	}

	execIDResp, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := m.client.ContainerExecAttach(ctx, execIDResp.ID, container.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach exec: %w", err)
	}
	defer resp.Close()

	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	return string(output), nil
}
