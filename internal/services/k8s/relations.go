package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// ResourceRelation represents a relationship between two resources
type ResourceRelation struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	UID       string `json:"uid"`
	Type      string `json:"type"` // "owner", "owned", "reference"
}

// RelationsService handles resource relationship queries
type RelationsService struct {
	maxDepth int
}

// NewRelationsService creates a new relations service
func NewRelationsService() *RelationsService {
	return &RelationsService{
		maxDepth: 3, // Limit recursion depth to prevent infinite loops
	}
}

// staticRelationshipMap defines known resource relationships
// Key: parent resource kind, Value: child resource kind
var staticRelationshipMap = map[string][]string{
	"Deployment":  {"ReplicaSet"},
	"ReplicaSet":  {"Pod"},
	"StatefulSet": {"Pod"},
	"DaemonSet":   {"Pod"},
	"Job":         {"Pod"},
	"CronJob":     {"Job"},
	"Service":     {"Endpoints", "EndpointSlice"},
	"Ingress":     {"Service"},
}

// GetRelatedResources recursively finds all related resources for a given resource
func (s *RelationsService) GetRelatedResources(
	ctx context.Context,
	client dynamic.Interface,
	namespace, kind, name string,
) ([]ResourceRelation, error) {
	visited := make(map[string]bool)
	var relations []ResourceRelation

	// Get the initial resource
	gvr, err := s.getGVR(kind)
	if err != nil {
		return nil, err
	}

	var resource *unstructured.Unstructured
	if namespace == "" {
		resource, err = client.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		resource, err = client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	// Track the initial resource as visited
	uid := string(resource.GetUID())
	visited[uid] = true

	// Find owned resources (children)
	if err := s.findOwnedResources(ctx, client, resource, &relations, visited, 0); err != nil {
		return nil, err
	}

	// Find owner resources (parents)
	if err := s.findOwnerResources(ctx, client, resource, &relations, visited, 0); err != nil {
		return nil, err
	}

	return relations, nil
}

// findOwnedResources finds resources owned by the current resource
func (s *RelationsService) findOwnedResources(
	ctx context.Context,
	client dynamic.Interface,
	owner *unstructured.Unstructured,
	relations *[]ResourceRelation,
	visited map[string]bool,
	depth int,
) error {
	if depth >= s.maxDepth {
		return nil
	}

	ownerKind := owner.GetKind()
	ownerNamespace := owner.GetNamespace()
	ownerUID := string(owner.GetUID())

	// Check static relationship map
	childKinds, exists := staticRelationshipMap[ownerKind]
	if !exists {
		return nil
	}

	for _, childKind := range childKinds {
		gvr, err := s.getGVR(childKind)
		if err != nil {
			continue
		}

		var list *unstructured.UnstructuredList
		if ownerNamespace == "" {
			list, err = client.Resource(gvr).List(ctx, metav1.ListOptions{})
		} else {
			list, err = client.Resource(gvr).Namespace(ownerNamespace).List(ctx, metav1.ListOptions{})
		}
		if err != nil {
			continue
		}

		for _, item := range list.Items {
			// Check if this resource is owned by the parent
			ownerRefs := item.GetOwnerReferences()
			for _, ref := range ownerRefs {
				if string(ref.UID) == ownerUID {
					childUID := string(item.GetUID())

					// Check for circular reference
					if visited[childUID] {
						continue
					}
					visited[childUID] = true

					// Add to relations
					*relations = append(*relations, ResourceRelation{
						Kind:      item.GetKind(),
						Name:      item.GetName(),
						Namespace: item.GetNamespace(),
						UID:       childUID,
						Type:      "owned",
					})

					// Recursively find children of this child
					if err := s.findOwnedResources(ctx, client, &item, relations, visited, depth+1); err != nil {
						continue
					}
				}
			}
		}
	}

	return nil
}

// findOwnerResources finds resources that own the current resource
func (s *RelationsService) findOwnerResources(
	ctx context.Context,
	client dynamic.Interface,
	resource *unstructured.Unstructured,
	relations *[]ResourceRelation,
	visited map[string]bool,
	depth int,
) error {
	if depth >= s.maxDepth {
		return nil
	}

	ownerRefs := resource.GetOwnerReferences()
	for _, ref := range ownerRefs {
		ownerUID := string(ref.UID)

		// Check for circular reference
		if visited[ownerUID] {
			continue
		}
		visited[ownerUID] = true

		// Try to get the owner resource
		gvr, err := s.getGVRFromAPIVersion(ref.APIVersion, ref.Kind)
		if err != nil {
			continue
		}

		var owner *unstructured.Unstructured
		namespace := resource.GetNamespace()
		if namespace == "" {
			owner, err = client.Resource(gvr).Get(ctx, ref.Name, metav1.GetOptions{})
		} else {
			owner, err = client.Resource(gvr).Namespace(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		}
		if err != nil {
			continue
		}

		// Add to relations
		*relations = append(*relations, ResourceRelation{
			Kind:      owner.GetKind(),
			Name:      owner.GetName(),
			Namespace: owner.GetNamespace(),
			UID:       ownerUID,
			Type:      "owner",
		})

		// Recursively find owners of this owner
		if err := s.findOwnerResources(ctx, client, owner, relations, visited, depth+1); err != nil {
			continue
		}
	}

	return nil
}

// getGVR returns the GroupVersionResource for a given kind
func (s *RelationsService) getGVR(kind string) (schema.GroupVersionResource, error) {
	// Map common kinds to GVR
	kindToGVR := map[string]schema.GroupVersionResource{
		"Pod":           {Group: "", Version: "v1", Resource: "pods"},
		"ReplicaSet":    {Group: "apps", Version: "v1", Resource: "replicasets"},
		"Deployment":    {Group: "apps", Version: "v1", Resource: "deployments"},
		"StatefulSet":   {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"DaemonSet":     {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"Job":           {Group: "batch", Version: "v1", Resource: "jobs"},
		"CronJob":       {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"Service":       {Group: "", Version: "v1", Resource: "services"},
		"Endpoints":     {Group: "", Version: "v1", Resource: "endpoints"},
		"EndpointSlice": {Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"},
		"Ingress":       {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	}

	gvr, exists := kindToGVR[kind]
	if !exists {
		return schema.GroupVersionResource{}, fmt.Errorf("unknown resource kind: %s", kind)
	}

	return gvr, nil
}

// getGVRFromAPIVersion returns the GroupVersionResource from APIVersion and Kind
func (s *RelationsService) getGVRFromAPIVersion(apiVersion, kind string) (schema.GroupVersionResource, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	// Convert kind to resource (simple pluralization)
	resource := s.kindToResource(kind)

	return schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource,
	}, nil
}

// kindToResource converts a kind to a resource name (simple pluralization)
func (s *RelationsService) kindToResource(kind string) string {
	// Common special cases
	specialCases := map[string]string{
		"Endpoints":     "endpoints",
		"Ingress":       "ingresses",
		"EndpointSlice": "endpointslices",
	}

	if resource, exists := specialCases[kind]; exists {
		return resource
	}

	// Simple pluralization (add 's')
	return kind + "s"
}
