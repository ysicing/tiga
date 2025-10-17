package prometheus

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/pkg/kube"
)

// AutoDiscoveryService manages automatic Prometheus discovery for clusters
type AutoDiscoveryService struct {
	clusterRepo repository.ClusterRepositoryInterface
	clientCache *kube.ClientCache
	detector    *ServiceDetector
	cfg         *config.Config

	// Track running discovery tasks to prevent duplicates
	runningTasks map[uuid.UUID]bool
	taskMutex    sync.RWMutex
}

// NewAutoDiscoveryService creates a new Prometheus auto-discovery service
func NewAutoDiscoveryService(
	clusterRepo repository.ClusterRepositoryInterface,
	clientCache *kube.ClientCache,
	cfg *config.Config,
) *AutoDiscoveryService {
	return &AutoDiscoveryService{
		clusterRepo:  clusterRepo,
		clientCache:  clientCache,
		detector:     NewServiceDetector(),
		cfg:          cfg,
		runningTasks: make(map[uuid.UUID]bool),
	}
}

// DiscoverForCluster performs Prometheus discovery for a single cluster
// This is an async operation - call in a goroutine
func (s *AutoDiscoveryService) DiscoverForCluster(ctx context.Context, cluster *models.Cluster) {
	clusterID := cluster.ID
	clusterName := cluster.Name

	// Check if auto-discovery is enabled
	if !s.cfg.Prometheus.AutoDiscovery {
		logrus.Debugf("Prometheus auto-discovery disabled for cluster %s", clusterName)
		return
	}

	// Check if a discovery task is already running for this cluster
	s.taskMutex.Lock()
	if s.runningTasks[clusterID] {
		logrus.Debugf("Discovery task already running for cluster %s", clusterName)
		s.taskMutex.Unlock()
		return
	}
	s.runningTasks[clusterID] = true
	s.taskMutex.Unlock()

	// Ensure task is cleaned up when done
	defer func() {
		s.taskMutex.Lock()
		delete(s.runningTasks, clusterID)
		s.taskMutex.Unlock()
	}()

	logrus.Infof("Starting Prometheus discovery for cluster %s", clusterName)

	// Get or create K8s client
	client, err := s.getOrCreateClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client for cluster %s: %v", clusterName, err)
		return
	}

	// Get discovery timeout from config (default 30s)
	timeout := time.Duration(s.cfg.Prometheus.DiscoveryTimeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Check if manual URL is configured for this cluster
	if s.cfg.Prometheus.ClusterURLs != nil {
		if manualURL, exists := s.cfg.Prometheus.ClusterURLs[clusterName]; exists && manualURL != "" {
			logrus.Infof("Using manual Prometheus URL for cluster %s: %s", clusterName, manualURL)
			s.updatePrometheusURL(ctx, clusterID, manualURL)
			return
		}
	}

	// Run detection with timeout
	result, err := s.detector.DetectWithTimeout(client, timeout)
	if err != nil {
		logrus.Warnf("Prometheus discovery failed for cluster %s: %v", clusterName, err)
		return
	}

	if !result.Found {
		logrus.Infof("No Prometheus service found in cluster %s", clusterName)
		return
	}

	// Update cluster with discovered Prometheus URL
	s.updatePrometheusURL(ctx, clusterID, result.URL)
	logrus.Infof("Prometheus discovered for cluster %s: %s", clusterName, result.URL)
}

// TriggerDiscoveryAsync triggers discovery for a cluster in background
// Returns immediately without waiting for discovery to complete
func (s *AutoDiscoveryService) TriggerDiscoveryAsync(cluster *models.Cluster) {
	go func() {
		ctx := context.Background()
		s.DiscoverForCluster(ctx, cluster)
	}()
}

// getOrCreateClient gets an existing client from cache or creates a new one
func (s *AutoDiscoveryService) getOrCreateClient(cluster *models.Cluster) (*kube.K8sClient, error) {
	// Check cache first
	if client, exists := s.clientCache.Get(cluster.ID); exists {
		return client, nil
	}

	// Decrypt kubeconfig
	// TODO: Implement decryption using utils.DecryptStringWithKey
	config := cluster.Config

	// Create new client
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(config))
	if err != nil {
		return nil, err
	}

	client, err := kube.NewClient(restConfig)
	if err != nil {
		return nil, err
	}

	// Cache the client
	s.clientCache.Set(cluster.ID, client)
	logrus.Infof("Created and cached K8s client for cluster %s", cluster.Name)

	return client, nil
}

// updatePrometheusURL updates the Prometheus URL in the database
func (s *AutoDiscoveryService) updatePrometheusURL(ctx context.Context, clusterID uuid.UUID, url string) {
	updates := map[string]interface{}{
		"prometheus_url": url,
	}

	if err := s.clusterRepo.Update(ctx, clusterID, updates); err != nil {
		logrus.Errorf("Failed to update Prometheus URL for cluster %s: %v", clusterID, err)
	}
}

// IsTaskRunning checks if a discovery task is currently running for a cluster
func (s *AutoDiscoveryService) IsTaskRunning(clusterID uuid.UUID) bool {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()
	return s.runningTasks[clusterID]
}
