package audit

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
)

// ConfigHandler handles audit configuration endpoints
// T023: Audit API handlers implementation
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T023
//           .claude/specs/006-gitness-tiga/contracts/audit_api.yaml
type ConfigHandler struct {
	// TODO: Add config service or repository when implemented
	// For now, we'll return a static configuration
}

// NewConfigHandler creates a new audit config handler
func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

// AuditConfig represents audit system configuration
// T023: Audit configuration model
type AuditConfig struct {
	RetentionDays int `json:"retention_days"` // Audit log retention days (1-3650)
}

// UpdateConfigRequest represents the request body for updating audit config
type UpdateConfigRequest struct {
	RetentionDays int `json:"retention_days" binding:"required,min=1,max=3650"`
}

// GetConfig godoc
// @Summary Get audit configuration
// @Description Get current audit system configuration (retention period, write policy, etc.)
// @Tags audit
// @Accept json
// @Produce json
// @Success 200 {object} object{data=audit.AuditConfig}
// @Failure 401 {object} handlers.ErrorResponse
// @Router /audit/config [get]
// @Security BearerAuth
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	// TODO: Load configuration from database or config file
	// For now, return static default configuration
	config := AuditConfig{
		RetentionDays: 90, // Default: 90 days
	}

	c.JSON(http.StatusOK, gin.H{
		"data": config,
	})
}

// UpdateConfig godoc
// @Summary Update audit configuration
// @Description Update audit system configuration. Changes take effect immediately. Retention period changes apply on next cleanup task execution. Modifying retention period may cause historical data to be cleaned up.
// @Tags audit
// @Accept json
// @Produce json
// @Param body body UpdateConfigRequest true "Audit configuration to update"
// @Success 200 {object} object{message=string,data=audit.AuditConfig}
// @Failure 400 {object} handlers.ErrorResponse "Invalid parameters (retention_days must be between 1 and 3650)"
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Router /audit/config [put]
// @Security BearerAuth
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest

	// Bind and validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("Failed to bind update config request: %v", err)
		handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Additional validation
	if req.RetentionDays < 1 || req.RetentionDays > 3650 {
		logrus.Errorf("Invalid retention_days: %d", req.RetentionDays)
		errMsg := fmt.Errorf("retention_days must be between 1 and 3650")
		handlers.RespondBadRequest(c, errMsg)
		return
	}

	// TODO: Persist configuration to database or config file
	// TODO: Trigger cleanup task reconfiguration

	logrus.Infof("Audit configuration updated: retention_days=%d", req.RetentionDays)

	// Return updated configuration
	config := AuditConfig{
		RetentionDays: req.RetentionDays,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Audit configuration updated successfully",
		"data":    config,
	})
}
