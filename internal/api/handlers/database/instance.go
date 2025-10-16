package database

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/api/middleware"

	dbservices "github.com/ysicing/tiga/internal/services/database"
)

// InstanceHandler exposes CRUD endpoints for managed database instances.
type InstanceHandler struct {
	manager *dbservices.DatabaseManager
	audit   *dbservices.AuditLogger
}

// NewInstanceHandler constructs an InstanceHandler.
func NewInstanceHandler(manager *dbservices.DatabaseManager, audit *dbservices.AuditLogger) *InstanceHandler {
	return &InstanceHandler{
		manager: manager,
		audit:   audit,
	}
}

type createInstanceRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port" binding:"required"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	SSLMode     string `json:"ssl_mode"`
	Description string `json:"description"`
}

// ListInstances handles GET /api/v1/database/instances
func (h *InstanceHandler) ListInstances(c *gin.Context) {
	instances, err := h.manager.ListInstances(c.Request.Context())
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"instances": instances,
		"count":     len(instances),
	})
}

// CreateInstance handles POST /api/v1/database/instances
func (h *InstanceHandler) CreateInstance(c *gin.Context) {
	var req createInstanceRequest
	if !handlers.BindJSON(c, &req) {
		return
	}

	if req.Port <= 0 {
		handlers.RespondBadRequest(c, fmt.Errorf("port must be greater than 0"))
		return
	}

	instance, err := h.manager.CreateInstance(c.Request.Context(), dbservices.CreateInstanceInput{
		Name:        req.Name,
		Type:        req.Type,
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		Password:    req.Password,
		SSLMode:     req.SSLMode,
		Description: req.Description,
	})
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	h.logAudit(c, dbservices.AuditEntry{
		InstanceID: &instance.ID,
		Action:     "instance.create",
		TargetType: "instance",
		TargetName: instance.Name,
		Details: map[string]interface{}{
			"type": req.Type,
			"host": req.Host,
			"port": req.Port,
		},
		Success: true,
	})

	handlers.RespondCreated(c, instance)
}

// GetInstance handles GET /api/v1/database/instances/{id}
func (h *InstanceHandler) GetInstance(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	instance, err := h.manager.GetInstance(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	handlers.RespondSuccess(c, instance)
}

// DeleteInstance handles DELETE /api/v1/database/instances/{id}
func (h *InstanceHandler) DeleteInstance(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	instance, err := h.manager.GetInstance(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	if err := h.manager.DeleteInstance(c.Request.Context(), instanceID); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	h.logAudit(c, dbservices.AuditEntry{
		InstanceID: &instanceID,
		Action:     "instance.delete",
		TargetType: "instance",
		TargetName: instance.Name,
		Success:    true,
	})

	handlers.RespondNoContent(c)
}

// TestConnection handles POST /api/v1/database/instances/{id}/test
func (h *InstanceHandler) TestConnection(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	err = h.manager.TestConnection(c.Request.Context(), instanceID)
	entry := dbservices.AuditEntry{
		InstanceID: &instanceID,
		Action:     "instance.test_connection",
		TargetType: "instance",
		TargetName: instanceID.String(),
		Success:    err == nil,
	}
	if err != nil {
		entry.Error = err
		h.logAudit(c, entry)
		handlers.RespondInternalError(c, err)
		return
	}

	h.logAudit(c, entry)
	handlers.RespondSuccessWithMessage(c, gin.H{"instance_id": instanceID}, "connection successful")
}

func (h *InstanceHandler) logAudit(c *gin.Context, entry dbservices.AuditEntry) {
	if h.audit == nil {
		return
	}

	if userID, err := middleware.GetUserID(c); err == nil {
		uid := userID.String()
		entry.Operator = uid
	} else {
		entry.Operator = "unknown"
	}

	entry.ClientIP = c.ClientIP()

	if err := h.audit.LogAction(c.Request.Context(), entry); err != nil {
		logrus.WithError(err).Warn("failed to write database audit log")
	}
}
