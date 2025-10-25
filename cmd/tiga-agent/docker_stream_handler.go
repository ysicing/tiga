package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/docker"
	"github.com/ysicing/tiga/proto"
)

// DockerStreamHandler handles Docker streaming operations
type DockerStreamHandler struct {
	dockerClient *docker.DockerClient
	sessions     sync.Map // session_id -> active stream context
}

// NewDockerStreamHandler creates a new Docker stream handler
func NewDockerStreamHandler(dockerClient *docker.DockerClient) *DockerStreamHandler {
	return &DockerStreamHandler{
		dockerClient: dockerClient,
	}
}

// HandleDockerStream processes Docker stream requests from server
func (h *DockerStreamHandler) HandleDockerStream(stream proto.HostMonitor_DockerStreamClient) error {
	logrus.Info("[DockerStream] Stream connection established")

	// Goroutine to receive messages from server
	errCh := make(chan error, 1)
	go func() {
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				logrus.Info("[DockerStream] Server closed stream")
				errCh <- nil
				return
			}
			if err != nil {
				logrus.WithError(err).Error("[DockerStream] Failed to receive message")
				errCh <- err
				return
			}

			// Handle message
			if err := h.handleMessage(stream, msg); err != nil {
				logrus.WithError(err).Error("[DockerStream] Failed to handle message")
				// Send error back to server
				h.sendError(stream, "", err.Error())
			}
		}
	}()

	// Wait for completion or error
	return <-errCh
}

// handleMessage handles a single Docker stream message
func (h *DockerStreamHandler) handleMessage(stream proto.HostMonitor_DockerStreamClient, msg *proto.DockerStreamMessage) error {
	switch m := msg.Message.(type) {
	case *proto.DockerStreamMessage_Init:
		return h.handleInit(stream, m.Init)
	case *proto.DockerStreamMessage_Data:
		return h.handleData(stream, m.Data)
	case *proto.DockerStreamMessage_Resize:
		return h.handleResize(stream, m.Resize)
	case *proto.DockerStreamMessage_Close:
		return h.handleClose(stream, m.Close)
	default:
		return fmt.Errorf("unknown message type: %T", m)
	}
}

// handleInit handles stream initialization from server
func (h *DockerStreamHandler) handleInit(stream proto.HostMonitor_DockerStreamClient, init *proto.DockerStreamInit) error {
	sessionID := init.SessionId
	operation := init.Operation

	logrus.WithFields(logrus.Fields{
		"session_id":   sessionID,
		"operation":    operation,
		"container_id": init.ContainerId,
		"image_name":   init.ImageName,
	}).Info("[DockerStream] Initializing stream operation")

	// Create context for this session
	ctx, cancel := context.WithCancel(context.Background())
	h.sessions.Store(sessionID, cancel)

	// Send ready signal
	if err := stream.Send(&proto.DockerStreamMessage{
		Message: &proto.DockerStreamMessage_Init{
			Init: &proto.DockerStreamInit{
				SessionId: sessionID,
				Ready:     true,
			},
		},
	}); err != nil {
		cancel()
		return fmt.Errorf("failed to send ready signal: %w", err)
	}

	// Execute operation based on type
	switch operation {
	case "exec_container":
		go h.handleExecContainer(ctx, stream, init)
	case "get_logs":
		go h.handleGetLogs(ctx, stream, init)
	case "get_stats":
		go h.handleGetStats(ctx, stream, init)
	case "pull_image":
		go h.handlePullImage(ctx, stream, init)
	case "get_events":
		go h.handleGetEvents(ctx, stream, init)
	default:
		cancel()
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return nil
}

// handleData handles data from server (e.g., terminal input)
func (h *DockerStreamHandler) handleData(stream proto.HostMonitor_DockerStreamClient, data *proto.DockerStreamData) error {
	// This is typically used for terminal input in exec_container
	// The actual implementation will forward data to the running container exec session
	logrus.Debugf("[DockerStream] Received data: session=%s type=%s len=%d", data.SessionId, data.DataType, len(data.Data))
	return nil
}

// handleResize handles terminal resize requests
func (h *DockerStreamHandler) handleResize(stream proto.HostMonitor_DockerStreamClient, resize *proto.DockerStreamResize) error {
	logrus.Infof("[DockerStream] Terminal resize: session=%s width=%d height=%d", resize.SessionId, resize.Width, resize.Height)
	// TODO: Implement terminal resize for exec sessions
	return nil
}

// handleClose handles stream close requests
func (h *DockerStreamHandler) handleClose(stream proto.HostMonitor_DockerStreamClient, close *proto.DockerStreamClose) error {
	sessionID := close.SessionId
	logrus.WithFields(logrus.Fields{
		"session_id": sessionID,
		"reason":     close.Reason,
	}).Info("[DockerStream] Closing session")

	// Cancel session context
	if cancel, ok := h.sessions.LoadAndDelete(sessionID); ok {
		if cancelFunc, ok := cancel.(context.CancelFunc); ok {
			cancelFunc()
		}
	}

	return nil
}

// handleExecContainer handles container exec operation
func (h *DockerStreamHandler) handleExecContainer(ctx context.Context, stream proto.HostMonitor_DockerStreamClient, init *proto.DockerStreamInit) {
	// TODO: Implement exec container with bidirectional I/O
	logrus.Info("[DockerStream] exec_container not yet implemented")
	h.sendError(stream, init.SessionId, "exec_container not yet implemented")
}

// handleGetLogs handles container logs streaming
func (h *DockerStreamHandler) handleGetLogs(ctx context.Context, stream proto.HostMonitor_DockerStreamClient, init *proto.DockerStreamInit) {
	sessionID := init.SessionId
	containerID := init.ContainerId

	logrus.WithFields(logrus.Fields{
		"session_id":   sessionID,
		"container_id": containerID,
	}).Info("[DockerStream] Starting container logs stream")

	// Parse parameters
	follow := init.Params["follow"] == "true"
	tail := init.Params["tail"]
	if tail == "" {
		tail = "100"
	}

	// Build Docker log options
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
		Timestamps: init.Params["timestamps"] == "true",
	}

	// Get logs from Docker
	logsReader, err := h.dockerClient.Client().ContainerLogs(ctx, containerID, options)
	if err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to get container logs")
		h.sendError(stream, sessionID, err.Error())
		return
	}
	defer logsReader.Close()

	// Stream logs to server
	scanner := bufio.NewScanner(logsReader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			logrus.Info("[DockerStream] Logs stream context cancelled")
			return
		default:
		}

		line := scanner.Text()

		// Send log data to server
		if err := stream.Send(&proto.DockerStreamMessage{
			Message: &proto.DockerStreamMessage_Data{
				Data: &proto.DockerStreamData{
					SessionId: sessionID,
					Data:      []byte(line + "\n"),
					DataType:  "stdout", // Docker API returns combined stdout/stderr
				},
			},
		}); err != nil {
			logrus.WithError(err).Error("[DockerStream] Failed to send log data")
			return
		}
	}

	if err := scanner.Err(); err != nil {
		logrus.WithError(err).Error("[DockerStream] Error reading logs")
		h.sendError(stream, sessionID, err.Error())
		return
	}

	logrus.Info("[DockerStream] Logs stream completed")
	h.sendClose(stream, sessionID, "completed")
}

// handleGetStats handles container stats streaming
func (h *DockerStreamHandler) handleGetStats(ctx context.Context, stream proto.HostMonitor_DockerStreamClient, init *proto.DockerStreamInit) {
	sessionID := init.SessionId
	containerID := init.ContainerId

	logrus.WithFields(logrus.Fields{
		"session_id":   sessionID,
		"container_id": containerID,
	}).Info("[DockerStream] Starting container stats stream")

	// Get stats stream from Docker
	statsReader, err := h.dockerClient.Client().ContainerStats(ctx, containerID, true) // stream=true
	if err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to get container stats")
		h.sendError(stream, sessionID, err.Error())
		return
	}
	defer statsReader.Body.Close()

	// Decode and stream stats to server
	decoder := json.NewDecoder(statsReader.Body)
	for {
		select {
		case <-ctx.Done():
			logrus.Info("[DockerStream] Stats stream context cancelled")
			return
		default:
		}

		var stats types.StatsJSON
		if err := decoder.Decode(&stats); err != nil {
			if err == io.EOF {
				logrus.Info("[DockerStream] Stats stream completed")
				h.sendClose(stream, sessionID, "completed")
				return
			}
			logrus.WithError(err).Error("[DockerStream] Failed to decode stats")
			h.sendError(stream, sessionID, err.Error())
			return
		}

		// Marshal stats back to JSON for transmission
		statsJSON, err := json.Marshal(stats)
		if err != nil {
			logrus.WithError(err).Error("[DockerStream] Failed to marshal stats")
			h.sendError(stream, sessionID, err.Error())
			return
		}

		// Send stats data to server
		if err := stream.Send(&proto.DockerStreamMessage{
			Message: &proto.DockerStreamMessage_Data{
				Data: &proto.DockerStreamData{
					SessionId: sessionID,
					Data:      statsJSON,
					DataType:  "stats",
				},
			},
		}); err != nil {
			logrus.WithError(err).Error("[DockerStream] Failed to send stats data")
			return
		}
	}
}

// handlePullImage handles image pull with progress streaming
func (h *DockerStreamHandler) handlePullImage(ctx context.Context, stream proto.HostMonitor_DockerStreamClient, init *proto.DockerStreamInit) {
	// TODO: Implement image pull with progress
	logrus.Info("[DockerStream] pull_image not yet implemented")
	h.sendError(stream, init.SessionId, "pull_image not yet implemented")
}

// handleGetEvents handles Docker events streaming
func (h *DockerStreamHandler) handleGetEvents(ctx context.Context, stream proto.HostMonitor_DockerStreamClient, init *proto.DockerStreamInit) {
	// TODO: Implement Docker events streaming
	logrus.Info("[DockerStream] get_events not yet implemented")
	h.sendError(stream, init.SessionId, "get_events not yet implemented")
}

// sendError sends error message to server
func (h *DockerStreamHandler) sendError(stream proto.HostMonitor_DockerStreamClient, sessionID, errMsg string) {
	if err := stream.Send(&proto.DockerStreamMessage{
		Message: &proto.DockerStreamMessage_Error{
			Error: &proto.DockerStreamError{
				SessionId: sessionID,
				Error:     errMsg,
			},
		},
	}); err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to send error message")
	}
}

// sendClose sends close message to server
func (h *DockerStreamHandler) sendClose(stream proto.HostMonitor_DockerStreamClient, sessionID, reason string) {
	if err := stream.Send(&proto.DockerStreamMessage{
		Message: &proto.DockerStreamMessage_Close{
			Close: &proto.DockerStreamClose{
				SessionId: sessionID,
				Reason:    reason,
			},
		},
	}); err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to send close message")
	}
}
