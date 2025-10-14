package database

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/api/middleware"
	dbservices "github.com/ysicing/tiga/internal/services/database"
)

// PermissionHandler manages database permission policies.
type PermissionHandler struct {
	permissionService *dbservices.PermissionService
	audit             *dbservices.AuditLogger
}

// NewPermissionHandler constructs a PermissionHandler.
func NewPermissionHandler(permissionService *dbservices.PermissionService, audit *dbservices.AuditLogger) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
		audit:             audit,
	}
}

type grantPermissionRequest struct {
	UserID     uuid.UUID `json:"user_id" binding:"required"`
	DatabaseID uuid.UUID `json:"database_id" binding:"required"`
	Role       string    `json:"role" binding:"required"`
}

// GrantPermission handles POST /api/v1/database/permissions
func (h *PermissionHandler) GrantPermission(c *gin.Context) {
	var req grantPermissionRequest
	if !handlers.BindJSON(c, &req) {
		return
	}

	operatorID, err := middleware.GetUserID(c)
	if err != nil {
		handlers.RespondUnauthorized(c, err)
		return
	}

	policy, err := h.permissionService.GrantPermission(c.Request.Context(), req.UserID, req.DatabaseID, req.Role, operatorID.String())
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	h.logAudit(c, dbservices.AuditEntry{
		InstanceID: nil,
		Action:     "permission.grant",
		TargetType: "permission",
		TargetName: policy.ID.String(),
		Details: map[string]interface{}{
			"user_id":     req.UserID,
			"database_id": req.DatabaseID,
			"role":        req.Role,
		},
		Success: true,
	})

	handlers.RespondCreated(c, policy)
}

// RevokePermission handles DELETE /api/v1/database/permissions/{id}
func (h *PermissionHandler) RevokePermission(c *gin.Context) {
	permissionID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	if err := h.permissionService.RevokePermission(c.Request.Context(), permissionID); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	h.logAudit(c, dbservices.AuditEntry{
		Action:     "permission.revoke",
		TargetType: "permission",
		TargetName: permissionID.String(),
		Success:    true,
	})

	handlers.RespondNoContent(c)
}

// GetUserPermissions handles GET /api/v1/database/users/{id}/permissions
func (h *PermissionHandler) GetUserPermissions(c *gin.Context) {
	userID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	permissions, err := h.permissionService.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"permissions": permissions,
		"count":       len(permissions),
	})
}

func (h *PermissionHandler) logAudit(c *gin.Context, entry dbservices.AuditEntry) {
	if h.audit == nil {
		return
	}
	if userID, err := middleware.GetUserID(c); err == nil {
		entry.Operator = userID.String()
	} else {
		entry.Operator = "unknown"
	}
	entry.ClientIP = c.ClientIP()
	if err := h.audit.LogAction(c.Request.Context(), entry); err != nil {
		logrus.WithError(err).Warn("failed to write database audit log")
	}
}
