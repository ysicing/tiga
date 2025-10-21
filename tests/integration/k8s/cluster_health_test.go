package k8s_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/pkg/kube"

	k8sservice "github.com/ysicing/tiga/internal/services/k8s"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// TestClusterHealthCheck tests cluster health check service
// Verification: quickstart.md V1.2
func TestClusterHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Step 1: Start Kind (Kubernetes in Docker) cluster using testcontainers
	// Kind image includes a complete Kubernetes cluster
	req := testcontainers.ContainerRequest{
		Image:        "kindest/node:v1.27.3",
		ExposedPorts: []string{"6443/tcp"},
		WaitingFor: wait.ForLog("Reached target multi-user.target").
			WithStartupTimeout(120 * time.Second),
		Privileged: true, // Kind needs privileged mode
	}

	kindContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start Kind container")
	defer kindContainer.Terminate(ctx)

	// Get container host and port for Kubernetes API server
	host, err := kindContainer.Host(ctx)
	require.NoError(t, err)

	port, err := kindContainer.MappedPort(ctx, "6443")
	require.NoError(t, err)

	// Step 2: Create Kubeconfig for the Kind cluster
	kubeconfig := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"kind-test": {
				Server:                fmt.Sprintf("https://%s:%s", host, port.Port()),
				InsecureSkipTLSVerify: true, // For testing only
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"kind-test": {
				Cluster:  "kind-test",
				AuthInfo: "kind-admin",
			},
		},
		CurrentContext: "kind-test",
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"kind-admin": {
				// Kind uses client certificate authentication
				// In real scenario, we would extract cert from container
				// For simplicity, using insecure skip TLS verify
			},
		},
	}

	kubeconfigBytes, err := clientcmd.Write(kubeconfig)
	require.NoError(t, err)

	// Step 3: Setup test database (SQLite in-memory)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.Cluster{})
	require.NoError(t, err)

	// Step 4: Create repository and services
	clusterRepo := repository.NewClusterRepository(db)
	clientCache := kube.NewClientCache()
	// Note: PrometheusAutoDiscoveryService is optional for this test
	// We pass nil as we're only testing health check functionality
	healthService := k8sservice.NewClusterHealthService(clusterRepo, clientCache, nil)

	// Step 5: Create cluster record in database
	cluster := &models.Cluster{
		Name:        "test-kind-cluster",
		Config:      string(kubeconfigBytes),
		InCluster:   false,
		Enable:      true,
		IsDefault:   true,
		Description: "Test Kind cluster for integration testing",
	}

	err = clusterRepo.Create(ctx, cluster)
	require.NoError(t, err, "Failed to create cluster record")

	// Verify initial state
	assert.Equal(t, models.ClusterHealthUnknown, cluster.HealthStatus, "Initial health status should be unknown")
	assert.Equal(t, 0, cluster.NodeCount, "Initial node count should be 0")
	assert.Equal(t, 0, cluster.PodCount, "Initial pod count should be 0")

	// Step 6: Start health check service
	healthCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Run health check service (checks every 60 seconds)
	// Start method doesn't return errors, it runs in background
	go healthService.Start(healthCtx)

	// Step 7: Wait for health check to complete (first run)
	// The health service runs immediately on start, then every 60 seconds
	// We'll wait up to 90 seconds for the first check to complete
	time.Sleep(10 * time.Second) // Give it time to perform first check

	// Reload cluster from database
	updatedCluster, err := clusterRepo.GetByID(ctx, cluster.ID)
	require.NoError(t, err, "Failed to reload cluster")

	// Step 8: Verify health status changed from unknown to healthy
	t.Logf("Health status: %s, Node count: %d, Pod count: %d",
		updatedCluster.HealthStatus, updatedCluster.NodeCount, updatedCluster.PodCount)

	// Assertions (V1.2 verification)
	assert.Equal(t, models.ClusterHealthHealthy, updatedCluster.HealthStatus,
		"Health status should be 'healthy' after health check")

	assert.Greater(t, updatedCluster.NodeCount, 0,
		"Node count should be greater than 0 (Kind has at least 1 control-plane node)")

	assert.GreaterOrEqual(t, updatedCluster.PodCount, 0,
		"Pod count should be greater than or equal to 0")

	assert.NotNil(t, updatedCluster.LastConnectedAt,
		"LastConnectedAt should be set after successful health check")

	// Step 9: Verify last_connected_at timestamp is recent
	timeSinceLastConnected := time.Since(*updatedCluster.LastConnectedAt)
	assert.Less(t, timeSinceLastConnected, 30*time.Second,
		"LastConnectedAt should be within the last 30 seconds")

	// Step 10: Stop health check service
	cancel()
	time.Sleep(2 * time.Second) // Give service time to cleanup
}

// TestClusterHealthCheckStateTransitions tests health status state transitions
func TestClusterHealthCheckStateTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Setup test database (SQLite in-memory)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.Cluster{})
	require.NoError(t, err)

	// Create repository
	clusterRepo := repository.NewClusterRepository(db)

	// Test state transitions: unknown → healthy → warning → error → unavailable
	t.Run("StateTransition_Unknown_To_Healthy", func(t *testing.T) {
		cluster := &models.Cluster{
			Name:         "transition-test-cluster",
			HealthStatus: models.ClusterHealthUnknown,
			Enable:       true,
		}

		err := clusterRepo.Create(ctx, cluster)
		require.NoError(t, err)

		// Simulate health check success
		now := time.Now()
		updates := map[string]interface{}{
			"health_status":     models.ClusterHealthHealthy,
			"node_count":        3,
			"pod_count":         25,
			"last_connected_at": &now,
		}

		err = clusterRepo.Update(ctx, cluster.ID, updates)
		require.NoError(t, err)

		// Verify transition
		updated, err := clusterRepo.GetByID(ctx, cluster.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ClusterHealthHealthy, updated.HealthStatus)
	})

	t.Run("StateTransition_Healthy_To_Unavailable", func(t *testing.T) {
		cluster := &models.Cluster{
			Name:         "unavailable-test-cluster",
			HealthStatus: models.ClusterHealthHealthy,
			NodeCount:    3,
			Enable:       true,
		}

		err := clusterRepo.Create(ctx, cluster)
		require.NoError(t, err)

		// Simulate connection failure
		updates := map[string]interface{}{
			"health_status": models.ClusterHealthUnavailable,
			"node_count":    0,
			"pod_count":     0,
		}

		err = clusterRepo.Update(ctx, cluster.ID, updates)
		require.NoError(t, err)

		// Verify transition
		updated, err := clusterRepo.GetByID(ctx, cluster.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ClusterHealthUnavailable, updated.HealthStatus)
		assert.Equal(t, 0, updated.NodeCount)
	})
}
