package minio

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// BucketHandler handles MinIO bucket operations
type BucketHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewBucketHandler creates a new bucket handler
func NewBucketHandler(instanceRepo repository.InstanceRepository) *BucketHandler {
	return &BucketHandler{
		instanceRepo: instanceRepo,
	}
}

// ListBuckets handles GET /api/v1/minio/instances/{id}/buckets
func (h *BucketHandler) ListBuckets(c *gin.Context) {
	instanceIDStr := c.Param("id")

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

	// List buckets - get info and extract buckets
	info, err := manager.GetInfo(c.Request.Context())
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	buckets := info["buckets"]

	handlers.RespondSuccess(c, buckets)
}

// CreateBucket handles POST /api/v1/minio/instances/{id}/buckets
func (h *BucketHandler) CreateBucket(c *gin.Context) {
	instanceIDStr := c.Param("id")

	var request struct {
		Name     string `json:"name" binding:"required"`
		Location string `json:"location"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

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

	// Create bucket using minio-go client directly
	opts := minio.MakeBucketOptions{Region: request.Location}
	if err := manager.GetClient().MakeBucket(c.Request.Context(), request.Name, opts); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondCreated(c, gin.H{
		"name":     request.Name,
		"location": request.Location,
	})
}

// DeleteBucket handles DELETE /api/v1/minio/instances/{id}/buckets/{bucket}
func (h *BucketHandler) DeleteBucket(c *gin.Context) {
	instanceIDStr := c.Param("id")
	bucketName := c.Param("bucket")

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

	// Delete bucket
	if err := manager.GetClient().RemoveBucket(c.Request.Context(), bucketName); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"message": "Bucket deleted successfully",
		"bucket":  bucketName,
	})
}
