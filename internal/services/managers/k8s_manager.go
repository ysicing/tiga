package managers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ysicing/tiga/internal/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sManager manages Kubernetes instances
type K8sManager struct {
	*BaseManager
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// NewK8sManager creates a new Kubernetes manager
func NewK8sManager() *K8sManager {
	return &K8sManager{
		BaseManager: NewBaseManager(),
	}
}

// Initialize initializes the Kubernetes manager
func (m *K8sManager) Initialize(ctx context.Context, instance *models.Instance) error {
	if err := m.BaseManager.Initialize(ctx, instance); err != nil {
		return err
	}
	return nil
}

// Connect establishes connection to Kubernetes
func (m *K8sManager) Connect(ctx context.Context) error {
	// Get kubeconfig from config
	kubeconfig := m.GetConfigValue("kubeconfig", "").(string)
	if kubeconfig == "" {
		return fmt.Errorf("%w: kubeconfig is required", ErrInvalidConfig)
	}

	// Build config from kubeconfig
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Test connection by getting server version
	if _, err := clientset.Discovery().ServerVersion(); err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	m.config = config
	m.clientset = clientset
	return nil
}

// Disconnect closes connection to Kubernetes
func (m *K8sManager) Disconnect(ctx context.Context) error {
	m.clientset = nil
	m.config = nil
	return nil
}

// HealthCheck checks Kubernetes health
func (m *K8sManager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if m.clientset == nil {
		return nil, ErrNotConnected
	}

	start := time.Now()
	status := &HealthStatus{
		Healthy:       false,
		LastCheckTime: time.Now().Format(time.RFC3339),
		Details:       make(map[string]interface{}),
	}

	// Get server version
	version, err := m.clientset.Discovery().ServerVersion()
	if err != nil {
		status.Message = fmt.Sprintf("Health check failed: %v", err)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status, nil
	}

	status.Details["version"] = version.GitVersion
	status.Details["platform"] = version.Platform

	// Get node count
	nodes, err := m.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		status.Details["node_count"] = len(nodes.Items)
		readyNodes := 0
		for _, node := range nodes.Items {
			for _, condition := range node.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == "True" {
					readyNodes++
					break
				}
			}
		}
		status.Details["ready_nodes"] = readyNodes
	}

	status.Healthy = true
	status.Message = "Kubernetes is healthy"
	status.ResponseTime = time.Since(start).Milliseconds()

	return status, nil
}

// CollectMetrics collects metrics from Kubernetes
func (m *K8sManager) CollectMetrics(ctx context.Context) (*ServiceMetrics, error) {
	if m.clientset == nil {
		return nil, ErrNotConnected
	}

	metrics := &ServiceMetrics{
		InstanceID: m.instance.ID,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metrics:    make(map[string]interface{}),
	}

	// Get node metrics
	nodes, err := m.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricsCollectionFailed, err)
	}

	metrics.Metrics["node_count"] = len(nodes.Items)

	readyNodes := 0
	var totalCPU, totalMemory int64
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				readyNodes++
				break
			}
		}

		cpu := node.Status.Capacity.Cpu().MilliValue()
		memory := node.Status.Capacity.Memory().Value()
		totalCPU += cpu
		totalMemory += memory
	}

	metrics.Metrics["ready_nodes"] = readyNodes
	metrics.Metrics["total_cpu_millicores"] = totalCPU
	metrics.Metrics["total_memory_bytes"] = totalMemory

	// Get pod metrics
	pods, err := m.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		metrics.Metrics["pod_count"] = len(pods.Items)

		runningPods := 0
		pendingPods := 0
		failedPods := 0
		for _, pod := range pods.Items {
			switch pod.Status.Phase {
			case "Running":
				runningPods++
			case "Pending":
				pendingPods++
			case "Failed":
				failedPods++
			}
		}

		metrics.Metrics["running_pods"] = runningPods
		metrics.Metrics["pending_pods"] = pendingPods
		metrics.Metrics["failed_pods"] = failedPods
	}

	// Get namespace count
	namespaces, err := m.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		metrics.Metrics["namespace_count"] = len(namespaces.Items)
	}

	// Get deployment count
	deployments, err := m.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err == nil {
		metrics.Metrics["deployment_count"] = len(deployments.Items)
	}

	// Get service count
	services, err := m.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err == nil {
		metrics.Metrics["service_count"] = len(services.Items)
	}

	return metrics, nil
}

// GetInfo returns Kubernetes service information
func (m *K8sManager) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	if m.clientset == nil {
		return nil, ErrNotConnected
	}

	info := make(map[string]interface{})

	// Get server version
	version, err := m.clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	info["version"] = version.GitVersion
	info["platform"] = version.Platform
	info["build_date"] = version.BuildDate
	info["go_version"] = version.GoVersion

	// Get nodes
	nodes, err := m.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		nodeList := make([]map[string]interface{}, 0, len(nodes.Items))
		for _, node := range nodes.Items {
			nodeInfo := map[string]interface{}{
				"name":    node.Name,
				"labels":  node.Labels,
				"os":      node.Status.NodeInfo.OSImage,
				"kernel":  node.Status.NodeInfo.KernelVersion,
				"cpu":     node.Status.Capacity.Cpu().String(),
				"memory":  node.Status.Capacity.Memory().String(),
				"kubelet": node.Status.NodeInfo.KubeletVersion,
				"runtime": node.Status.NodeInfo.ContainerRuntimeVersion,
			}

			// Get node status
			for _, condition := range node.Status.Conditions {
				if condition.Type == "Ready" {
					nodeInfo["ready"] = condition.Status == "True"
					break
				}
			}

			nodeList = append(nodeList, nodeInfo)
		}
		info["nodes"] = nodeList
	}

	// Get namespaces
	namespaces, err := m.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		nsList := make([]string, 0, len(namespaces.Items))
		for _, ns := range namespaces.Items {
			nsList = append(nsList, ns.Name)
		}
		info["namespaces"] = nsList
	}

	return info, nil
}

// ValidateConfig validates Kubernetes configuration
func (m *K8sManager) ValidateConfig(config map[string]interface{}) error {
	kubeconfig, ok := config["kubeconfig"].(string)
	if !ok || kubeconfig == "" {
		return fmt.Errorf("%w: kubeconfig is required", ErrInvalidConfig)
	}

	// Try to parse kubeconfig
	if _, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig)); err != nil {
		return fmt.Errorf("%w: invalid kubeconfig: %v", ErrInvalidConfig, err)
	}

	return nil
}

// Type returns the service type
func (m *K8sManager) Type() string {
	return "k8s"
}
