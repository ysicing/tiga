package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/pkg/kube"
	"github.com/ysicing/tiga/pkg/prometheus"
)

type ClientSet struct {
	ClusterID  uuid.UUID // Cluster UUID for database references
	Name       string
	Version    string // Kubernetes version
	K8sClient  *kube.K8sClient
	PromClient *prometheus.Client

	config        string
	prometheusURL string
}

type ClusterManager struct {
	mu             sync.RWMutex // Protects clusters map from concurrent access
	clusters       map[string]*ClientSet
	defaultContext string
	clusterRepo    *repository.ClusterRepository
}

func createClientSetInCluster(clusterID uuid.UUID, name, prometheusURL string) (*ClientSet, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return newClientSet(clusterID, name, config, prometheusURL)
}

func createClientSetFromConfig(clusterID uuid.UUID, name, content, prometheusURL string) (*ClientSet, error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(content))
	if err != nil {
		logrus.Warnf("Failed to create REST config for cluster %s: %v", name, err)
		return nil, err
	}
	cs, err := newClientSet(clusterID, name, restConfig, prometheusURL)
	if err != nil {
		return nil, err
	}
	cs.config = content

	return cs, nil
}

func newClientSet(clusterID uuid.UUID, name string, k8sConfig *rest.Config, prometheusURL string) (*ClientSet, error) {
	cs := &ClientSet{
		ClusterID:     clusterID,
		Name:          name,
		prometheusURL: prometheusURL,
	}
	var err error
	cs.K8sClient, err = kube.NewClient(k8sConfig)
	if err != nil {
		logrus.Warnf("Failed to create k8s client for cluster %s: %v", name, err)
		return nil, err
	}

	if prometheusURL != "" {
		cs.PromClient, err = prometheus.NewClient(prometheusURL)
		if err != nil {
			logrus.Warnf("Failed to create Prometheus client for cluster %s, some features may not work as expected, err: %v", name, err)
		}
	}
	v, err := cs.K8sClient.ClientSet.Discovery().ServerVersion()
	if err != nil {
		logrus.Warnf("Failed to get server version for cluster %s: %v", name, err)
	} else {
		cs.Version = v.String()
	}
	logrus.Infof("Loaded K8s client for cluster: %s, version: %s", name, cs.Version)
	return cs, nil
}

func (cm *ClusterManager) GetClientSet(clusterName string) (*ClientSet, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if len(cm.clusters) == 0 {
		return nil, fmt.Errorf("no clusters available")
	}
	if clusterName == "" {
		if cm.defaultContext == "" {
			// If no default context is set, return the first available cluster
			for _, cs := range cm.clusters {
				return cs, nil
			}
		}
		// Recursive call - unlock first to avoid deadlock
		cm.mu.RUnlock()
		defer cm.mu.RLock() // Re-acquire before defer unlock
		return cm.GetClientSet(cm.defaultContext)
	}
	if cluster, ok := cm.clusters[clusterName]; ok {
		return cluster, nil
	}
	return nil, fmt.Errorf("cluster not found: %s", clusterName)
}

var (
	syncNow = make(chan struct{}, 1)
)

func syncClusters(cm *ClusterManager) error {
	clusters, err := cm.clusterRepo.List(context.Background())
	if err != nil {
		logrus.Warnf("list cluster err: %v", err)
		time.Sleep(5 * time.Second)
		return err
	}

	// Prepare updates outside the lock to minimize lock hold time
	dbClusterMap := make(map[string]interface{})
	updates := make(map[string]*ClientSet)      // Clusters to add/update
	removals := make([]string, 0)               // Clusters to remove
	var newDefaultContext string

	for _, cluster := range clusters {
		dbClusterMap[cluster.Name] = cluster
		if cluster.IsDefault {
			newDefaultContext = cluster.Name
		}

		// Get current cluster (with read lock)
		cm.mu.RLock()
		current, currentExist := cm.clusters[cluster.Name]
		cm.mu.RUnlock()

		if shouldUpdateCluster(current, cluster) {
			if currentExist {
				current.K8sClient.Stop(cluster.Name)
			}
			if cluster.Enable {
				clientSet, err := buildClientSet(cluster)
				if err != nil {
					logrus.Warnf("Failed to build k8s client for cluster %s, in cluster: %t, err: %v", cluster.Name, cluster.InCluster, err)
					continue
				}
				updates[cluster.Name] = clientSet
			} else if currentExist {
				// Cluster exists but is now disabled - mark for removal
				removals = append(removals, cluster.Name)
			}
		}
	}

	// Check for clusters to remove (not in DB anymore)
	cm.mu.RLock()
	for name, clientSet := range cm.clusters {
		if _, ok := dbClusterMap[name]; !ok {
			removals = append(removals, name)
			clientSet.K8sClient.Stop(name)
		}
	}
	cm.mu.RUnlock()

	// Apply all changes with write lock
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Apply updates
	for name, clientSet := range updates {
		cm.clusters[name] = clientSet
	}

	// Apply removals
	for _, name := range removals {
		delete(cm.clusters, name)
	}

	// Update default context
	if newDefaultContext != "" {
		cm.defaultContext = newDefaultContext
	}

	return nil
}

// shouldUpdateCluster decides whether the cached ClientSet needs to be updated
// based on the desired state from the database.
func shouldUpdateCluster(cs *ClientSet, cluster *models.Cluster) bool {
	// enable/disable toggle
	if (cs == nil && cluster.Enable) || (cs != nil && !cluster.Enable) {
		logrus.Infof("Cluster %s status changed, updating, enabled -> %v", cluster.Name, cluster.Enable)
		return true
	}
	if cs == nil && !cluster.Enable {
		return false
	}

	if cs == nil || cs.K8sClient == nil || cs.K8sClient.ClientSet == nil {
		return true
	}

	// kubeconfig change
	if cs.config != string(cluster.Config) {
		logrus.Infof("Kubeconfig changed for cluster %s, updating", cluster.Name)
		return true
	}

	// prometheus URL change
	if cs.prometheusURL != cluster.PrometheusURL {
		logrus.Infof("Prometheus URL changed for cluster %s, updating", cluster.Name)
		return true
	}

	// k8s version change
	// TODO: Replace direct ClientSet.Discovery() call with a small DiscoveryInterface.
	// current code depends on *kubernetes.Clientset, which is hard to mock in tests.
	version, err := cs.K8sClient.ClientSet.Discovery().ServerVersion()
	if err != nil {
		logrus.Warnf("Failed to get server version for cluster %s: %v", cluster.Name, err)
	} else if version.String() != cs.Version {
		logrus.Infof("Server version changed for cluster %s, updating, old: %s, new: %s", cluster.Name, cs.Version, version.String())
		return true
	}

	return false
}

func buildClientSet(cluster *models.Cluster) (*ClientSet, error) {
	if cluster.InCluster {
		return createClientSetInCluster(cluster.ID, cluster.Name, cluster.PrometheusURL)
	}
	return createClientSetFromConfig(cluster.ID, cluster.Name, string(cluster.Config), cluster.PrometheusURL)
}

func NewClusterManager() (*ClusterManager, error) {
	cm := new(ClusterManager)
	cm.clusters = make(map[string]*ClientSet)
	cm.clusterRepo = repository.NewClusterRepository(models.DB)

	// Background goroutine for periodic cluster sync
	go func() {
		// Initial sync (async to avoid blocking startup)
		if err := syncClusters(cm); err != nil {
			logrus.Warnf("Failed to sync clusters on startup: %v", err)
		}

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := syncClusters(cm); err != nil {
					logrus.Warnf("Failed to sync clusters: %v", err)
				}
			case <-syncNow:
				if err := syncClusters(cm); err != nil {
					logrus.Warnf("Failed to sync clusters: %v", err)
				}
			}
		}
	}()

	return cm, nil
}
