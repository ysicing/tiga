package docker

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// ImageHandler handles Docker image operations
type ImageHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewImageHandler creates a new image handler
func NewImageHandler(instanceRepo repository.InstanceRepository) *ImageHandler {
	return &ImageHandler{
		instanceRepo: instanceRepo,
	}
}

// ListImages handles GET /api/v1/docker/instances/{id}/images
func (h *ImageHandler) ListImages(c *gin.Context) {
	instanceIDStr := c.Param("id")
	all := c.DefaultQuery("all", "false") == "true"

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

	if instance.Type != "docker" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not Docker type"))
		return
	}

	// Create Docker manager
	manager := managers.NewDockerManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// List images
	images, err := manager.ListImages(c.Request.Context(), all)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"images": images,
		"count":  len(images),
	})
}

// GetImage handles GET /api/v1/docker/instances/{id}/images/{image}
func (h *ImageHandler) GetImage(c *gin.Context) {
	instanceIDStr := c.Param("id")
	imageID := c.Param("image")

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

	if instance.Type != "docker" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not Docker type"))
		return
	}

	// Create Docker manager
	manager := managers.NewDockerManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Get image
	imageInspect, err := manager.GetImage(c.Request.Context(), imageID)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, imageInspect)
}

// PullImage handles POST /api/v1/docker/instances/{id}/images/pull
func (h *ImageHandler) PullImage(c *gin.Context) {
	instanceIDStr := c.Param("id")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var request struct {
		Image string `json:"image" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	if instance.Type != "docker" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not Docker type"))
		return
	}

	// Create Docker manager
	manager := managers.NewDockerManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Pull image
	reader, err := manager.PullImage(c.Request.Context(), request.Image)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer reader.Close()

	// Read pull progress (stream response)
	body, err := io.ReadAll(reader)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"image":  request.Image,
		"status": string(body),
	})
}

// DeleteImage handles DELETE /api/v1/docker/instances/{id}/images/{image}
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	instanceIDStr := c.Param("id")
	imageID := c.Param("image")
	force := c.DefaultQuery("force", "false") == "true"
	prune := c.DefaultQuery("prune", "false") == "true"

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

	if instance.Type != "docker" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not Docker type"))
		return
	}

	// Create Docker manager
	manager := managers.NewDockerManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Remove image
	responses, err := manager.RemoveImage(c.Request.Context(), imageID, force, prune)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"deleted": responses,
	})
}

// PruneImages handles POST /api/v1/docker/instances/{id}/images/prune
func (h *ImageHandler) PruneImages(c *gin.Context) {
	instanceIDStr := c.Param("id")
	all := c.DefaultQuery("all", "false") == "true"

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

	if instance.Type != "docker" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not Docker type"))
		return
	}

	// Create Docker manager
	manager := managers.NewDockerManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Prune images
	report, err := manager.PruneImages(c.Request.Context(), all)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"images_deleted":  report.ImagesDeleted,
		"space_reclaimed": report.SpaceReclaimed,
	})
}
