package kruise

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// DaemonSetHandler handles OpenKruise Advanced DaemonSet operations
type DaemonSetHandler struct {
	*crd.GenericCRDHandler
}

// NewDaemonSetHandler creates a new Advanced DaemonSet handler
func NewDaemonSetHandler(client *kube.K8sClient) *DaemonSetHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps.kruise.io",
		Version:  "v1alpha1",
		Resource: "daemonsets",
	}
	return &DaemonSetHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}

// Restart triggers a rolling restart of the Advanced DaemonSet
// This is done by updating the spec.template.metadata.annotations with a restart timestamp
func (h *DaemonSetHandler) Restart(ctx context.Context, namespace, name string) error {
	// Get the DaemonSet
	obj, err := h.Get(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Add or update restart annotation
	annotations, found, err := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	if err != nil {
		return fmt.Errorf("failed to get annotations: %w", err)
	}
	if !found || annotations == nil {
		annotations = make(map[string]string)
	}

	// Set restart timestamp
	annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// Update annotations
	if err := unstructured.SetNestedStringMap(obj.Object, annotations, "spec", "template", "metadata", "annotations"); err != nil {
		return fmt.Errorf("failed to set annotations: %w", err)
	}

	// Update the DaemonSet
	return h.Update(ctx, obj)
}
