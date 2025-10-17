package traefik

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// IngressRouteHandler handles Traefik IngressRoute operations
type IngressRouteHandler struct {
	*crd.GenericCRDHandler
}

// NewIngressRouteHandler creates a new IngressRoute handler
func NewIngressRouteHandler(client *kube.K8sClient) *IngressRouteHandler {
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}
	return &IngressRouteHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
