package tailscale

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// ProxyClassHandler handles Tailscale ProxyClass operations
type ProxyClassHandler struct {
	*crd.GenericCRDHandler
}

// NewProxyClassHandler creates a new ProxyClass handler
func NewProxyClassHandler(client *kube.K8sClient) *ProxyClassHandler {
	gvr := schema.GroupVersionResource{
		Group:    "tailscale.com",
		Version:  "v1alpha1",
		Resource: "proxyclasses",
	}
	return &ProxyClassHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
