package services

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// K8sService manages Kubernetes clusters
type K8sService struct {
	clusterRepo         *repository.ClusterRepository
	resourceHistoryRepo *repository.ResourceHistoryRepository
}

// NewK8sService creates a new K8s service
func NewK8sService(
	clusterRepo *repository.ClusterRepository,
	resourceHistoryRepo *repository.ResourceHistoryRepository,
) *K8sService {
	return &K8sService{
		clusterRepo:         clusterRepo,
		resourceHistoryRepo: resourceHistoryRepo,
	}
}

// ImportClustersFromKubeconfig imports clusters from kubeconfig file
func (s *K8sService) ImportClustersFromKubeconfig(ctx context.Context, kubeconfigPath string) error {
	logrus.Infof("Importing clusters from kubeconfig: %s", kubeconfigPath)

	// Load kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	imported := 0
	for contextName, context := range config.Contexts {
		// Build single-cluster kubeconfig
		singleConfig := &clientcmdapi.Config{
			Clusters:       map[string]*clientcmdapi.Cluster{context.Cluster: config.Clusters[context.Cluster]},
			AuthInfos:      map[string]*clientcmdapi.AuthInfo{context.AuthInfo: config.AuthInfos[context.AuthInfo]},
			Contexts:       map[string]*clientcmdapi.Context{contextName: context},
			CurrentContext: contextName,
		}

		configBytes, err := clientcmd.Write(*singleConfig)
		if err != nil {
			logrus.Warnf("Failed to serialize config for %s: %v", contextName, err)
			continue
		}

		// Check if cluster already exists
		existing, err := s.clusterRepo.GetByName(ctx, contextName)
		if err == nil && existing != nil {
			logrus.Infof("Cluster %s already exists, skipping", contextName)
			continue
		}

		// Create new cluster
		cluster := &models.Cluster{
			Name:      contextName,
			Config:    string(configBytes),
			IsDefault: contextName == config.CurrentContext,
			Enable:    true,
		}

		if err := s.clusterRepo.Create(ctx, cluster); err != nil {
			logrus.Warnf("Failed to import cluster %s: %v", contextName, err)
			continue
		}

		imported++
		logrus.Infof("Imported cluster: %s", contextName)
	}

	logrus.Infof("Imported %d clusters from kubeconfig", imported)
	return nil
}

// ListClusters lists all clusters
func (s *K8sService) ListClusters(ctx context.Context) ([]*models.Cluster, error) {
	return s.clusterRepo.List(ctx)
}

// GetCluster gets a cluster by ID
func (s *K8sService) GetCluster(ctx context.Context, name string) (*models.Cluster, error) {
	return s.clusterRepo.GetByName(ctx, name)
}

// CreateCluster creates a new cluster
func (s *K8sService) CreateCluster(ctx context.Context, cluster *models.Cluster) error {
	// If this cluster should be default, clear existing default
	if cluster.IsDefault {
		if err := s.clusterRepo.ClearDefault(ctx); err != nil {
			return fmt.Errorf("failed to clear default cluster: %w", err)
		}
	}
	return s.clusterRepo.Create(ctx, cluster)
}

// UpdateCluster updates a cluster
func (s *K8sService) UpdateCluster(ctx context.Context, cluster *models.Cluster) error {
	// If setting as default, clear existing default
	if cluster.IsDefault {
		if err := s.clusterRepo.ClearDefault(ctx); err != nil {
			return fmt.Errorf("failed to clear default cluster: %w", err)
		}
	}

	// Build updates map
	updates := map[string]interface{}{
		"name":           cluster.Name,
		"description":    cluster.Description,
		"config":         cluster.Config,
		"in_cluster":     cluster.InCluster,
		"is_default":     cluster.IsDefault,
		"prometheus_url": cluster.PrometheusURL,
		"enable":         cluster.Enable,
	}

	return s.clusterRepo.Update(ctx, cluster.ID, updates)
}

// DeleteCluster deletes a cluster (soft delete)
func (s *K8sService) DeleteCluster(ctx context.Context, id string) error {
	cluster, err := s.clusterRepo.GetByName(ctx, id)
	if err != nil {
		return fmt.Errorf("cluster not found: %w", err)
	}
	return s.clusterRepo.Delete(ctx, cluster.ID)
}
