package database

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/api/middleware"

	dbservices "github.com/ysicing/tiga/internal/services/database"
)

// DatabaseHandler manages logical database operations under an instance.
type DatabaseHandler struct {
	service *dbservices.DatabaseService
	audit   *dbservices.AuditLogger
}

// NewDatabaseHandler constructs a DatabaseHandler.
func NewDatabaseHandler(service *dbservices.DatabaseService, audit *dbservices.AuditLogger) *DatabaseHandler {
	return &DatabaseHandler{
		service: service,
		audit:   audit,
	}
}

type createDatabaseRequest struct {
	Name      string `json:"name" binding:"required"`
	Charset   string `json:"charset"`
	Collation string `json:"collation"`
	Owner     string `json:"owner"`
}

// ListDatabases handles GET /api/v1/database/instances/{id}/databases
func (h *DatabaseHandler) ListDatabases(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	databases, err := h.service.ListDatabases(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"databases": databases,
		"count":     len(databases),
	})
}

// CreateDatabase handles POST /api/v1/database/instances/{id}/databases
func (h *DatabaseHandler) CreateDatabase(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var req createDatabaseRequest
	if !handlers.BindJSON(c, &req) {
		return
	}

	record, err := h.service.CreateDatabase(c.Request.Context(), instanceID, dbservices.CreateDatabaseInput{
		Name:      req.Name,
		Charset:   req.Charset,
		Collation: req.Collation,
		Owner:     req.Owner,
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
		Action:     "database.create",
		TargetType: "database",
		TargetName: req.Name,
		Details: map[string]interface{}{
			"charset":   req.Charset,
			"collation": req.Collation,
			"owner":     req.Owner,
		},
		Success: true,
	})

	handlers.RespondCreated(c, record)
}

// DeleteDatabase handles DELETE /api/v1/database/databases/{id}
func (h *DatabaseHandler) DeleteDatabase(c *gin.Context) {
	databaseID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	confirmName := c.Query("confirm_name")

	record, err := h.service.DeleteDatabase(c.Request.Context(), databaseID, confirmName)
	if err != nil {
		if errors.Is(err, dbservices.ErrOperationNotSupported) {
			handlers.RespondBadRequest(c, err)
		} else {
			handlers.RespondInternalError(c, err)
		}
		return
	}

	h.logAudit(c, dbservices.AuditEntry{
		InstanceID: &record.InstanceID,
		Action:     "database.delete",
		TargetType: "database",
		TargetName: record.Name,
		Details: map[string]interface{}{
			"database_id": record.ID,
		},
		Success: true,
	})

	handlers.RespondNoContent(c)
}

func (h *DatabaseHandler) logAudit(c *gin.Context, entry dbservices.AuditEntry) {
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
