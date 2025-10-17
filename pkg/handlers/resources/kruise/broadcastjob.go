package kruise

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// BroadcastJobHandler handles OpenKruise BroadcastJob operations
type BroadcastJobHandler struct {
	*crd.GenericCRDHandler
}

// NewBroadcastJobHandler creates a new BroadcastJob handler
func NewBroadcastJobHandler(client *kube.K8sClient) *BroadcastJobHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps.kruise.io",
		Version:  "v1alpha1",
		Resource: "broadcastjobs",
	}
	return &BroadcastJobHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
