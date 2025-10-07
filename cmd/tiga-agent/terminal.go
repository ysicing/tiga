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
	ctx := context.Background()

	// Create IOStream connection
	stream, err := client.IOStream(ctx)
	if err != nil {
		logrus.Errorf("Failed to create IOStream: %v", err)
		return
	}

	// Send StreamID with magic header (0xff 0x05 0xff 0x05)
	magicHeader := []byte{0xff, 0x05, 0xff, 0x05}
	streamIDBytes := append(magicHeader, []byte(streamID)...)
	if err := stream.Send(&proto.IOStreamData{Data: streamIDBytes}); err != nil {
		logrus.Errorf("Failed to send StreamID: %v", err)
		return
	}

	// Start PTY
	tty, err := pty.Start()
	if err != nil {
		logrus.Errorf("Failed to start PTY: %v", err)
		stream.Send(&proto.IOStreamData{Data: []byte(err.Error())})
		stream.CloseSend()
		return
	}

	defer func() {
		if err := tty.Close(); err != nil {
			logrus.Errorf("Error closing PTY: %v", err)
		}
		if err := stream.CloseSend(); err != nil {
			logrus.Errorf("Error closing stream: %v", err)
		}
		logrus.Infof("Terminal session closed: %s", streamID)
	}()

	logrus.Infof("Terminal session started: %s", streamID)

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
					stream.Send(&proto.IOStreamData{Data: []byte(err.Error())})
				}
				stream.CloseSend()
				cancel()
				return
			}

			if err := stream.Send(&proto.IOStreamData{Data: buf[:n]}); err != nil {
				logrus.Errorf("Failed to send PTY output: %v", err)
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
				logrus.Errorf("Stream receive error: %v", err)
			}
			return
		}

		if len(data.Data) == 0 {
			continue
		}

		// Parse command based on first byte
		switch data.Data[0] {
		case 0x00: // Terminal input
			if len(data.Data) > 1 {
				if _, err := tty.Write(data.Data[1:]); err != nil {
					logrus.Errorf("Failed to write to PTY: %v", err)
					return
				}
			}

		case 0x01: // Window resize
			if len(data.Data) > 1 {
				var resize WindowSize
				decoder := json.NewDecoder(strings.NewReader(string(data.Data[1:])))
				if err := decoder.Decode(&resize); err != nil {
					logrus.Errorf("Failed to decode resize message: %v", err)
					continue
				}

				if err := tty.Setsize(resize.Cols, resize.Rows); err != nil {
					logrus.Errorf("Failed to resize PTY: %v", err)
				}
			}

		default:
			// Unknown command, ignore
			logrus.Warnf("Unknown command byte: 0x%02x", data.Data[0])
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
