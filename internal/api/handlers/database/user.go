package database

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/api/middleware"
	dbservices "github.com/ysicing/tiga/internal/services/database"
)

// UserHandler manages database users.
type UserHandler struct {
	userService *dbservices.UserService
	audit       *dbservices.AuditLogger
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(userService *dbservices.UserService, audit *dbservices.AuditLogger) *UserHandler {
	return &UserHandler{
		userService: userService,
		audit:       audit,
	}
}

type createUserRequest struct {
	Username    string   `json:"username" binding:"required"`
	Password    string   `json:"password" binding:"required"`
	Host        string   `json:"host"`
	Description string   `json:"description"`
	Roles       []string `json:"roles"`
}

type updatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// ListUsers handles GET /api/v1/database/instances/{id}/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	users, err := h.userService.ListUsers(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"users": users,
		"count": len(users),
	})
}

// CreateUser handles POST /api/v1/database/instances/{id}/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var req createUserRequest
	if !handlers.BindJSON(c, &req) {
		return
	}

	record, err := h.userService.CreateUser(c.Request.Context(), instanceID, dbservices.CreateUserInput{
		Username:    req.Username,
		Password:    req.Password,
		Host:        req.Host,
		Description: req.Description,
		Roles:       req.Roles,
	})
	if err != nil {
		if errors.Is(err, dbservices.ErrOperationNotSupported) {
			handlers.RespondBadRequest(c, err)
		} else {
			handlers.RespondInternalError(c, err)
		}
		return
	}

	h.logAudit(c, dbservices.AuditEntry{
		InstanceID: &instanceID,
		Action:     "user.create",
		TargetType: "user",
		TargetName: req.Username,
		Details: map[string]interface{}{
			"host":  record.Host,
			"roles": req.Roles,
		},
		Success: true,
	})

	handlers.RespondCreated(c, record)
}

// UpdatePassword handles PATCH /api/v1/database/users/{id}
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	userID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var req updatePasswordRequest
	if !handlers.BindJSON(c, &req) {
		return
	}

	if err := h.userService.UpdatePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, dbservices.ErrOperationNotSupported) {
			handlers.RespondBadRequest(c, err)
		} else {
			handlers.RespondInternalError(c, err)
		}
		return
	}

	h.logAudit(c, dbservices.AuditEntry{
		Action:     "user.update",
		TargetType: "user",
		TargetName: userID.String(),
		Success:    true,
	})

	handlers.RespondSuccessWithMessage(c, gin.H{"user_id": userID}, "password updated")
}

// DeleteUser handles DELETE /api/v1/database/users/{id}
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	record, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, dbservices.ErrOperationNotSupported) {
			handlers.RespondBadRequest(c, err)
		} else {
			handlers.RespondInternalError(c, err)
		}
		return
	}

	h.logAudit(c, dbservices.AuditEntry{
		InstanceID: &record.InstanceID,
		Action:     "user.delete",
		TargetType: "user",
		TargetName: record.Username,
		Details: map[string]interface{}{
			"user_id": userID,
		},
		Success: true,
	})

	handlers.RespondNoContent(c)
}

func (h *UserHandler) logAudit(c *gin.Context, entry dbservices.AuditEntry) {
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
