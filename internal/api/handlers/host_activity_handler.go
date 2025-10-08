package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"gorm.io/gorm"
)

// HostActivityHandler handles host activity log operations
type HostActivityHandler struct {
	db *gorm.DB
}

// NewHostActivityHandler creates a new host activity handler
func NewHostActivityHandler(db *gorm.DB) *HostActivityHandler {
	return &HostActivityHandler{db: db}
}

// ListActivities lists host activity logs with pagination and filtering
func (h *HostActivityHandler) ListActivities(c *gin.Context) {
	hostID := c.Param("id")

	// Parse query parameters
	page := 1
	pageSize := 20
	actionType := c.Query("action_type") // terminal, agent, system, user
	action := c.Query("action")           // Specific action

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	// Build query
	query := h.db.Model(&models.HostActivityLog{})

	if hostID != "" {
		hostUUID, err := uuid.Parse(hostID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid host ID"})
			return
		}
		query = query.Where("host_node_id = ?", hostUUID)
	}

	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated results
	var activities []models.HostActivityLog
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Preload("User").
		Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to fetch activities"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"activities":  activities,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// CreateActivity creates a new activity log (typically called internally)
func (h *HostActivityHandler) CreateActivity(c *gin.Context) {
	var req struct {
		HostNodeID  string  `json:"host_node_id" binding:"required"`
		UserID      *string `json:"user_id"`
		Action      string  `json:"action" binding:"required"`
		ActionType  string  `json:"action_type" binding:"required"`
		Description string  `json:"description"`
		Metadata    string  `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	hostUUID, err := uuid.Parse(req.HostNodeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid host ID"})
		return
	}

	activity := &models.HostActivityLog{
		HostNodeID:  hostUUID,
		Action:      req.Action,
		ActionType:  req.ActionType,
		Description: req.Description,
		Metadata:    req.Metadata,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	}

	if req.UserID != nil {
		userUUID, err := uuid.Parse(*req.UserID)
		if err == nil {
			activity.UserID = &userUUID
		}
	}

	if err := h.db.Create(activity).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create activity"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    activity,
	})
}

// LogActivity is a helper function to log activity from other handlers
func LogActivity(db *gorm.DB, hostID uuid.UUID, action, actionType, description string, userID *uuid.UUID, metadata string) error {
	activity := &models.HostActivityLog{
		HostNodeID:  hostID,
		Action:      action,
		ActionType:  actionType,
		Description: description,
		Metadata:    metadata,
		UserID:      userID,
	}

	return db.Create(activity).Error
}
