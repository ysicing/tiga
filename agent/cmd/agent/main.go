package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/agent/internal/docker"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	defaultPort = "50051"
)

func main() {
	// Configure logging
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetLevel(logrus.InfoLevel)

	// Get port from environment or use default
	port := os.Getenv("AGENT_PORT")
	if port == "" {
		port = defaultPort
	}

	logrus.WithField("port", port).Info("Starting Tiga Docker Agent")

	// Initialize Docker client
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize Docker client")
	}
	defer dockerClient.Close()

	// Test Docker connection
	logrus.Info("Testing Docker connection...")
	if err := dockerClient.Ping(nil); err != nil {
		logrus.WithError(err).Fatal("Failed to connect to Docker daemon")
	}
	logrus.WithFields(logrus.Fields{
		"docker_version": dockerClient.DockerVersion(),
		"api_version":    dockerClient.APIVersion(),
	}).Info("Docker connection successful")

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB max receive
		grpc.MaxSendMsgSize(10 * 1024 * 1024), // 10MB max send
	)

	// Register Docker service
	dockerService := docker.NewDockerService(dockerClient)
	pb.RegisterDockerServiceServer(grpcServer, dockerService)

	// Register reflection service (for grpcurl and debugging)
	reflection.Register(grpcServer)

	logrus.Info("Docker service registered successfully")

	// Create TCP listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create listener")
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logrus.Info("Shutting down gracefully...")
		grpcServer.GracefulStop()
		logrus.Info("Server stopped")
	}()

	// Start serving
	logrus.WithField("address", fmt.Sprintf("0.0.0.0:%s", port)).Info("Agent gRPC server started")
	if err := grpcServer.Serve(listener); err != nil {
		logrus.WithError(err).Fatal("Failed to serve gRPC")
	}
}
