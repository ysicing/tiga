package k8s_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/ysicing/tiga/internal/services/prometheus"
	"github.com/ysicing/tiga/pkg/kube"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestServiceDetector_Detect tests Prometheus service detection
func TestServiceDetector_Detect(t *testing.T) {
	detector := prometheus.NewServiceDetector()

	t.Run("DetectPrometheusServer", func(t *testing.T) {
		// Create fake clientset with Prometheus service
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus-server",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "http",
							Port: 9090,
						},
					},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		assert.True(t, result.Found, "Should find Prometheus service")
		assert.Equal(t, "monitoring", result.Namespace)
		assert.Equal(t, "prometheus-server", result.Service)
		assert.Equal(t, int32(9090), result.Port)
		assert.Equal(t, "http://prometheus-server.monitoring.svc.cluster.local:9090", result.URL)
	})

	t.Run("DetectPrometheusK8s", func(t *testing.T) {
		// Test with prometheus-k8s (Prometheus Operator)
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus-k8s",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "web",
							Port: 9090,
						},
					},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		assert.True(t, result.Found)
		assert.Equal(t, "prometheus-k8s", result.Service)
	})

	t.Run("DetectKubePrometheusStack", func(t *testing.T) {
		// Test with kube-prometheus-stack (Helm chart)
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kube-prometheus-stack-prometheus",
					Namespace: "prometheus",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port: 9090,
						},
					},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		assert.True(t, result.Found)
		assert.Equal(t, "prometheus", result.Namespace)
		assert.Equal(t, "kube-prometheus-stack-prometheus", result.Service)
	})

	t.Run("NoPrometheusFound", func(t *testing.T) {
		// Empty cluster with no Prometheus
		clientset := fake.NewSimpleClientset()

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		assert.False(t, result.Found, "Should not find Prometheus in empty cluster")
		assert.Empty(t, result.URL)
	})

	t.Run("WrongServiceName", func(t *testing.T) {
		// Service exists but with wrong name
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "grafana",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 3000}},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		assert.False(t, result.Found, "Should not detect Grafana as Prometheus")
	})
}

// TestServiceDetector_NamespacePriority tests namespace search priority
func TestServiceDetector_NamespacePriority(t *testing.T) {
	detector := prometheus.NewServiceDetector()

	t.Run("MonitoringNamespaceFirst", func(t *testing.T) {
		// Create Prometheus in both monitoring and default namespaces
		// Should find monitoring namespace first (higher priority)
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 9090}},
				},
			},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 9090}},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		// Should find the one in "monitoring" namespace first (higher priority)
		assert.True(t, result.Found)
		assert.Equal(t, "monitoring", result.Namespace,
			"Should prioritize 'monitoring' namespace over 'default'")
	})
}

// TestServiceDetector_CustomPort tests custom port detection
func TestServiceDetector_CustomPort(t *testing.T) {
	detector := prometheus.NewServiceDetector()

	t.Run("CustomPort", func(t *testing.T) {
		// Prometheus with custom port 9091
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "web",
							Port: 9091, // Custom port
						},
					},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		assert.True(t, result.Found)
		assert.Equal(t, int32(9091), result.Port, "Should detect custom port")
		assert.Contains(t, result.URL, ":9091", "URL should contain custom port")
	})

	t.Run("NoPorts", func(t *testing.T) {
		// Service without ports (should use default 9090)
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		assert.True(t, result.Found)
		assert.Equal(t, int32(9090), result.Port, "Should use default port 9090 when no ports defined")
	})
}

// TestServiceDetector_Timeout tests detection with timeout
func TestServiceDetector_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	detector := prometheus.NewServiceDetector()

	t.Run("DetectionWithTimeout", func(t *testing.T) {
		// Create normal service
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 9090}},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		// Should complete within timeout
		result, err := detector.DetectWithTimeout(k8sClient, 5*time.Second)
		require.NoError(t, err)

		assert.True(t, result.Found)
		t.Logf("Detection completed successfully within timeout")
	})

	t.Run("VeryShortTimeout", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		// Very short timeout should still complete (fake client is fast)
		result, err := detector.DetectWithTimeout(k8sClient, 100*time.Millisecond)
		require.NoError(t, err)

		assert.False(t, result.Found)
		t.Logf("Detection completed within 100ms timeout")
	})
}

// TestServiceDetector_MultipleServices tests behavior with multiple Prometheus services
func TestServiceDetector_MultipleServices(t *testing.T) {
	detector := prometheus.NewServiceDetector()

	t.Run("MultiplePrometheuServices", func(t *testing.T) {
		// Multiple Prometheus services in same namespace
		// Should return the first match according to service names priority
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus-operated",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 9090}},
				},
			},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus-server",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 9090}},
				},
			},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 9090}},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		result, err := detector.Detect(context.Background(), k8sClient)
		require.NoError(t, err)

		assert.True(t, result.Found)
		assert.Equal(t, "monitoring", result.Namespace)
		// Should find one of the Prometheus services
		assert.Contains(t, []string{"prometheus", "prometheus-server", "prometheus-operated"},
			result.Service, "Should find a valid Prometheus service")

		t.Logf("Found Prometheus service: %s", result.Service)
	})
}

// TestServiceDetector_ContextCancellation tests context cancellation
func TestServiceDetector_ContextCancellation(t *testing.T) {
	detector := prometheus.NewServiceDetector()

	t.Run("CancelledContext", func(t *testing.T) {
		clientset := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus",
					Namespace: "monitoring",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 9090}},
				},
			},
		)

		k8sClient := &kube.K8sClient{
			ClientSet: clientset,
		}

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := detector.Detect(ctx, k8sClient)

		// Should handle cancelled context gracefully
		// Either returns an error or returns a result (depending on timing)
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled, "Should return context.Canceled error")
		} else {
			// If it completes before checking context, that's also valid
			t.Logf("Detection completed before context cancellation was checked")
		}

		if result != nil {
			t.Logf("Result: Found=%v", result.Found)
		}
	})
}
