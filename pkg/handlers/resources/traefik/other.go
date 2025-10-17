package traefik

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// IngressRouteTCPHandler handles Traefik IngressRouteTCP operations
type IngressRouteTCPHandler struct {
	*crd.GenericCRDHandler
}

// NewIngressRouteTCPHandler creates a new IngressRouteTCP handler
func NewIngressRouteTCPHandler(client *kube.K8sClient) *IngressRouteTCPHandler {
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutetcps",
	}
	return &IngressRouteTCPHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}

// IngressRouteUDPHandler handles Traefik IngressRouteUDP operations
type IngressRouteUDPHandler struct {
	*crd.GenericCRDHandler
}

// NewIngressRouteUDPHandler creates a new IngressRouteUDP handler
func NewIngressRouteUDPHandler(client *kube.K8sClient) *IngressRouteUDPHandler {
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressrouteudps",
	}
	return &IngressRouteUDPHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
