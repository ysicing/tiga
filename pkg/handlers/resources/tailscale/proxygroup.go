package tailscale

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// ProxyGroupHandler handles Tailscale ProxyGroup operations
type ProxyGroupHandler struct {
	*crd.GenericCRDHandler
}

// NewProxyGroupHandler creates a new ProxyGroup handler
func NewProxyGroupHandler(client *kube.K8sClient) *ProxyGroupHandler {
	gvr := schema.GroupVersionResource{
		Group:    "tailscale.com",
		Version:  "v1alpha1",
		Resource: "proxygroups",
	}
	return &ProxyGroupHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
