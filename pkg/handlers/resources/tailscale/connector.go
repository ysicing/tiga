package tailscale

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// ConnectorHandler handles Tailscale Connector operations
type ConnectorHandler struct {
	*crd.GenericCRDHandler
}

// NewConnectorHandler creates a new Connector handler
func NewConnectorHandler(client *kube.K8sClient) *ConnectorHandler {
	gvr := schema.GroupVersionResource{
		Group:    "tailscale.com",
		Version:  "v1alpha1",
		Resource: "connectors",
	}
	return &ConnectorHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
