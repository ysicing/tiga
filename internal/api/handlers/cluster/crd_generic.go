package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
)

// ListCRDResources godoc
// @Summary List CRD resources
// @Description List resources of a specific CRD type
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param group query string true "API group (e.g., apps.kruise.io)"
// @Param version query string true "API version (e.g., v1alpha1)"
// @Param resource query string true "Resource type (e.g., clonesets)"
// @Param namespace query string false "Namespace (default: all namespaces)"
// @Success 200 {object} map[string]interface{} "code=200, data={items:[], total:int}"
// @Failure 400 {object} map[string]interface{} "code=400, message=Missing required parameters"
// @Failure 404 {object} map[string]interface{} "code=404, message=CRD not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/crd-resources [get]
// @Security Bearer
func (h *ClusterHandler) ListCRDResources(c *gin.Context) {
	ctx := c.Request.Context()

	// Get cluster from context
	cluster, ok := middleware.GetClusterFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to get cluster from context",
		})
		return
	}

	// Get required query parameters
	group := c.Query("group")
	version := c.Query("version")
	resource := c.Query("resource")

	if group == "" || version == "" || resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Missing required parameters: group, version, resource",
		})
		return
	}

	namespace := c.Query("namespace")

	// Create K8s client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create generic CRD handler
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	handler := crd.NewGenericCRDHandler(client, gvr)

	// List resources
	list, err := handler.List(ctx, namespace)
	if err != nil {
		logrus.Errorf("Failed to list CRD resources %s: %v", resource, err)
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "CRD not found or not installed",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to list CRD resources",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "CRD resources retrieved successfully",
		"data": gin.H{
			"items": list.Items,
			"total": len(list.Items),
		},
	})
}

// GetCRDResource godoc
// @Summary Get a CRD resource
// @Description Get detailed information about a specific CRD resource
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param group query string true "API group"
// @Param version query string true "API version"
// @Param resource query string true "Resource type"
// @Param namespace query string true "Namespace"
// @Param name path string true "Resource name"
// @Success 200 {object} map[string]interface{} "code=200, data={resource}"
// @Failure 404 {object} map[string]interface{} "code=404, message=Resource not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/crd-resources/{name} [get]
// @Security Bearer
func (h *ClusterHandler) GetCRDResource(c *gin.Context) {
	ctx := c.Request.Context()

	// Get cluster from context
	cluster, ok := middleware.GetClusterFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to get cluster from context",
		})
		return
	}

	// Get parameters
	group := c.Query("group")
	version := c.Query("version")
	resource := c.Query("resource")
	namespace := c.Query("namespace")
	name := c.Param("name")

	if group == "" || version == "" || resource == "" || namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Missing required parameters: group, version, resource, namespace",
		})
		return
	}

	// Create K8s client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create generic CRD handler
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	handler := crd.NewGenericCRDHandler(client, gvr)

	// Get resource
	obj, err := handler.Get(ctx, namespace, name)
	if err != nil {
		logrus.Errorf("Failed to get CRD resource %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"message": "Resource not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Resource retrieved successfully",
		"data":    obj,
	})
}

// CreateCRDResource godoc
// @Summary Create a CRD resource
// @Description Create a new CRD resource
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param body body map[string]interface{} true "Resource definition"
// @Success 201 {object} map[string]interface{} "code=201, message=success"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid request"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/crd-resources [post]
// @Security Bearer
func (h *ClusterHandler) CreateCRDResource(c *gin.Context) {
	ctx := c.Request.Context()

	// Get cluster from context
	cluster, ok := middleware.GetClusterFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to get cluster from context",
		})
		return
	}

	// Parse request body
	var obj unstructured.Unstructured
	if err := c.ShouldBindJSON(&obj.Object); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid resource definition",
		})
		return
	}

	// Validate GVK
	gvk := obj.GroupVersionKind()
	if gvk.Group == "" || gvk.Version == "" || gvk.Kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Resource must have apiVersion and kind",
		})
		return
	}

	// Create K8s client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create generic CRD handler
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: gvk.Kind + "s", // Simple pluralization - may need improvement
	}
	handler := crd.NewGenericCRDHandler(client, gvr)

	// Create resource
	if err := handler.Create(ctx, &obj); err != nil {
		logrus.Errorf("Failed to create CRD resource: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to create resource",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    http.StatusCreated,
		"message": "Resource created successfully",
	})
}

// UpdateCRDResource godoc
// @Summary Update a CRD resource
// @Description Update an existing CRD resource
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param name path string true "Resource name"
// @Param body body map[string]interface{} true "Updated resource definition"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid request"
// @Failure 404 {object} map[string]interface{} "code=404, message=Resource not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/crd-resources/{name} [put]
// @Security Bearer
func (h *ClusterHandler) UpdateCRDResource(c *gin.Context) {
	ctx := c.Request.Context()

	// Get cluster from context
	cluster, ok := middleware.GetClusterFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to get cluster from context",
		})
		return
	}

	// Parse request body
	var obj unstructured.Unstructured
	if err := c.ShouldBindJSON(&obj.Object); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid resource definition",
		})
		return
	}

	// Validate GVK
	gvk := obj.GroupVersionKind()
	if gvk.Group == "" || gvk.Version == "" || gvk.Kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Resource must have apiVersion and kind",
		})
		return
	}

	// Create K8s client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create generic CRD handler
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: gvk.Kind + "s",
	}
	handler := crd.NewGenericCRDHandler(client, gvr)

	// Update resource
	if err := handler.Update(ctx, &obj); err != nil {
		logrus.Errorf("Failed to update CRD resource: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to update resource",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Resource updated successfully",
	})
}

// DeleteCRDResource godoc
// @Summary Delete a CRD resource
// @Description Delete a specific CRD resource
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param group query string true "API group"
// @Param version query string true "API version"
// @Param resource query string true "Resource type"
// @Param namespace query string true "Namespace"
// @Param name path string true "Resource name"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 404 {object} map[string]interface{} "code=404, message=Resource not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/crd-resources/{name} [delete]
// @Security Bearer
func (h *ClusterHandler) DeleteCRDResource(c *gin.Context) {
	ctx := c.Request.Context()

	// Get cluster from context
	cluster, ok := middleware.GetClusterFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to get cluster from context",
		})
		return
	}

	// Get parameters
	group := c.Query("group")
	version := c.Query("version")
	resource := c.Query("resource")
	namespace := c.Query("namespace")
	name := c.Param("name")

	if group == "" || version == "" || resource == "" || namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Missing required parameters: group, version, resource, namespace",
		})
		return
	}

	// Create K8s client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create generic CRD handler
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}
	handler := crd.NewGenericCRDHandler(client, gvr)

	// Delete resource
	if err := handler.Delete(ctx, namespace, name); err != nil {
		logrus.Errorf("Failed to delete CRD resource %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to delete resource",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Resource deleted successfully",
	})
}

// ApplyCRDResourceYAML godoc
// @Summary Apply CRD resource from YAML
// @Description Create or update a CRD resource from YAML definition
// @Tags k8s-crd
// @Accept plain
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param body body string true "YAML definition"
// @Success 200 {object} map[string]interface{} "code=200, message=success, data={action:'created'|'updated'}"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid YAML"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/crd-resources/apply [post]
// @Security Bearer
func (h *ClusterHandler) ApplyCRDResourceYAML(c *gin.Context) {
	ctx := c.Request.Context()

	// Get cluster from context
	cluster, ok := middleware.GetClusterFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to get cluster from context",
		})
		return
	}

	// Read YAML from request body
	yamlBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Failed to read request body",
		})
		return
	}

	// Parse YAML to unstructured object
	var obj unstructured.Unstructured
	if err := yaml.Unmarshal(yamlBytes, &obj.Object); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid YAML format",
		})
		return
	}

	// Validate GVK
	gvk := obj.GroupVersionKind()
	if gvk.Group == "" || gvk.Version == "" || gvk.Kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "YAML must have apiVersion and kind",
		})
		return
	}

	// Create K8s client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create generic CRD handler
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: gvk.Kind + "s", // Simple pluralization
	}
	handler := crd.NewGenericCRDHandler(client, gvr)

	// Try to get existing resource
	namespace := obj.GetNamespace()
	if namespace == "" {
		namespace = "default"
	}
	name := obj.GetName()

	action := "created"
	existing, err := handler.Get(ctx, namespace, name)
	if err == nil && existing != nil {
		// Resource exists, update it
		obj.SetResourceVersion(existing.GetResourceVersion())
		if err := handler.Update(ctx, &obj); err != nil {
			logrus.Errorf("Failed to update CRD resource: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "Failed to update resource",
			})
			return
		}
		action = "updated"
	} else {
		// Resource doesn't exist, create it
		if err := handler.Create(ctx, &obj); err != nil {
			logrus.Errorf("Failed to create CRD resource: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": "Failed to create resource",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Resource applied successfully",
		"data": gin.H{
			"action": action,
		},
	})
}
