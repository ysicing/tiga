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

// StatefulSetHandler handles OpenKruise Advanced StatefulSet operations
type StatefulSetHandler struct {
	*crd.GenericCRDHandler
}

// NewStatefulSetHandler creates a new Advanced StatefulSet handler
func NewStatefulSetHandler(client *kube.K8sClient) *StatefulSetHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps.kruise.io",
		Version:  "v1beta1",
		Resource: "statefulsets",
	}
	return &StatefulSetHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}

// Restart triggers a rolling restart of the Advanced StatefulSet
// This is done by updating the spec.template.metadata.annotations with a restart timestamp
func (h *StatefulSetHandler) Restart(ctx context.Context, namespace, name string) error {
	// Get the StatefulSet
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

	// Update the StatefulSet
	return h.Update(ctx, obj)
}

// GetReplicas gets the current replica count
func (h *StatefulSetHandler) GetReplicas(ctx context.Context, namespace, name string) (int32, error) {
	return h.GetScale(ctx, namespace, name)
}

// SetReplicas sets the replica count
func (h *StatefulSetHandler) SetReplicas(ctx context.Context, namespace, name string, replicas int32) error {
	return h.UpdateScale(ctx, namespace, name, replicas)
}
