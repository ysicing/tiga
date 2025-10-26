package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
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

	// Check if this is a Ready confirmation from Server
	if init.Ready {
		logrus.WithField("session_id", sessionID).Info("[DockerStream] Received Ready confirmation from server")
		return nil
	}

	// This is an operation initialization from Server
	operation := init.Operation

	logrus.WithFields(logrus.Fields{
		"session_id":   sessionID,
		"operation":    operation,
		"container_id": init.ContainerId,
		"image_name":   init.ImageName,
	}).Info("[DockerStream] Initializing stream operation")

	// Validate operation
	if operation == "" {
		return fmt.Errorf("missing operation in init message")
	}

	// Create context for this session
	ctx, cancel := context.WithCancel(context.Background())
	h.sessions.Store(sessionID, cancel)

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
	sessionID := init.SessionId
	containerID := init.ContainerId

	logrus.WithFields(logrus.Fields{
		"session_id":   sessionID,
		"container_id": containerID,
	}).Info("[DockerStream] Starting exec_container")

	// Extract shell and terminal size from params
	shell := init.Params["shell"]
	if shell == "" {
		shell = "/bin/sh"
	}

	// Parse terminal size (default to 30x120)
	rows := 30
	cols := 120
	if rowsStr := init.Params["rows"]; rowsStr != "" {
		fmt.Sscanf(rowsStr, "%d", &rows)
	}
	if colsStr := init.Params["cols"]; colsStr != "" {
		fmt.Sscanf(colsStr, "%d", &cols)
	}

	// Create exec instance
	execConfig := types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true, // Always use TTY for terminal
		Cmd:          []string{shell},
		Env:          []string{"TERM=xterm-256color"},
	}

	execID, err := h.dockerClient.Client().ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to create exec instance")
		h.sendError(stream, sessionID, fmt.Sprintf("Failed to create exec: %v", err))
		return
	}

	logrus.WithField("exec_id", execID.ID).Info("[DockerStream] Exec instance created")

	// Attach to exec instance
	attachResp, err := h.dockerClient.Client().ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{
		Tty: true,
	})
	if err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to attach to exec")
		h.sendError(stream, sessionID, fmt.Sprintf("Failed to attach: %v", err))
		return
	}
	defer attachResp.Close()

	// Resize TTY to initial size
	if err := h.dockerClient.Client().ContainerExecResize(ctx, execID.ID, container.ResizeOptions{
		Height: uint(rows),
		Width:  uint(cols),
	}); err != nil {
		logrus.WithError(err).Warn("[DockerStream] Failed to resize TTY initially")
	}

	logrus.Info("[DockerStream] Attached to exec instance, starting I/O forwarding")

	// Channel to coordinate shutdown
	done := make(chan bool, 1)

	// Goroutine: Read output from Docker exec and send to Server with buffering
	go func() {
		readBuf := make([]byte, 8192)
		sendBuf := make([]byte, 0, 16384)               // Accumulated buffer
		ticker := time.NewTicker(50 * time.Millisecond) // Flush every 50ms
		defer ticker.Stop()

		flushBuffer := func() {
			if len(sendBuf) > 0 {
				// Send accumulated data
				if err := stream.Send(&proto.DockerStreamMessage{
					Message: &proto.DockerStreamMessage_Data{
						Data: &proto.DockerStreamData{
							SessionId: sessionID,
							Data:      append([]byte(nil), sendBuf...), // Copy to avoid data race
							DataType:  "stdout",
						},
					},
				}); err != nil {
					logrus.WithError(err).Error("[DockerStream] Failed to send output")
					done <- true
					return
				}
				sendBuf = sendBuf[:0] // Clear buffer
			}
		}

		for {
			select {
			case <-ticker.C:
				// Periodic flush
				flushBuffer()

			case <-done:
				return

			default:
				// Try to read with timeout
				attachResp.Conn.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
				n, err := attachResp.Reader.Read(readBuf)

				if n > 0 {
					// Append to send buffer
					sendBuf = append(sendBuf, readBuf[:n]...)

					// Flush if buffer is getting large (> 4KB) for responsiveness
					if len(sendBuf) > 4096 {
						flushBuffer()
					}
				}

				if err != nil {
					if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
						// Read timeout, continue to next iteration
						continue
					}
					if err != io.EOF {
						logrus.WithError(err).Error("[DockerStream] Error reading from exec")
					}
					// Flush any remaining data before exiting
					flushBuffer()
					done <- true
					return
				}
			}
		}
	}()

	// Goroutine: Read messages from Server (stdin, resize)
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					logrus.WithError(err).Error("[DockerStream] Error receiving from server")
				}
				done <- true
				return
			}

			switch m := msg.Message.(type) {
			case *proto.DockerStreamMessage_Data:
				// Forward stdin to exec
				if m.Data.DataType == "stdin" {
					if _, err := attachResp.Conn.Write(m.Data.Data); err != nil {
						logrus.WithError(err).Error("[DockerStream] Error writing to stdin")
						done <- true
						return
					}
				}

			case *proto.DockerStreamMessage_Resize:
				// Resize TTY
				if err := h.dockerClient.Client().ContainerExecResize(ctx, execID.ID, container.ResizeOptions{
					Height: uint(m.Resize.Height),
					Width:  uint(m.Resize.Width),
				}); err != nil {
					logrus.WithError(err).Warn("[DockerStream] Failed to resize TTY")
				} else {
					logrus.WithFields(logrus.Fields{
						"width":  m.Resize.Width,
						"height": m.Resize.Height,
					}).Debug("[DockerStream] TTY resized")
				}

			case *proto.DockerStreamMessage_Close:
				// Client requested close
				logrus.Info("[DockerStream] Received close request from server")
				done <- true
				return
			}
		}
	}()

	// Goroutine: Poll exec status and send exit code
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				done <- true
				return
			case <-done:
				return
			case <-ticker.C:
				inspectResp, err := h.dockerClient.Client().ContainerExecInspect(ctx, execID.ID)
				if err != nil {
					logrus.WithError(err).Error("[DockerStream] Error inspecting exec")
					continue
				}

				if !inspectResp.Running {
					logrus.WithField("exit_code", inspectResp.ExitCode).Info("[DockerStream] Exec finished")
					// Send close message with exit code
					stream.Send(&proto.DockerStreamMessage{
						Message: &proto.DockerStreamMessage_Close{
							Close: &proto.DockerStreamClose{
								SessionId: sessionID,
								Reason:    fmt.Sprintf("Process exited with code %d", inspectResp.ExitCode),
							},
						},
					})
					done <- true
					return
				}
			}
		}
	}()

	// Wait for completion
	<-done
	logrus.Info("[DockerStream] Exec session completed")
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
	sessionID := init.SessionId
	imageName := init.ImageName

	logrus.WithFields(logrus.Fields{
		"session_id": sessionID,
		"image_name": imageName,
	}).Info("[DockerStream] Starting image pull stream")

	// Parse parameters
	registryAuth := init.Params["registry_auth"] // Base64 encoded auth

	// Build Docker pull options
	pullOptions := image.PullOptions{
		RegistryAuth: registryAuth,
	}

	logrus.Debugf("[DockerStream] Calling Docker ImagePull API for image: %s", imageName)

	// Pull image from Docker
	pullReader, err := h.dockerClient.Client().ImagePull(ctx, imageName, pullOptions)
	if err != nil {
		logrus.WithError(err).Error("[DockerStream] Failed to start image pull")
		h.sendError(stream, sessionID, err.Error())
		return
	}
	defer pullReader.Close()

	logrus.Debug("[DockerStream] Docker ImagePull API started successfully, reading pull stream...")

	// IMPORTANT: Must read the entire pull stream to ensure Docker saves the image
	// Even if sending to server fails, we must continue reading to complete the pull
	scanner := bufio.NewScanner(pullReader)
	streamBroken := false
	lineCount := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			logrus.Warn("[DockerStream] Pull stream context cancelled, but continuing to read pull stream to ensure image is saved")
			streamBroken = true
		default:
		}

		line := scanner.Bytes()
		lineCount++

		// Debug: Log first 5 and last few progress lines
		if lineCount <= 5 {
			logrus.Debugf("[DockerStream] Pull progress line %d: %s", lineCount, string(line))
		}

		// Try to send progress to server (but don't abort if it fails)
		if !streamBroken {
			if err := stream.Send(&proto.DockerStreamMessage{
				Message: &proto.DockerStreamMessage_Data{
					Data: &proto.DockerStreamData{
						SessionId: sessionID,
						Data:      line,
						DataType:  "progress",
					},
				},
			}); err != nil {
				logrus.WithError(err).Warn("[DockerStream] Failed to send pull progress (continuing to read pull stream)")
				streamBroken = true
			}
		}
		// Continue reading even if stream is broken to ensure Docker completes the pull
	}

	if err := scanner.Err(); err != nil {
		logrus.WithError(err).Error("[DockerStream] Error reading pull progress")
		if !streamBroken {
			h.sendError(stream, sessionID, err.Error())
		}
		return
	}

	logrus.Infof("[DockerStream] Image pull stream completed successfully, read %d progress lines", lineCount)

	// Verify the image exists locally after pull
	inspectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err = h.dockerClient.Client().ImageInspectWithRaw(inspectCtx, imageName)
	if err != nil {
		logrus.WithError(err).Errorf("[DockerStream] WARNING: Image pull stream completed but image not found locally: %s", imageName)
		if !streamBroken {
			h.sendError(stream, sessionID, fmt.Sprintf("image pulled but not found locally: %v", err))
		}
		return
	}

	logrus.Infof("[DockerStream] ✓ Verified: Image %s exists locally after pull", imageName)

	if !streamBroken {
		h.sendClose(stream, sessionID, "completed")
	} else {
		logrus.Info("[DockerStream] Image pulled successfully but stream was broken, not sending close message")
	}
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
