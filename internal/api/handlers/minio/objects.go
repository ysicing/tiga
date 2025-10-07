package minio

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// ObjectHandler handles MinIO object operations
type ObjectHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewObjectHandler creates a new object handler
func NewObjectHandler(instanceRepo repository.InstanceRepository) *ObjectHandler {
	return &ObjectHandler{
		instanceRepo: instanceRepo,
	}
}

// ListObjects handles GET /api/v1/minio/instances/{id}/buckets/{bucket}/objects
func (h *ObjectHandler) ListObjects(c *gin.Context) {
	instanceIDStr := c.Param("id")
	bucketName := c.Param("bucket")
	prefix := c.DefaultQuery("prefix", "")
	recursive := c.DefaultQuery("recursive", "false") == "true"

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	// Create MinIO manager
	manager := managers.NewMinIOManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// List objects
	objectsCh := manager.GetClient().ListObjects(c.Request.Context(), bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	})

	objects := []map[string]interface{}{}
	for object := range objectsCh {
		if object.Err != nil {
			handlers.RespondInternalError(c, object.Err)
			return
		}

		objects = append(objects, map[string]interface{}{
			"key":           object.Key,
			"size":          object.Size,
			"etag":          object.ETag,
			"last_modified": object.LastModified,
			"storage_class": object.StorageClass,
			"is_dir":        object.Key[len(object.Key)-1] == '/',
		})
	}

	handlers.RespondSuccess(c, gin.H{
		"bucket":  bucketName,
		"prefix":  prefix,
		"objects": objects,
		"count":   len(objects),
	})
}

// GetObject handles GET /api/v1/minio/instances/{id}/buckets/{bucket}/objects/{object}
func (h *ObjectHandler) GetObject(c *gin.Context) {
	instanceIDStr := c.Param("id")
	bucketName := c.Param("bucket")
	objectName := c.Param("object")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	// Create MinIO manager
	manager := managers.NewMinIOManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Get object
	object, err := manager.GetClient().GetObject(c.Request.Context(), bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer object.Close()

	// Get object info
	stat, err := object.Stat()
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	// Set headers
	c.Header("Content-Type", stat.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
	c.Header("ETag", stat.ETag)
	c.Header("Last-Modified", stat.LastModified.Format(http.TimeFormat))

	// Stream object content
	c.DataFromReader(http.StatusOK, stat.Size, stat.ContentType, object, map[string]string{})
}

// UploadObject handles POST /api/v1/minio/instances/{id}/buckets/{bucket}/objects
func (h *ObjectHandler) UploadObject(c *gin.Context) {
	instanceIDStr := c.Param("id")
	bucketName := c.Param("bucket")
	objectName := c.Query("name")

	if objectName == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("object name is required"))
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}
	defer file.Close()

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	// Create MinIO manager
	manager := managers.NewMinIOManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Upload object
	info, err := manager.GetClient().PutObject(
		c.Request.Context(),
		bucketName,
		objectName,
		file,
		header.Size,
		minio.PutObjectOptions{
			ContentType: header.Header.Get("Content-Type"),
		},
	)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondCreated(c, gin.H{
		"bucket": bucketName,
		"key":    objectName,
		"size":   info.Size,
		"etag":   info.ETag,
	})
}

// DeleteObject handles DELETE /api/v1/minio/instances/{id}/buckets/{bucket}/objects/{object}
func (h *ObjectHandler) DeleteObject(c *gin.Context) {
	instanceIDStr := c.Param("id")
	bucketName := c.Param("bucket")
	objectName := c.Param("object")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	// Create MinIO manager
	manager := managers.NewMinIOManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Delete object
	if err := manager.GetClient().RemoveObject(c.Request.Context(), bucketName, objectName, minio.RemoveObjectOptions{}); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"message": "Object deleted successfully",
		"bucket":  bucketName,
		"object":  objectName,
	})
}
