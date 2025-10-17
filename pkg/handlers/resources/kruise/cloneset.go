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

// CloneSetHandler handles OpenKruise CloneSet operations
type CloneSetHandler struct {
	*crd.GenericCRDHandler
}

// NewCloneSetHandler creates a new CloneSet handler
func NewCloneSetHandler(client *kube.K8sClient) *CloneSetHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps.kruise.io",
		Version:  "v1alpha1",
		Resource: "clonesets",
	}
	return &CloneSetHandler{
		GenericCRDHandler: crd.NewGenericCRDHandler(client, gvr),
	}
}

// Restart triggers a rolling restart of the CloneSet
// This is done by updating the spec.template.metadata.annotations with a restart timestamp
func (h *CloneSetHandler) Restart(ctx context.Context, namespace, name string) error {
	// Get the CloneSet
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

	// Update the CloneSet
	return h.Update(ctx, obj)
}

// GetReplicas gets the current replica count
func (h *CloneSetHandler) GetReplicas(ctx context.Context, namespace, name string) (int32, error) {
	return h.GetScale(ctx, namespace, name)
}

// SetReplicas sets the replica count
func (h *CloneSetHandler) SetReplicas(ctx context.Context, namespace, name string, replicas int32) error {
	return h.UpdateScale(ctx, namespace, name, replicas)
}
