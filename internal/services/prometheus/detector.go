package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/pkg/kube"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceDetector detects Prometheus services in a Kubernetes cluster
type ServiceDetector struct {
	// Common Prometheus service names
	serviceNames []string
	// Common Prometheus namespaces
	namespaces []string
	// Service port (default 9090)
	defaultPort int32
}

// NewServiceDetector creates a new Prometheus ServiceDetector
func NewServiceDetector() *ServiceDetector {
	return &ServiceDetector{
		serviceNames: []string{
			"prometheus",
			"prometheus-server",
			"prometheus-k8s",
			"kube-prometheus-stack-prometheus",
			"prometheus-operated",
			"prometheus-service",
		},
		namespaces: []string{
			"monitoring",
			"prometheus",
			"kube-prometheus",
			"kube-system",
			"default",
		},
		defaultPort: 9090,
	}
}

// DetectResult represents the result of Prometheus detection
type DetectResult struct {
	Found     bool   // Whether Prometheus was found
	URL       string // Prometheus URL (format: http://service.namespace.svc.cluster.local:port)
	Namespace string // Namespace where Prometheus was found
	Service   string // Service name
	Port      int32  // Service port
}

// Detect attempts to find a Prometheus service in the cluster
// Returns the first matching service found
func (d *ServiceDetector) Detect(ctx context.Context, client *kube.K8sClient) (*DetectResult, error) {
	logrus.Debug("Starting Prometheus service detection")

	// Try each namespace
	for _, namespace := range d.namespaces {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// List all services in this namespace
		services, err := client.ClientSet.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			logrus.Debugf("Failed to list services in namespace %s: %v", namespace, err)
			continue
		}

		// Check each service against known Prometheus service names
		for _, svc := range services.Items {
			for _, expectedName := range d.serviceNames {
				if svc.Name == expectedName {
					// Found a matching service
					port := d.defaultPort
					if len(svc.Spec.Ports) > 0 {
						port = svc.Spec.Ports[0].Port
					}

					url := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
						svc.Name, svc.Namespace, port)

					logrus.Infof("Prometheus detected: %s in namespace %s", svc.Name, svc.Namespace)

					return &DetectResult{
						Found:     true,
						URL:       url,
						Namespace: svc.Namespace,
						Service:   svc.Name,
						Port:      port,
					}, nil
				}
			}
		}
	}

	logrus.Debug("No Prometheus service found")
	return &DetectResult{Found: false}, nil
}

// DetectWithTimeout runs detection with a timeout
func (d *ServiceDetector) DetectWithTimeout(client *kube.K8sClient, timeout time.Duration) (*DetectResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return d.Detect(ctx, client)
}
