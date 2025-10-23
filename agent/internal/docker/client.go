package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// DockerClient wraps the Docker SDK client with additional metadata
type DockerClient struct {
	client     *client.Client
	apiVersion string
	dockerVersion string
}

// NewDockerClient creates a new Docker client with API version negotiation and validation
func NewDockerClient() (*DockerClient, error) {
	// Create client with automatic API version negotiation
	cli, err := client.NewClientWithOpts(
		client.FromEnv,                      // Read from DOCKER_HOST, DOCKER_TLS_VERIFY, DOCKER_CERT_PATH
		client.WithAPIVersionNegotiation(),  // Automatically negotiate API version
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Get Docker server version information
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := cli.ServerVersion(ctx)
	if err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to get Docker server version: %w", err)
	}

	// Validate minimum API version (1.41 = Docker 20.10+)
	if !isVersionSupported(info.APIVersion) {
		cli.Close()
		return nil, fmt.Errorf(
			"Docker API version %s is not supported, minimum required: 1.41 (Docker 20.10+). "+
				"Please upgrade your Docker installation. See: https://docs.docker.com/engine/install/",
			info.APIVersion,
		)
	}

	logrus.WithFields(logrus.Fields{
		"docker_version": info.Version,
		"api_version":    info.APIVersion,
		"os":             info.Os,
		"arch":           info.Arch,
		"kernel_version": info.KernelVersion,
		"go_version":     info.GoVersion,
	}).Info("Docker client initialized successfully")

	return &DockerClient{
		client:        cli,
		apiVersion:    info.APIVersion,
		dockerVersion: info.Version,
	}, nil
}

// isVersionSupported checks if the Docker API version meets minimum requirements
// Minimum version: 1.41 (Docker 20.10+)
func isVersionSupported(apiVersion string) bool {
	// API version format: "1.41", "1.43", etc.
	parts := strings.Split(apiVersion, ".")
	if len(parts) != 2 {
		logrus.WithField("api_version", apiVersion).Warn("Invalid API version format")
		return false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		logrus.WithError(err).WithField("api_version", apiVersion).Warn("Failed to parse major version")
		return false
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		logrus.WithError(err).WithField("api_version", apiVersion).Warn("Failed to parse minor version")
		return false
	}

	// Check: major > 1 OR (major == 1 AND minor >= 41)
	if major > 1 {
		return true
	}
	if major == 1 && minor >= 41 {
		return true
	}

	logrus.WithFields(logrus.Fields{
		"api_version":      apiVersion,
		"required_version": "1.41+",
	}).Warn("Docker API version is below minimum required version")

	return false
}

// Client returns the underlying Docker SDK client
func (dc *DockerClient) Client() *client.Client {
	return dc.client
}

// APIVersion returns the negotiated API version
func (dc *DockerClient) APIVersion() string {
	return dc.apiVersion
}

// DockerVersion returns the Docker daemon version
func (dc *DockerClient) DockerVersion() string {
	return dc.dockerVersion
}

// Close closes the Docker client connection
func (dc *DockerClient) Close() error {
	if dc.client != nil {
		return dc.client.Close()
	}
	return nil
}

// Ping tests the connection to Docker daemon
func (dc *DockerClient) Ping(ctx context.Context) error {
	_, err := dc.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping Docker daemon: %w", err)
	}
	return nil
}
