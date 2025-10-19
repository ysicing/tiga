package kube

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	toolscache "k8s.io/client-go/tools/cache"
	metricsv1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var runtimeScheme = runtime.NewScheme()

func init() {
	// Disable controller-runtime logging to avoid klog dependency
	log.SetLogger(logr.New(log.NullLogSink{}))
	_ = scheme.AddToScheme(runtimeScheme)
	_ = apiextensionsv1.AddToScheme(runtimeScheme)
	_ = gatewayapiv1.Install(runtimeScheme)
	_ = metricsv1.AddToScheme(runtimeScheme)
}

// K8sClient holds the Kubernetes client instances
type K8sClient struct {
	client.Client
	ClientSet     kubernetes.Interface // Changed to interface to support both real and fake clients
	Configuration *rest.Config
	MetricsClient *metricsclient.Clientset

	cancel context.CancelFunc
}

// NewClient creates a K8sClient from a rest.Config
func NewClient(config *rest.Config) (*K8sClient, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	metricsClient, err := metricsclient.NewForConfig(config)
	if err != nil {
		logrus.Warnf("failed to create metrics client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var c client.Client
	if os.Getenv("DISABLE_CACHE") == "true" {
		c, err = client.New(config, client.Options{
			Scheme: runtimeScheme,
		})
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
	} else {
		mgr, err := manager.New(config, manager.Options{
			Scheme:         runtimeScheme,
			LeaderElection: false,
			Metrics: metricsserver.Options{
				BindAddress: "0", // Disable metrics server
			},
			Cache: cache.Options{
				DefaultWatchErrorHandler: func(ctx context.Context, r *toolscache.Reflector, err error) {
				},
			},
		})
		if err != nil {
			cancel()
			return nil, err
		}

		// Add field indexer for Pod spec.nodeName to enable efficient querying by node
		if err := mgr.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, "spec.nodeName", func(rawObj client.Object) []string {
			pod := rawObj.(*corev1.Pod)
			if pod.Spec.NodeName == "" {
				return nil
			}
			return []string{pod.Spec.NodeName}
		}); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create field indexer for spec.nodeName: %w", err)
		}
		go func() {
			if err := mgr.Start(ctx); err != nil {
				fmt.Printf("Error starting manager: %v\n", err)
			}
		}()
		if !mgr.GetCache().WaitForCacheSync(ctx) {
			cancel()
			return nil, fmt.Errorf("failed to wait for cache sync")
		}
		c = mgr.GetClient()
	}

	return &K8sClient{
		Client:        c,
		ClientSet:     clientset,
		Configuration: config,
		MetricsClient: metricsClient,
		cancel:        cancel,
	}, nil
}

func (k *K8sClient) Stop(name string) {
	logrus.Infof("Stopping K8s client for %s", name)
	k.cancel()
}

// GetScheme returns the runtime scheme used by the client
func GetScheme() *runtime.Scheme {
	return runtimeScheme
}

// ClientCache manages K8s client instances for multiple clusters (Phase 0)
// Thread-safe cache using cluster UUID as key
type ClientCache struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]*K8sClient
}

// NewClientCache creates a new ClientCache instance
func NewClientCache() *ClientCache {
	return &ClientCache{
		clients: make(map[uuid.UUID]*K8sClient),
	}
}

// Get retrieves a client from cache by cluster ID
func (c *ClientCache) Get(clusterID uuid.UUID) (*K8sClient, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	client, exists := c.clients[clusterID]
	return client, exists
}

// Set stores a client in cache by cluster ID
func (c *ClientCache) Set(clusterID uuid.UUID, client *K8sClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[clusterID] = client
}

// Delete removes a client from cache and stops it
func (c *ClientCache) Delete(clusterID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if client, exists := c.clients[clusterID]; exists {
		client.Stop(clusterID.String())
		delete(c.clients, clusterID)
	}
}

// Clear removes all clients from cache and stops them
func (c *ClientCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, client := range c.clients {
		client.Stop(id.String())
	}
	c.clients = make(map[uuid.UUID]*K8sClient)
}

// Count returns the number of cached clients
func (c *ClientCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.clients)
}
