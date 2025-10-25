package docker

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// RecordingHandler handles terminal recording operations
type RecordingHandler struct {
	db            *gorm.DB
	recordingRepo repository.TerminalRecordingRepositoryInterface
}

// NewRecordingHandler creates a new RecordingHandler
func NewRecordingHandler(
	db *gorm.DB,
	recordingRepo repository.TerminalRecordingRepositoryInterface,
) *RecordingHandler {
	return &RecordingHandler{
		db:            db,
		recordingRepo: recordingRepo,
	}
}

// GetRecordings godoc
// @Summary Get terminal recordings list
// @Description Get a list of terminal recordings with filtering and pagination
// @Tags docker-recordings
// @Accept json
// @Produce json
// @Param user_id query string false "Filter by user ID"
// @Param instance_id query string false "Filter by Docker instance ID"
// @Param container_id query string false "Filter by container ID"
// @Param start_date query string false "Filter by start date (RFC3339)"
// @Param end_date query string false "Filter by end date (RFC3339)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param sort_by query string false "Sort field" default(started_at)
// @Param sort_order query string false "Sort order (ASC/DESC)" default(DESC)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/docker/recordings [get]
func (h *RecordingHandler) GetRecordings(c *gin.Context) {
	// Parse filter parameters
	filter := &repository.TerminalRecordingFilter{
		Limit:     20,
		Offset:    0,
		SortBy:    "started_at",
		SortOrder: "DESC",
	}

	// Parse user_id
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_USER_ID",
					"message": "Invalid user ID format",
				},
			})
			return
		}
		filter.UserID = &userID
	}

	// Parse instance_id
	if instanceIDStr := c.Query("instance_id"); instanceIDStr != "" {
		instanceID, err := uuid.Parse(instanceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_INSTANCE_ID",
					"message": "Invalid instance ID format",
				},
			})
			return
		}
		filter.InstanceID = &instanceID
	}

	// Parse container_id
	if containerID := c.Query("container_id"); containerID != "" {
		filter.ContainerID = &containerID
	}

	// Parse date filters
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_START_DATE",
					"message": "Invalid start date format (use RFC3339)",
				},
			})
			return
		}
		filter.StartDate = &startDate
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_END_DATE",
					"message": "Invalid end date format (use RFC3339)",
				},
			})
			return
		}
		filter.EndDate = &endDate
	}

	// Parse pagination
	if pageStr := c.Query("page"); pageStr != "" {
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		filter.Offset = (page - 1) * filter.Limit
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, _ := strconv.Atoi(pageSizeStr)
		if pageSize > 0 && pageSize <= 100 {
			filter.Limit = pageSize
		}
	}

	// Parse sorting
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filter.SortBy = sortBy
	}
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		filter.SortOrder = sortOrder
	}

	// Fetch recordings
	recordings, total, err := h.recordingRepo.List(c.Request.Context(), filter)
	if err != nil {
		logrus.WithError(err).Error("Failed to list recordings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to fetch recordings",
			},
		})
		return
	}

	// Convert to metadata format
	metadataList := make([]*models.RecordingMetadata, len(recordings))
	for i, recording := range recordings {
		metadataList[i] = recording.ToMetadata()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"recordings": metadataList,
			"pagination": gin.H{
				"total":     total,
				"page":      (filter.Offset / filter.Limit) + 1,
				"page_size": filter.Limit,
			},
		},
	})
}

// GetRecording godoc
// @Summary Get terminal recording details
// @Description Get details of a specific terminal recording
// @Tags docker-recordings
// @Accept json
// @Produce json
// @Param id path string true "Recording ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/docker/recordings/{id} [get]
func (h *RecordingHandler) GetRecording(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid recording ID format",
			},
		})
		return
	}

	recording, err := h.recordingRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Recording not found",
				},
			})
			return
		}

		logrus.WithError(err).Error("Failed to get recording")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to fetch recording",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    recording.ToMetadata(),
	})
}

// GetRecordingPlayback godoc
// @Summary Stream recording file for playback
// @Description Stream the asciinema recording file for browser playback
// @Tags docker-recordings
// @Accept json
// @Produce application/x-asciicast
// @Param id path string true "Recording ID"
// @Success 200 {file} file
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/docker/recordings/{id}/playback [get]
func (h *RecordingHandler) GetRecordingPlayback(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid recording ID format",
			},
		})
		return
	}

	recording, err := h.recordingRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Recording not found",
				},
			})
			return
		}

		logrus.WithError(err).Error("Failed to get recording")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to fetch recording",
			},
		})
		return
	}

	// Check if recording is completed
	if !recording.IsCompleted() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "RECORDING_NOT_COMPLETED",
				"message": "Recording is still in progress",
			},
		})
		return
	}

	// Check if file exists
	if _, err := os.Stat(recording.StoragePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FILE_NOT_FOUND",
				"message": "Recording file not found on disk",
			},
		})
		return
	}

	// Set proper headers for asciinema playback
	c.Header("Content-Type", "application/x-asciicast")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.cast\"", recording.ID))
	c.Header("Cache-Control", "public, max-age=3600")

	// Serve the file
	c.File(recording.StoragePath)
}

// DeleteRecording godoc
// @Summary Delete a terminal recording
// @Description Delete a terminal recording and its file
// @Tags docker-recordings
// @Accept json
// @Produce json
// @Param id path string true "Recording ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/docker/recordings/{id} [delete]
func (h *RecordingHandler) DeleteRecording(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid recording ID format",
			},
		})
		return
	}

	// Get recording to find file path
	recording, err := h.recordingRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Recording not found",
				},
			})
			return
		}

		logrus.WithError(err).Error("Failed to get recording")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to fetch recording",
			},
		})
		return
	}

	// Delete file from disk
	if recording.StoragePath != "" {
		if err := os.Remove(recording.StoragePath); err != nil && !os.IsNotExist(err) {
			logrus.WithError(err).Warn("Failed to delete recording file")
			// Continue anyway - delete from database
		}
	}

	// Delete from database
	if err := h.recordingRepo.Delete(c.Request.Context(), id); err != nil {
		logrus.WithError(err).Error("Failed to delete recording from database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete recording",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Recording deleted successfully",
	})
}

// GetRecordingStatistics godoc
// @Summary Get recording statistics
// @Description Get statistics about terminal recordings
// @Tags docker-recordings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/docker/recordings/statistics [get]
func (h *RecordingHandler) GetRecordingStatistics(c *gin.Context) {
	stats, err := h.recordingRepo.GetStatistics(c.Request.Context())
	if err != nil {
		logrus.WithError(err).Error("Failed to get recording statistics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to fetch statistics",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
