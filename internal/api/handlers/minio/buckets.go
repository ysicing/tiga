package minio

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"

	msvc "github.com/ysicing/tiga/internal/services/minio"
	pkmn "github.com/ysicing/tiga/pkg/minio"
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

	svc := msvc.NewBucketService(&h.instanceRepo)
	buckets, err := svc.ListBuckets(c.Request.Context(), instance.ID)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	// Convert to simple array for response
	out := make([]map[string]interface{}, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, map[string]interface{}{"name": b.Name, "created_at": b.CreationDate.Format("2006-01-02T15:04:05Z07:00")})
	}
	handlers.RespondSuccess(c, out)
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

	svc := msvc.NewBucketService(&h.instanceRepo)
	if err := svc.CreateBucket(c.Request.Context(), instance.ID, request.Name, request.Location); err != nil {
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

	svc := msvc.NewBucketService(&h.instanceRepo)
	if err := svc.DeleteBucket(c.Request.Context(), instance.ID, bucketName); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"message": "Bucket deleted successfully",
		"bucket":  bucketName,
	})
}

// GetBucket handles GET /api/v1/minio/instances/{id}/buckets/{bucket}
func (h *BucketHandler) GetBucket(c *gin.Context) {
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

	svc := msvc.NewBucketService(&h.instanceRepo)
	info, err := svc.GetBucketInfo(c.Request.Context(), instance.ID, bucketName)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	handlers.RespondSuccess(c, info)
}

// UpdateBucketPolicy handles PUT /api/v1/minio/instances/{id}/buckets/{bucket}/policy
func (h *BucketHandler) UpdateBucketPolicy(c *gin.Context) {
	instanceIDStr := c.Param("id")
	bucketName := c.Param("bucket")

	var request struct {
		Policy map[string]interface{} `json:"policy"`
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

	// Validate policy shape
	if err := pkmn.ValidatePolicy(request.Policy); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Marshal to JSON string and apply policy
	policyBytes, err := json.Marshal(request.Policy)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	svc := msvc.NewBucketService(&h.instanceRepo)
	if err := svc.UpdateBucketPolicy(c.Request.Context(), instance.ID, bucketName, string(policyBytes)); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{"message": "Bucket policy updated"})
}
