package main

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/cmd/tiga-agent/pty"
	"github.com/ysicing/tiga/proto"
)

// WindowSize represents terminal window dimensions
type WindowSize struct {
	Cols uint32 `json:"cols"`
	Rows uint32 `json:"rows"`
}

// handleTerminalSession handles a PTY terminal session
func handleTerminalSession(client proto.HostMonitorClient, streamID string) {
	logrus.Debugf("[Terminal:%s] Starting terminal session handler", streamID)
	ctx := context.Background()

	// Create IOStream connection
	stream, err := client.IOStream(ctx)
	if err != nil {
		logrus.Errorf("[Terminal:%s] Failed to create IOStream: %v", streamID, err)
		return
	}
	logrus.Debugf("[Terminal:%s] IOStream connection established", streamID)

	// Send StreamID with magic header (0xff 0x05 0xff 0x05)
	magicHeader := []byte{0xff, 0x05, 0xff, 0x05}
	streamIDBytes := append(magicHeader, []byte(streamID)...)
	if err := stream.Send(&proto.IOStreamData{Data: streamIDBytes}); err != nil {
		logrus.Errorf("[Terminal:%s] Failed to send StreamID: %v", streamID, err)
		return
	}
	logrus.Debugf("[Terminal:%s] Sent StreamID to server", streamID)

	// Start PTY
	tty, err := pty.Start()
	if err != nil {
		logrus.Errorf("[Terminal:%s] Failed to start PTY: %v", streamID, err)
		stream.Send(&proto.IOStreamData{Data: []byte(err.Error())})
		stream.CloseSend()
		return
	}
	logrus.Debugf("[Terminal:%s] PTY started successfully", streamID)

	defer func() {
		if err := tty.Close(); err != nil {
			logrus.Errorf("[Terminal:%s] Error closing PTY: %v", streamID, err)
		}
		if err := stream.CloseSend(); err != nil {
			logrus.Errorf("[Terminal:%s] Error closing stream: %v", streamID, err)
		}
		logrus.Infof("[Terminal:%s] Terminal session closed", streamID)
	}()

	logrus.Infof("[Terminal:%s] Terminal session started", streamID)

	// Create context for cancellation
	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start keepalive goroutine
	go terminalKeepAlive(sessionCtx, stream)

	// Read from PTY and send to server
	go func() {
		buf := make([]byte, 10240)
		for {
			n, err := tty.Read(buf)
			if err != nil {
				if err != io.EOF {
					logrus.Debugf("[Terminal:%s] PTY read error: %v", streamID, err)
					stream.Send(&proto.IOStreamData{Data: []byte(err.Error())})
				}
				stream.CloseSend()
				cancel()
				return
			}

			logrus.Debugf("[Terminal:%s] Read %d bytes from PTY", streamID, n)
			if err := stream.Send(&proto.IOStreamData{Data: buf[:n]}); err != nil {
				logrus.Errorf("[Terminal:%s] Failed to send PTY output: %v", streamID, err)
				cancel()
				return
			}
		}
	}()

	// Receive from server and write to PTY
	for {
		data, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				logrus.Errorf("[Terminal:%s] Stream receive error: %v", streamID, err)
			}
			return
		}

		if len(data.Data) == 0 {
			logrus.Debugf("[Terminal:%s] Received keepalive message", streamID)
			continue
		}

		logrus.Debugf("[Terminal:%s] Received %d bytes from server, command type: 0x%02x", streamID, len(data.Data), data.Data[0])

		// Parse command based on first byte
		switch data.Data[0] {
		case 0x00: // Terminal input
			if len(data.Data) > 1 {
				logrus.Debugf("[Terminal:%s] Writing %d bytes to PTY (input)", streamID, len(data.Data)-1)
				if _, err := tty.Write(data.Data[1:]); err != nil {
					logrus.Errorf("[Terminal:%s] Failed to write to PTY: %v", streamID, err)
					return
				}
			}

		case 0x01: // Window resize
			if len(data.Data) > 1 {
				var resize WindowSize
				decoder := json.NewDecoder(strings.NewReader(string(data.Data[1:])))
				if err := decoder.Decode(&resize); err != nil {
					logrus.Errorf("[Terminal:%s] Failed to decode resize message: %v", streamID, err)
					continue
				}

				logrus.Debugf("[Terminal:%s] Resizing PTY to %dx%d", streamID, resize.Cols, resize.Rows)
				if err := tty.Setsize(resize.Cols, resize.Rows); err != nil {
					logrus.Errorf("[Terminal:%s] Failed to resize PTY: %v", streamID, err)
				}
			}

		default:
			// Unknown command, ignore
			logrus.Warnf("[Terminal:%s] Unknown command byte: 0x%02x", streamID, data.Data[0])
		}
	}
}

// terminalKeepAlive sends periodic keepalive messages
func terminalKeepAlive(ctx context.Context, stream proto.HostMonitor_IOStreamClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	keepaliveMsg := &proto.IOStreamData{Data: []byte{}}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := stream.Send(keepaliveMsg); err != nil {
				return
			}
		}
	}
}
