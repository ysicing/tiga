package cluster

import (
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/pkg/kube"
)

func Test_shouldUpdateCluster(t *testing.T) {
	type args struct {
		cs      *ClientSet
		cluster *models.Cluster
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "enable/disable toggle, disable -> enable",
			args: args{
				cs:      nil,
				cluster: &models.Cluster{Name: "test", Enable: true},
			},
			want: true,
		},
		{
			name: "enable/disable toggle, enable -> disable",
			args: args{
				cs: &ClientSet{
					Name: "test",
				},
				cluster: &models.Cluster{Name: "test", Enable: false},
			},
			want: true,
		},
		{
			name: "disable cluster, keep disable",
			args: args{
				cs:      nil,
				cluster: &models.Cluster{Name: "test", Enable: false},
			},
			want: false,
		},
		{
			name: "invalid ClientSet(nil k8sClient), need update",
			args: args{
				cs: &ClientSet{
					Name:      "test",
					Version:   "v1.34.0",
					K8sClient: nil,
				},
				cluster: &models.Cluster{Name: "test", Enable: true},
			},
			want: true,
		},
		{
			name: "invalid ClientSet(nil k8sClient.ClientSet), need update",
			args: args{
				cs: &ClientSet{
					Name:    "test",
					Version: "v1.34.0",
					K8sClient: &kube.K8sClient{
						ClientSet: nil,
					},
				},
				cluster: &models.Cluster{Name: "test", Enable: true},
			},
			want: true,
		},
		{
			name: "k8s config change, need update",
			args: args{
				cs: &ClientSet{
					Name:    "test",
					Version: "v1.34.0",
					K8sClient: &kube.K8sClient{
						ClientSet: &kubernetes.Clientset{},
					},
					configHash: hashConfig("test-config"),
				},
				cluster: &models.Cluster{Name: "test", Enable: true, Config: "test-config-new"},
			},
			want: true,
		},
		{
			name: "prometheus url change, need update",
			args: args{
				cs: &ClientSet{
					Name:    "test",
					Version: "v1.34.0",
					K8sClient: &kube.K8sClient{
						ClientSet: &kubernetes.Clientset{},
					},
					prometheusURL: "test-prometheus-url",
				},
				cluster: &models.Cluster{Name: "test", Enable: true, PrometheusURL: "test-prometheus-url-new"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUpdateCluster(tt.args.cs, tt.args.cluster); got != tt.want {
				t.Errorf("shouldUpdateCluster() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("k8s version change, need update", func(t *testing.T) {
		mockey.PatchConvey("mock ServerVersion change", t, func() {
			mockey.Mock((*discovery.DiscoveryClient).ServerVersion).
				Return(&version.Info{GitVersion: "v1.34.0"}, nil).Build()
			cs := &ClientSet{
				Name:    "test",
				Version: "v1.33.0",
				K8sClient: &kube.K8sClient{
					ClientSet: &kubernetes.Clientset{DiscoveryClient: &discovery.DiscoveryClient{}},
				},
			}
			cluster := &models.Cluster{Name: "test", Enable: true}

			got := shouldUpdateCluster(cs, cluster)
			assert.True(t, got, "expected update when k8s version changed")
		})
	})

	t.Run("same, skip update", func(t *testing.T) {
		mockey.PatchConvey("mock ServerVersion change", t, func() {
			mockey.Mock((*discovery.DiscoveryClient).ServerVersion).
				Return(&version.Info{GitVersion: "v1.34.0"}, nil).Build()
			cs := &ClientSet{
				Name:    "test",
				Version: "v1.34.0",
				K8sClient: &kube.K8sClient{
					ClientSet: &kubernetes.Clientset{DiscoveryClient: &discovery.DiscoveryClient{}},
				},
				configHash:    hashConfig("test-config"),
				prometheusURL: "test-prometheus-url",
			}
			cluster := &models.Cluster{
				Name:          "test",
				Enable:        true,
				Config:        "test-config",
				PrometheusURL: "test-prometheus-url",
			}
			got := shouldUpdateCluster(cs, cluster)
			assert.False(t, got, "expected no update when all the same")
		})
	})
}
