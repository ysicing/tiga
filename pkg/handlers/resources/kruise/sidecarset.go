package kruise

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// SidecarSetHandler handles OpenKruise SidecarSet operations
type SidecarSetHandler struct {
	*crd.GenericCRDHandler
}

// NewSidecarSetHandler creates a new SidecarSet handler
func NewSidecarSetHandler(client *kube.K8sClient) *SidecarSetHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps.kruise.io",
		Version:  "v1alpha1",
		Resource: "sidecarsets",
	}
	return &SidecarSetHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
