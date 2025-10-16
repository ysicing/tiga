package testcontainers

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go/wait"

	tc "github.com/testcontainers/testcontainers-go"
)

type MinioContainer struct {
	Container tc.Container
	Host      string
	Port      int
	AccessKey string
	SecretKey string
}

func (m *MinioContainer) Endpoint() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

func (m *MinioContainer) Terminate(ctx context.Context) error {
	if m.Container != nil {
		return m.Container.Terminate(ctx)
	}
	return nil
}

// StartMinioContainer launches a MinIO server using testcontainers.
func StartMinioContainer(ctx context.Context) (*MinioContainer, error) {
	req := tc.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:             []string{"server", "/data"},
		WaitingFor:      wait.ForHTTP("/minio/health/live").WithPort("9000/tcp").WithStartupTimeout(2 * time.Minute),
		AutoRemove:      true,
		AlwaysPullImage: false,
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	mapped, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	return &MinioContainer{
		Container: container,
		Host:      host,
		Port:      mapped.Int(),
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
	}, nil
}
