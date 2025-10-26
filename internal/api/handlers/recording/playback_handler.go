package recording

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services/recording"
)

// PlaybackHandler handles terminal recording playback and download requests
type PlaybackHandler struct {
	managerService *recording.ManagerService
}

// NewPlaybackHandler creates a new playback handler instance
func NewPlaybackHandler(managerService *recording.ManagerService) *PlaybackHandler {
	return &PlaybackHandler{
		managerService: managerService,
	}
}

// GetPlaybackContent retrieves recording content in Asciinema v2 format
// @Summary Get recording playback content
// @Description Get recording content in Asciinema v2 format for playback
// @Tags recordings
// @Accept json
// @Produce text/plain
// @Param id path string true "Recording ID (UUID)"
// @Success 200 {string} string "Asciinema v2 format content"
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse "Recording still in progress"
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings/{id}/playback [get]
// @Security BearerAuth
func (h *PlaybackHandler) GetPlaybackContent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get recording content reader
	reader, err := h.managerService.GetPlaybackContent(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "recording is still in progress" {
			c.JSON(http.StatusConflict, gin.H{"error": "Recording is still in progress"})
			return
		}
		if err.Error() == "recording not found: record not found" {
			handlers.RespondNotFound(c, err)
			return
		}
		handlers.RespondInternalError(c, err)
		return
	}
	defer reader.Close()

	// Stream content to client
	// Content-Type: text/plain for Asciinema format
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Copy content to response
	c.Status(http.StatusOK)
	if _, err := c.Writer.Write([]byte{}); err != nil {
		// Start streaming
		return
	}

	// Use DataFromReader for efficient streaming
	c.DataFromReader(http.StatusOK, -1, "text/plain", reader, nil)
}

// DownloadRecording provides recording file for download
// @Summary Download recording file
// @Description Download recording file as .cast file
// @Tags recordings
// @Accept json
// @Produce application/octet-stream
// @Param id path string true "Recording ID (UUID)"
// @Success 200 {file} binary "Recording .cast file"
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse "Recording still in progress"
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings/{id}/download [get]
// @Security BearerAuth
func (h *PlaybackHandler) DownloadRecording(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get recording file reader and filename
	reader, filename, err := h.managerService.DownloadRecording(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "recording is still in progress" {
			c.JSON(http.StatusConflict, gin.H{"error": "Recording is still in progress"})
			return
		}
		if err.Error() == "recording not found: record not found" {
			handlers.RespondNotFound(c, err)
			return
		}
		handlers.RespondInternalError(c, err)
		return
	}
	defer reader.Close()

	// Set headers for file download
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Stream file to client
	c.DataFromReader(http.StatusOK, -1, "application/octet-stream", reader, nil)
}
