package docker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ExecContainer implements the ExecContainer bidirectional streaming RPC method
// It manages a Docker exec session with stdin/stdout/stderr forwarding and TTY resizing
func (s *DockerService) ExecContainer(stream pb.DockerService_ExecContainerServer) error {
	ctx := stream.Context()

	// Wait for initial ExecStart message
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	// Ensure first message is ExecStart
	startMsg := req.GetStart()
	if startMsg == nil {
		return fmt.Errorf("first message must be ExecStart")
	}

	logrus.WithFields(logrus.Fields{
		"container_id": startMsg.ContainerId,
		"cmd":          startMsg.Cmd,
		"tty":          startMsg.Tty,
	}).Info("Starting exec session")

	// Create exec instance
	execConfig := types.ExecConfig{
		AttachStdin:  startMsg.AttachStdin,
		AttachStdout: startMsg.AttachStdout,
		AttachStderr: startMsg.AttachStderr,
		Tty:          startMsg.Tty,
		Cmd:          startMsg.Cmd,
		Env:          convertEnvMap(startMsg.Env),
	}

	execID, err := s.dockerClient.Client().ContainerExecCreate(ctx, startMsg.ContainerId, execConfig)
	if err != nil {
		logrus.WithError(err).Error("Failed to create exec instance")
		sendError := stream.Send(&pb.ExecResponse{
			Response: &pb.ExecResponse_Error{
				Error: &pb.ExecError{Message: err.Error()},
			},
		})
		if sendError != nil {
			return sendError
		}
		return err
	}

	logrus.WithField("exec_id", execID.ID).Info("Exec instance created")

	// Attach to exec instance
	attachResp, err := s.dockerClient.Client().ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{
		Tty: startMsg.Tty,
	})
	if err != nil {
		logrus.WithError(err).Error("Failed to attach to exec instance")
		sendError := stream.Send(&pb.ExecResponse{
			Response: &pb.ExecResponse_Error{
				Error: &pb.ExecError{Message: err.Error()},
			},
		})
		if sendError != nil {
			return sendError
		}
		return err
	}
	defer attachResp.Close()

	logrus.Info("Attached to exec instance")

	// Use WaitGroup to manage concurrent goroutines
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Goroutine 1: Read output from Docker exec and send to client
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.forwardExecOutput(stream, &attachResp, startMsg.Tty); err != nil {
			logrus.WithError(err).Error("Error forwarding exec output")
			errChan <- err
		}
	}()

	// Goroutine 2: Read input from client and write to Docker exec stdin
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer attachResp.CloseWrite() // Signal EOF to exec when client stops sending

		for {
			req, err := stream.Recv()
			if err == io.EOF {
				logrus.Info("Client closed input stream")
				return
			}
			if err != nil {
				logrus.WithError(err).Error("Error receiving from client")
				errChan <- err
				return
			}

			// Handle different request types
			if inputMsg := req.GetInput(); inputMsg != nil {
				// Forward stdin data
				if _, writeErr := attachResp.Conn.Write(inputMsg.Data); writeErr != nil {
					logrus.WithError(writeErr).Error("Error writing to exec stdin")
					errChan <- writeErr
					return
				}
			} else if resizeMsg := req.GetResize(); resizeMsg != nil {
				// Resize TTY
				if resizeErr := s.dockerClient.Client().ContainerExecResize(ctx, execID.ID, container.ResizeOptions{
					Height: uint(resizeMsg.Height),
					Width:  uint(resizeMsg.Width),
				}); resizeErr != nil {
					logrus.WithError(resizeErr).Warn("Failed to resize TTY")
					// Don't fail the entire exec session for resize errors
				} else {
					logrus.WithFields(logrus.Fields{
						"width":  resizeMsg.Width,
						"height": resizeMsg.Height,
					}).Debug("TTY resized")
				}
			}
		}
	}()

	// Goroutine 3: Poll exec status and send exit code when finished
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.waitForExecExit(ctx, stream, execID.ID); err != nil {
			logrus.WithError(err).Error("Error waiting for exec exit")
			errChan <- err
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Check if any goroutine encountered an error
	for err := range errChan {
		if err != nil && err != io.EOF {
			return err
		}
	}

	logrus.Info("Exec session completed")
	return nil
}

// forwardExecOutput reads stdout/stderr from Docker exec and sends to gRPC client
func (s *DockerService) forwardExecOutput(stream pb.DockerService_ExecContainerServer, attachResp *types.HijackedResponse, tty bool) error {
	if tty {
		// TTY mode: no stream multiplexing, raw output
		buf := make([]byte, 8192)
		for {
			n, err := attachResp.Reader.Read(buf)
			if n > 0 {
				// Send output to client
				if sendErr := stream.Send(&pb.ExecResponse{
					Response: &pb.ExecResponse_Output{
						Output: &pb.ExecOutput{
							Data:   buf[:n],
							Stream: "stdout", // TTY mode doesn't distinguish streams
						},
					},
				}); sendErr != nil {
					return sendErr
				}
			}
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
		}
	} else {
		// Non-TTY mode: use Docker's stream multiplexing
		// stdcopy.StdCopy demultiplexes stdout and stderr
		stdout := &execStreamWriter{stream: stream, streamType: "stdout"}
		stderr := &execStreamWriter{stream: stream, streamType: "stderr"}

		_, err := stdcopy.StdCopy(stdout, stderr, attachResp.Reader)
		return err
	}
}

// waitForExecExit polls Docker exec status and sends exit code when finished
func (s *DockerService) waitForExecExit(ctx context.Context, stream pb.DockerService_ExecContainerServer, execID string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Inspect exec to check if it's finished
			inspectResp, err := s.dockerClient.Client().ContainerExecInspect(ctx, execID)
			if err != nil {
				return err
			}

			if !inspectResp.Running {
				// Exec has finished, send exit code
				logrus.WithField("exit_code", inspectResp.ExitCode).Info("Exec finished")
				return stream.Send(&pb.ExecResponse{
					Response: &pb.ExecResponse_Exit{
						Exit: &pb.ExecExit{
							ExitCode: int32(inspectResp.ExitCode),
						},
					},
				})
			}
		}
	}
}

// execStreamWriter implements io.Writer to forward Docker exec output to gRPC stream
type execStreamWriter struct {
	stream     pb.DockerService_ExecContainerServer
	streamType string // "stdout" or "stderr"
}

func (w *execStreamWriter) Write(p []byte) (int, error) {
	err := w.stream.Send(&pb.ExecResponse{
		Response: &pb.ExecResponse_Output{
			Output: &pb.ExecOutput{
				Data:   p,
				Stream: w.streamType,
			},
		},
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// convertEnvMap converts protobuf env map to Docker env slice
func convertEnvMap(envMap map[string]string) []string {
	if len(envMap) == 0 {
		return nil
	}

	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}
