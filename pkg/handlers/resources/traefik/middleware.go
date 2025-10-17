package traefik

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// MiddlewareHandler handles Traefik Middleware operations
type MiddlewareHandler struct {
	*crd.GenericCRDHandler
}

// NewMiddlewareHandler creates a new Middleware handler
func NewMiddlewareHandler(client *kube.K8sClient) *MiddlewareHandler {
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "middlewares",
	}
	return &MiddlewareHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
