package docker

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services/docker"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ImageHandler handles Docker image API requests
type ImageHandler struct {
	imageService   *docker.ImageService
	agentForwarder *docker.AgentForwarder
}

// NewImageHandler creates a new ImageHandler
func NewImageHandler(
	imageService *docker.ImageService,
	agentForwarder *docker.AgentForwarder,
) *ImageHandler {
	return &ImageHandler{
		imageService:   imageService,
		agentForwarder: agentForwarder,
	}
}

// GetImages godoc
// @Summary List Docker images
// @Description Get a list of Docker images for a specific instance with filtering
// @Tags docker-images
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param all query boolean false "Show all images (default: false, only show non-dangling images)"
// @Param filter query string false "Filter images (e.g., 'reference=nginx:*')"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/images [get]
// @Security BearerAuth
func (h *ImageHandler) GetImages(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	// Parse query parameters
	all := c.DefaultQuery("all", "false") == "true"
	filter := c.Query("filter")

	// Create images list request
	req := &pb.ListImagesRequest{
		All:     all,
		Filters: filter,
	}

	// Get images from agent
	resp, err := h.agentForwarder.ListImages(instanceID, req)
	if err != nil {
		logrus.WithError(err).WithField("instance_id", instanceID).Error("Failed to list images")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, resp.Images)
}

// GetImage godoc
// @Summary Get image details
// @Description Get detailed information about a specific Docker image
// @Tags docker-images
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param image_id path string true "Image ID or name:tag"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/images/{image_id} [get]
// @Security BearerAuth
func (h *ImageHandler) GetImage(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	imageID := c.Param("image_id")
	if imageID == "" {
		basehandlers.RespondBadRequest(c, fmt.Errorf("image_id is required"))
		return
	}

	// Create image inspect request
	req := &pb.GetImageRequest{
		ImageId: imageID,
	}

	// Get image details from agent
	resp, err := h.agentForwarder.GetImage(instanceID, req)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id": instanceID,
			"image_id":    imageID,
		}).Error("Failed to get image")
		basehandlers.RespondNotFound(c, err)
		return
	}

	basehandlers.RespondSuccess(c, resp.Image)
}

// DeleteImageRequest represents the request body for deleting an image
type DeleteImageRequest struct {
	ImageID string `json:"image_id" binding:"required"`
	Force   bool   `json:"force"`    // Force removal even if in use
	NoPrune bool   `json:"no_prune"` // Do not delete untagged parents
}

// DeleteImage godoc
// @Summary Delete Docker image
// @Description Delete a Docker image
// @Tags docker-images
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body DeleteImageRequest true "Delete image request"
// @Success 200 {object} handlers.SuccessResponse{data=object}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/images/delete [post]
// @Security BearerAuth
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req DeleteImageRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Delete image through service layer (with audit logging)
	resp, err := h.imageService.DeleteImage(c.Request.Context(), instanceID, req.ImageID, req.Force, req.NoPrune, userIDPtr, username, clientIP)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id": instanceID,
			"image_id":    req.ImageID,
			"user":        username,
		}).Error("Failed to delete image")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, map[string]interface{}{
		"message": "Image deleted successfully",
		"deleted": resp.Deleted,
	})
}

// TagImageRequest represents the request body for tagging an image
type TagImageRequest struct {
	Source string `json:"source" binding:"required"` // Source image (e.g., nginx:1.21)
	Target string `json:"target" binding:"required"` // Target tag (e.g., myregistry.com/nginx:latest)
}

// TagImage godoc
// @Summary Tag Docker image
// @Description Create a tag for a Docker image
// @Tags docker-images
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body TagImageRequest true "Tag image request"
// @Success 200 {object} handlers.SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/images/tag [post]
// @Security BearerAuth
func (h *ImageHandler) TagImage(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req TagImageRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Tag image through service layer (with audit logging)
	err = h.imageService.TagImage(c.Request.Context(), instanceID, req.Source, req.Target, userIDPtr, username, clientIP)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id": instanceID,
			"source":      req.Source,
			"target":      req.Target,
			"user":        username,
		}).Error("Failed to tag image")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, map[string]interface{}{
		"message": "Image tagged successfully",
		"source":  req.Source,
		"target":  req.Target,
	})
}

// PullImageRequest represents the request body for pulling an image
type PullImageRequest struct {
	Image        string `json:"image" binding:"required"` // Image name (e.g., nginx:1.21)
	RegistryAuth string `json:"registry_auth,omitempty"`  // Base64 encoded registry auth (optional)
}

// PullImage godoc
// @Summary Pull Docker image (streaming)
// @Description Pull a Docker image with real-time progress updates using Server-Sent Events (SSE)
// @Tags docker-images
// @Accept json
// @Produce text/event-stream
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body PullImageRequest true "Pull image request"
// @Success 200 {string} string "SSE stream of pull progress"
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/images/pull [post]
// @Security BearerAuth
func (h *ImageHandler) PullImage(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req PullImageRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Start pull image stream
	stream, err := h.imageService.PullImage(c.Request.Context(), instanceID, req.Image, req.RegistryAuth)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id": instanceID,
			"image":       req.Image,
			"user":        username,
		}).Error("Failed to start image pull")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create a channel to signal when to stop
	clientGone := c.Request.Context().Done()

	// Create flusher for immediate data sending
	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		logrus.Error("Streaming not supported")
		basehandlers.RespondInternalError(c, fmt.Errorf("streaming not supported"))
		return
	}

	logrus.WithFields(logrus.Fields{
		"instance_id": instanceID,
		"image":       req.Image,
		"user":        username,
	}).Info("Starting image pull stream")

	pullSuccess := false
	pullError := ""

	// Stream pull progress to client
	for {
		select {
		case <-clientGone:
			// Client disconnected
			logrus.WithFields(logrus.Fields{
				"instance_id": instanceID,
				"image":       req.Image,
			}).Info("Client disconnected from pull stream")

			// Create audit log for the pull operation
			_ = h.imageService.CreatePullAuditLog(c.Request.Context(), instanceID, req.Image, pullSuccess, pullError, userIDPtr, username, clientIP)
			return

		default:
			// Read progress from gRPC stream
			progress, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					// Stream ended normally
					pullSuccess = true
					logrus.WithFields(logrus.Fields{
						"instance_id": instanceID,
						"image":       req.Image,
					}).Info("Image pull completed")

					// Send completion event
					completionData := map[string]interface{}{
						"status":  "completed",
						"message": "Image pulled successfully",
					}
					completionJSON, _ := json.Marshal(completionData)
					fmt.Fprintf(c.Writer, "event: complete\ndata: %s\n\n", completionJSON)
					flusher.Flush()

					// Create audit log
					_ = h.imageService.CreatePullAuditLog(c.Request.Context(), instanceID, req.Image, pullSuccess, pullError, userIDPtr, username, clientIP)
					return
				}

				// Stream error
				pullSuccess = false
				pullError = err.Error()
				logrus.WithError(err).WithFields(logrus.Fields{
					"instance_id": instanceID,
					"image":       req.Image,
				}).Error("Error during image pull")

				// Send error event
				errorData := map[string]interface{}{
					"error":   err.Error(),
					"message": "Failed to pull image",
				}
				errorJSON, _ := json.Marshal(errorData)
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
				flusher.Flush()

				// Create audit log
				_ = h.imageService.CreatePullAuditLog(c.Request.Context(), instanceID, req.Image, pullSuccess, pullError, userIDPtr, username, clientIP)
				return
			}

			// Convert progress to JSON
			progressJSON, err := json.Marshal(progress)
			if err != nil {
				logrus.WithError(err).Error("Failed to marshal progress to JSON")
				continue
			}

			// Send progress as SSE event
			// Format: data: {json}\n\n
			fmt.Fprintf(c.Writer, "data: %s\n\n", progressJSON)
			flusher.Flush()
		}
	}
}
