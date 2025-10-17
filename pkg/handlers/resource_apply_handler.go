package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/pkg/cluster"
	"github.com/ysicing/tiga/pkg/common"
	"github.com/ysicing/tiga/pkg/rbac"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	syaml "sigs.k8s.io/yaml"
)

type ResourceApplyHandler struct {
}

func NewResourceApplyHandler() *ResourceApplyHandler {
	return &ResourceApplyHandler{}
}

type ApplyResourceRequest struct {
	YAML string `json:"yaml" binding:"required"`
}

// ApplyResource applies a YAML resource to the cluster
func (h *ResourceApplyHandler) ApplyResource(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(models.User)

	var req ApplyResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Decode YAML into unstructured object
	decodeUniversal := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}

	_, _, err := decodeUniversal.Decode([]byte(req.YAML), nil, obj)
	if err != nil {
		logrus.Errorf("Failed to decode YAML: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid YAML format: " + err.Error()})
		return
	}

	resource := strings.ToLower(obj.GetKind()) + "s"
	if !rbac.CanAccess(user, resource, "create", cs.Name, obj.GetNamespace()) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": rbac.NoAccess(user.Key(), string(common.VerbCreate), resource, obj.GetNamespace(), cs.Name)})
		return
	}

	ctx := c.Request.Context()

	existingObj := &unstructured.Unstructured{}
	existingObj.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	existingObj.SetName(obj.GetName())
	existingObj.SetNamespace(obj.GetNamespace())

	err = cs.K8sClient.Get(ctx, client.ObjectKey{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}, existingObj)

	defer func() {
		previousYAML := []byte{}
		if existingObj.GetResourceVersion() != "" {
			existingObj.SetManagedFields(nil)
			previousYAML, _ = syaml.Marshal(existingObj)
		}
		errMessage := ""
		if err != nil {
			errMessage = err.Error()
		}
		// TODO: User.ID is uint but ResourceHistory.OperatorID is UUID
		operatorUUID := uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("user-%d", user.ID)))
		models.DB.Create(&models.ResourceHistory{
			ClusterID:     cs.ClusterID,
			ResourceType:  resource,
			ResourceName:  obj.GetName(),
			Namespace:     obj.GetNamespace(),
			OperationType: "apply",
			ResourceYAML:  req.YAML,
			PreviousYAML:  string(previousYAML),
			OperatorID:    operatorUUID,
			OperatorName:  user.Username,
			Success:       err == nil,
			ErrorMessage:  errMessage,
		})
	}()

	switch {
	case apierrors.IsNotFound(err):
		if err := cs.K8sClient.Create(ctx, obj); err != nil {
			logrus.Errorf("Failed to create resource: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create resource: " + err.Error()})
			return
		}
	case err == nil:
		obj.SetResourceVersion(existingObj.GetResourceVersion())
		if err := cs.K8sClient.Update(ctx, obj); err != nil {
			logrus.Errorf("Failed to update resource: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update resource: " + err.Error()})
			return
		}
	default:
		logrus.Errorf("Failed to get resource: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get resource: " + err.Error()})
		return
	}

	logrus.Infof("Successfully applied resource: %s/%s", obj.GetKind(), obj.GetName())
	c.JSON(http.StatusOK, gin.H{
		"message":   "Resource applied successfully",
		"kind":      obj.GetKind(),
		"name":      obj.GetName(),
		"namespace": obj.GetNamespace(),
	})
}
