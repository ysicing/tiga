package docker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/docker"
	"github.com/ysicing/tiga/internal/services/host"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ImageHandler handles Docker image API requests
type ImageHandler struct {
	imageService   *docker.ImageService
	agentForwarder *docker.AgentForwarderV2
	auditHelper    *docker.AuditHelper
}

// NewImageHandler creates a new ImageHandler
func NewImageHandler(
	imageService *docker.ImageService,
	agentForwarder *docker.AgentForwarderV2,
	auditHelper *docker.AuditHelper,
) *ImageHandler {
	return &ImageHandler{
		imageService:   imageService,
		agentForwarder: agentForwarder,
		auditHelper:    auditHelper,
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

	// Log audit (after operation)
	defer func() {
		count := 0
		if resp != nil {
			count = len(resp.Images)
		}
		h.auditHelper.LogListOperation(
			c,
			"list_images",
			"image",
			instanceID,
			"images",
			count,
			err,
		)
	}()

	if err != nil {
		logrus.WithError(err).WithField("instance_id", instanceID).Error("Failed to list images")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Return response with images array and total count
	basehandlers.RespondSuccess(c, map[string]interface{}{
		"images": resp.Images,
		"total":  len(resp.Images),
	})
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

	// Log audit (after operation)
	defer func() {
		h.auditHelper.LogGetOperation(
			c,
			"get_image",
			"image",
			instanceID,
			imageID,
			instanceID,
			"",
			err,
		)
	}()

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

	// Delete image through service layer (with unified audit logging)
	resp, err := h.imageService.DeleteImage(c, instanceID, req.ImageID, req.Force, req.NoPrune)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id": instanceID,
			"image_id":    req.ImageID,
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

	// Tag image through service layer (with unified audit logging)
	err = h.imageService.TagImage(c, instanceID, req.Source, req.Target)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id": instanceID,
			"source":      req.Source,
			"target":      req.Target,
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
	startTime := time.Now()
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req PullImageRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Start pull image stream
	sessionInterface, err := h.imageService.PullImage(c.Request.Context(), instanceID, req.Image, req.RegistryAuth)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id": instanceID,
			"image":       req.Image,
		}).Error("Failed to start image pull")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Type assert to *host.DockerStreamSession
	session, ok := sessionInterface.(*host.DockerStreamSession)
	if !ok {
		logrus.Error("Failed to cast session to *host.DockerStreamSession")
		basehandlers.RespondInternalError(c, fmt.Errorf("internal error: invalid session type"))
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

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
		"session_id":  session.SessionID,
	}).Info("Starting image pull stream")

	// Wait for Agent to connect and be ready (30 second timeout)
	if err := session.WaitForReady(30 * time.Second); err != nil {
		logrus.WithError(err).Error("Agent failed to connect")
		errorData := map[string]interface{}{
			"error": "Agent did not connect in time",
		}
		errorJSON, _ := json.Marshal(errorData)
		fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
		flusher.Flush()
		return
	}

	pullSuccess := false
	pullErrorMsg := ""

	// Helper function to log pull audit
	logPullAudit := func() {
		duration := time.Since(startTime).Milliseconds()
		var auditErr error
		if !pullSuccess && pullErrorMsg != "" {
			auditErr = fmt.Errorf("%s", pullErrorMsg)
		}

		h.auditHelper.LogDockerOperation(c, docker.AuditParams{
			Action:       models.DockerActionPullImage,
			ResourceType: models.DockerResourceTypeImage,
			ResourceID:   instanceID,
			ResourceName: req.Image,
			InstanceID:   instanceID,
			InstanceName: "",
			ExtraData: map[string]interface{}{
				"image":         req.Image,
				"registry_auth": req.RegistryAuth != "",
			},
			Error:    auditErr,
			Duration: duration,
		})
	}

	// Stream pull progress to client
	for {
		select {
		case <-c.Request.Context().Done():
			// Client disconnected
			logrus.WithFields(logrus.Fields{
				"instance_id": instanceID,
				"image":       req.Image,
			}).Info("Client disconnected from pull stream")

			// Create audit log for the pull operation
			logPullAudit()
			return

		case data, ok := <-session.DataChan:
			if !ok {
				// DataChan closed
				return
			}

			// Parse and forward progress data
			var progressData map[string]interface{}
			if err := json.Unmarshal(data.Data, &progressData); err != nil {
				logrus.WithError(err).Warn("Failed to parse progress data")
				continue
			}

			// Send progress as SSE event
			progressJSON, _ := json.Marshal(progressData)
			fmt.Fprintf(c.Writer, "data: %s\n\n", progressJSON)
			flusher.Flush()

		case streamErr, ok := <-session.ErrorChan:
			if !ok {
				return
			}

			// Stream error from Agent
			pullSuccess = false
			pullErrorMsg = streamErr.Error
			logrus.WithFields(logrus.Fields{
				"instance_id": instanceID,
				"image":       req.Image,
				"error":       pullErrorMsg,
			}).Error("Error during image pull")

			// Send error event
			errorData := map[string]interface{}{
				"error":   pullErrorMsg,
				"message": "Failed to pull image",
			}
			errorJSON, _ := json.Marshal(errorData)
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
			flusher.Flush()

			// Create audit log
			logPullAudit()
			return

		case closeMsg, ok := <-session.CloseChan:
			if !ok {
				return
			}

			// Stream completed normally
			pullSuccess = closeMsg.Reason == "completed"
			logrus.WithFields(logrus.Fields{
				"instance_id": instanceID,
				"image":       req.Image,
				"reason":      closeMsg.Reason,
			}).Info("Image pull stream closed")

			// Send completion event as data (not named event)
			completionData := map[string]interface{}{
				"status":  "completed",
				"message": "Image pulled successfully",
			}
			completionJSON, _ := json.Marshal(completionData)
			fmt.Fprintf(c.Writer, "data: %s\n\n", completionJSON)
			flusher.Flush()

			// Create audit log
			logPullAudit()
			return
		}
	}
}
