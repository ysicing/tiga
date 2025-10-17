package kruise

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// AdvancedCronJobHandler handles OpenKruise AdvancedCronJob operations
type AdvancedCronJobHandler struct {
	*crd.GenericCRDHandler
}

// NewAdvancedCronJobHandler creates a new AdvancedCronJob handler
func NewAdvancedCronJobHandler(client *kube.K8sClient) *AdvancedCronJobHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps.kruise.io",
		Version:  "v1alpha1",
		Resource: "advancedcronjobs",
	}
	return &AdvancedCronJobHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}
