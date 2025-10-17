package k8s

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/prometheus"
	"github.com/ysicing/tiga/pkg/kube"
)

// ClusterHealthService manages cluster health checks (Phase 0, enhanced in Phase 1)
type ClusterHealthService struct {
	clusterRepo      repository.ClusterRepositoryInterface
	clientCache      *kube.ClientCache
	prometheusDiscov *prometheus.AutoDiscoveryService
	interval         time.Duration
	stopCh           chan struct{}
}

// NewClusterHealthService creates a new ClusterHealthService instance
func NewClusterHealthService(
	clusterRepo repository.ClusterRepositoryInterface,
	clientCache *kube.ClientCache,
	prometheusDiscov *prometheus.AutoDiscoveryService,
) *ClusterHealthService {
	return &ClusterHealthService{
		clusterRepo:      clusterRepo,
		clientCache:      clientCache,
		prometheusDiscov: prometheusDiscov,
		interval:         60 * time.Second, // Check every 60 seconds
		stopCh:           make(chan struct{}),
	}
}

// Start begins the health check loop in a background goroutine
func (s *ClusterHealthService) Start(ctx context.Context) {
	logrus.Info("Starting cluster health check service (60s interval)")

	// Run initial health check immediately
	go s.checkAllClusters(ctx)

	// Start periodic health checks
	ticker := time.NewTicker(s.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.checkAllClusters(ctx)
			case <-s.stopCh:
				ticker.Stop()
				logrus.Info("Cluster health check service stopped")
				return
			case <-ctx.Done():
				ticker.Stop()
				logrus.Info("Cluster health check service stopped (context cancelled)")
				return
			}
		}
	}()
}

// Stop stops the health check service
func (s *ClusterHealthService) Stop() {
	close(s.stopCh)
}

// checkAllClusters performs health check on all enabled clusters
func (s *ClusterHealthService) checkAllClusters(ctx context.Context) {
	clusters, err := s.clusterRepo.GetAllEnabled(ctx)
	if err != nil {
		logrus.Errorf("Failed to fetch enabled clusters: %v", err)
		return
	}

	logrus.Debugf("Health check: found %d enabled clusters", len(clusters))

	for _, cluster := range clusters {
		s.checkClusterHealth(ctx, cluster)
	}
}

// checkClusterHealth checks the health of a single cluster
func (s *ClusterHealthService) checkClusterHealth(ctx context.Context, cluster *models.Cluster) {
	clusterID := cluster.ID
	clusterName := cluster.Name
	previousStatus := cluster.HealthStatus

	// Get or create K8s client
	client, err := s.getOrCreateClient(cluster)
	if err != nil {
		logrus.Warnf("Cluster %s: failed to create client: %v", clusterName, err)
		s.updateClusterStatus(ctx, clusterID, models.ClusterHealthUnavailable, 0, 0)
		return
	}

	// Check connectivity by calling ServerVersion
	version, err := client.ClientSet.Discovery().ServerVersion()
	if err != nil {
		logrus.Warnf("Cluster %s: health check failed: %v", clusterName, err)
		s.updateClusterStatus(ctx, clusterID, models.ClusterHealthError, 0, 0)
		return
	}

	logrus.Debugf("Cluster %s: connected, server version %s", clusterName, version.GitVersion)

	// Count nodes
	nodeList, err := client.ClientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.Warnf("Cluster %s: failed to list nodes: %v", clusterName, err)
		s.updateClusterStatus(ctx, clusterID, models.ClusterHealthWarning, 0, 0)
		return
	}
	nodeCount := len(nodeList.Items)

	// Count pods across all namespaces
	podList, err := client.ClientSet.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logrus.Warnf("Cluster %s: failed to list pods: %v", clusterName, err)
		s.updateClusterStatus(ctx, clusterID, models.ClusterHealthWarning, nodeCount, 0)
		return
	}
	podCount := len(podList.Items)

	// All checks passed - cluster is healthy
	s.updateClusterStatus(ctx, clusterID, models.ClusterHealthHealthy, nodeCount, podCount)
	logrus.Debugf("Cluster %s: healthy, %d nodes, %d pods", clusterName, nodeCount, podCount)

	// Phase 1 enhancement: Trigger Prometheus discovery on first successful connection
	if previousStatus != models.ClusterHealthHealthy && s.prometheusDiscov != nil {
		logrus.Infof("Cluster %s became healthy, triggering Prometheus discovery", clusterName)
		s.prometheusDiscov.TriggerDiscoveryAsync(cluster)
	}
}

// getOrCreateClient gets an existing client from cache or creates a new one
func (s *ClusterHealthService) getOrCreateClient(cluster *models.Cluster) (*kube.K8sClient, error) {
	// Check cache first
	if client, exists := s.clientCache.Get(cluster.ID); exists {
		return client, nil
	}

	// Create new client
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Config))
	if err != nil {
		return nil, err
	}

	client, err := kube.NewClient(config)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.clientCache.Set(cluster.ID, client)
	logrus.Infof("Created and cached K8s client for cluster %s", cluster.Name)

	return client, nil
}

// updateClusterStatus updates the cluster health status in database
func (s *ClusterHealthService) updateClusterStatus(
	ctx context.Context,
	clusterID uuid.UUID,
	healthStatus string,
	nodeCount int,
	podCount int,
) {
	now := time.Now()
	updates := map[string]interface{}{
		"health_status":     healthStatus,
		"node_count":        nodeCount,
		"pod_count":         podCount,
		"last_connected_at": now,
	}

	if err := s.clusterRepo.Update(ctx, clusterID, updates); err != nil {
		logrus.Errorf("Failed to update cluster %s status: %v", clusterID, err)
	}
}
