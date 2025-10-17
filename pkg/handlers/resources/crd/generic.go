package crd

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ysicing/tiga/pkg/kube"
)

// GenericCRDHandler provides generic CRUD operations for any CRD
type GenericCRDHandler struct {
	client *kube.K8sClient
	gvr    schema.GroupVersionResource
}

// NewGenericCRDHandler creates a new generic CRD handler
// Example: NewGenericCRDHandler(client, schema.GroupVersionResource{
//     Group:    "apps.kruise.io",
//     Version:  "v1alpha1",
//     Resource: "clonesets",
// })
func NewGenericCRDHandler(client *kube.K8sClient, gvr schema.GroupVersionResource) *GenericCRDHandler {
	return &GenericCRDHandler{
		client: client,
		gvr:    gvr,
	}
}

// List retrieves all resources of this CRD type in a namespace
// If namespace is empty, retrieves cluster-scoped resources
func (h *GenericCRDHandler) List(ctx context.Context, namespace string) (*unstructured.UnstructuredList, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   h.gvr.Group,
		Version: h.gvr.Version,
		Kind:    h.gvr.Resource, // Will be corrected by the API
	})

	var listOpts []client.ListOption
	if namespace != "" {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}

	if err := h.client.List(ctx, list, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", h.gvr.Resource, err)
	}

	return list, nil
}

// Get retrieves a single resource by name and namespace
func (h *GenericCRDHandler) Get(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   h.gvr.Group,
		Version: h.gvr.Version,
		Kind:    h.gvr.Resource,
	})

	key := client.ObjectKey{Name: name}
	if namespace != "" {
		key.Namespace = namespace
	}

	if err := h.client.Get(ctx, key, obj); err != nil {
		return nil, fmt.Errorf("failed to get %s/%s: %w", namespace, name, err)
	}

	return obj, nil
}

// Create creates a new resource
func (h *GenericCRDHandler) Create(ctx context.Context, obj *unstructured.Unstructured) error {
	if err := h.client.Create(ctx, obj); err != nil {
		return fmt.Errorf("failed to create %s: %w", obj.GetName(), err)
	}
	return nil
}

// Update updates an existing resource
func (h *GenericCRDHandler) Update(ctx context.Context, obj *unstructured.Unstructured) error {
	if err := h.client.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update %s: %w", obj.GetName(), err)
	}
	return nil
}

// Delete deletes a resource
func (h *GenericCRDHandler) Delete(ctx context.Context, namespace, name string) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   h.gvr.Group,
		Version: h.gvr.Version,
		Kind:    h.gvr.Resource,
	})
	obj.SetNamespace(namespace)
	obj.SetName(name)

	if err := h.client.Delete(ctx, obj); err != nil {
		return fmt.Errorf("failed to delete %s/%s: %w", namespace, name, err)
	}
	return nil
}

// Patch patches a resource with a JSON patch
func (h *GenericCRDHandler) Patch(ctx context.Context, namespace, name string, patchData []byte, patchType client.Patch) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   h.gvr.Group,
		Version: h.gvr.Version,
		Kind:    h.gvr.Resource,
	})
	obj.SetNamespace(namespace)
	obj.SetName(name)

	if err := h.client.Patch(ctx, obj, patchType); err != nil {
		return fmt.Errorf("failed to patch %s/%s: %w", namespace, name, err)
	}
	return nil
}

// GetScale retrieves the scale subresource for a resource
func (h *GenericCRDHandler) GetScale(ctx context.Context, namespace, name string) (int32, error) {
	obj, err := h.Get(ctx, namespace, name)
	if err != nil {
		return 0, err
	}

	// Try to get replicas from spec.replicas
	replicas, found, err := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	if err != nil {
		return 0, fmt.Errorf("failed to get replicas: %w", err)
	}
	if !found {
		return 0, fmt.Errorf("replicas not found in spec")
	}

	return int32(replicas), nil
}

// UpdateScale updates the scale subresource
func (h *GenericCRDHandler) UpdateScale(ctx context.Context, namespace, name string, replicas int32) error {
	obj, err := h.Get(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Update spec.replicas
	if err := unstructured.SetNestedField(obj.Object, int64(replicas), "spec", "replicas"); err != nil {
		return fmt.Errorf("failed to set replicas: %w", err)
	}

	return h.Update(ctx, obj)
}

// IsCRDInstalled checks if the CRD is installed in the cluster
func IsCRDInstalled(ctx context.Context, client *kube.K8sClient, crdName string) (bool, error) {
	crdList := &unstructured.UnstructuredList{}
	crdList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinitionList",
	})

	if err := client.List(ctx, crdList); err != nil {
		return false, fmt.Errorf("failed to list CRDs: %w", err)
	}

	for _, item := range crdList.Items {
		if item.GetName() == crdName {
			return true, nil
		}
	}

	return false, nil
}

// GetCRDsByGroup retrieves all CRDs in a specific API group
func GetCRDsByGroup(ctx context.Context, client *kube.K8sClient, group string) ([]string, error) {
	crdList := &unstructured.UnstructuredList{}
	crdList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinitionList",
	})

	if err := client.List(ctx, crdList); err != nil {
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

	var crdNames []string
	for _, item := range crdList.Items {
		// Get the group from spec.group
		itemGroup, found, err := unstructured.NestedString(item.Object, "spec", "group")
		if err != nil || !found {
			continue
		}
		if itemGroup == group {
			crdNames = append(crdNames, item.GetName())
		}
	}

	return crdNames, nil
}
