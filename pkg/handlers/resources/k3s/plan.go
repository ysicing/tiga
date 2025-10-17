package k3s

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// PlanHandler handles K3s Upgrade Plan operations
type PlanHandler struct {
	*crd.GenericCRDHandler
}

// NewPlanHandler creates a new Plan handler
func NewPlanHandler(client *kube.K8sClient) *PlanHandler {
	gvr := schema.GroupVersionResource{
		Group:    "upgrade.cattle.io",
		Version:  "v1",
		Resource: "plans",
	}
	return &PlanHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
