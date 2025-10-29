package audit

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// EventHandler handles audit event query endpoints
// T023: Audit API handlers implementation
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T023
//
//	.claude/specs/006-gitness-tiga/contracts/audit_api.yaml
type EventHandler struct {
	eventRepo repository.AuditEventRepository
}

// NewEventHandler creates a new audit event handler
func NewEventHandler(eventRepo repository.AuditEventRepository) *EventHandler {
	return &EventHandler{
		eventRepo: eventRepo,
	}
}

// ListEvents godoc
// @Summary Get audit logs list
// @Description Get paginated audit logs with multi-dimensional filtering by user, resource type, action, time range, client IP, and request ID
// @Tags audit
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 20)" minimum(1) maximum(100)
// @Param user_uid query string false "Filter by user UID"
// @Param resource_type query string false "Filter by resource type" Enums(cluster, pod, deployment, service, database, databaseInstance, user, role, scheduledTask)
// @Param action query string false "Filter by action type" Enums(created, updated, deleted, read, enabled, disabled, login, logout, granted, revoked)
// @Param start_time query int false "Start time (Unix milliseconds)"
// @Param end_time query int false "End time (Unix milliseconds)"
// @Param client_ip query string false "Filter by client IP"
// @Param request_id query string false "Filter by request ID"
// @Success 200 {object} object{data=[]models.AuditEvent,pagination=handlers.Pagination}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /audit/events [get]
// @Security BearerAuth
func (h *EventHandler) ListEvents(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Build filters
	filters := make(map[string]interface{})

	if subsystem := c.Query("subsystem"); subsystem != "" {
		filters["subsystem"] = subsystem
	}

	if userUID := c.Query("user_uid"); userUID != "" {
		filters["user_uid"] = userUID
	}

	if resourceType := c.Query("resource_type"); resourceType != "" {
		// Validate resource type enum
		validResourceTypes := []models.ResourceType{
			models.ResourceTypeCluster,
			models.ResourceTypePod,
			models.ResourceTypeDeployment,
			models.ResourceTypeService,
			models.ResourceTypeConfigMap,
			models.ResourceTypeSecret,
			models.ResourceTypeDatabase,
			models.ResourceTypeDatabaseInstance,
			models.ResourceTypeDatabaseUser,
			models.ResourceTypeMinIO,
			models.ResourceTypeRedis,
			models.ResourceTypeMySQL,
			models.ResourceTypePostgreSQL,
			models.ResourceTypeUser,
			models.ResourceTypeRole,
			models.ResourceTypeInstance,
			models.ResourceTypeScheduledTask,
		}

		isValid := false
		for _, validType := range validResourceTypes {
			if string(validType) == resourceType {
				filters["resource_type"] = models.ResourceType(resourceType)
				isValid = true
				break
			}
		}

		if !isValid {
			err := fmt.Errorf("invalid resource_type: %s", resourceType)
			handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid resource_type value")
			return
		}
	}

	if action := c.Query("action"); action != "" {
		// Validate action enum
		validActions := []models.Action{
			models.ActionCreated,
			models.ActionUpdated,
			models.ActionDeleted,
			models.ActionRead,
			models.ActionEnabled,
			models.ActionDisabled,
			models.ActionBypassed,
			models.ActionForcePush,
			models.ActionLogin,
			models.ActionLogout,
			models.ActionGranted,
			models.ActionRevoked,
		}

		isValid := false
		for _, validAction := range validActions {
			if string(validAction) == action {
				filters["action"] = models.Action(action)
				isValid = true
				break
			}
		}

		if !isValid {
			err := fmt.Errorf("invalid action: %s", action)
			handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid action value")
			return
		}
	}

	// Parse time range filters
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTimeMs, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid start_time format")
			return
		}
		filters["start_time"] = startTimeMs
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTimeMs, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid end_time format")
			return
		}
		filters["end_time"] = endTimeMs
	}

	if clientIP := c.Query("client_ip"); clientIP != "" {
		filters["client_ip"] = clientIP
	}

	if requestID := c.Query("request_id"); requestID != "" {
		filters["request_id"] = requestID
	}

	// Query events
	events, err := h.eventRepo.List(c.Request.Context(), filters)
	if err != nil {
		logrus.Errorf("Failed to list audit events: %v", err)
		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to list audit events")
		return
	}

	// Apply manual pagination (since repository doesn't support it yet)
	// TODO: Move pagination to repository layer for better performance
	totalCount := len(events)
	start := offset
	end := offset + pageSize

	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}

	pagedEvents := events[start:end]

	// Build pagination metadata
	totalPages := (totalCount + pageSize - 1) / pageSize
	pagination := handlers.Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      int64(totalCount),
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       pagedEvents,
		"pagination": pagination,
	})
}

// GetEvent godoc
// @Summary Get audit event details
// @Description Get complete details of a single audit event including full object diff data
// @Tags audit
// @Accept json
// @Produce json
// @Param id path string true "Audit event ID (UUID)"
// @Success 200 {object} object{data=models.AuditEvent}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Router /audit/events/{id} [get]
// @Security BearerAuth
func (h *EventHandler) GetEvent(c *gin.Context) {
	eventID := c.Param("id")

	event, err := h.eventRepo.GetByID(c.Request.Context(), eventID)
	if err != nil {
		logrus.Errorf("Failed to get audit event %s: %v", eventID, err)
		handlers.RespondErrorWithMessage(c, http.StatusNotFound, err, "Audit event not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": event,
	})
}

// T029: ListK8sEvents retrieves K8s audit events
// @Summary List K8s audit events
// @Description Get paginated K8s audit events with cluster filtering
// @Tags audit
// @Accept json
// @Produce json
// @Param clusterName path string true "K8s cluster name"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 20)" minimum(1) maximum(100)
// @Success 200 {object} object{data=[]models.AuditEvent,pagination=handlers.Pagination}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /audit/events/k8s/{clusterName} [get]
// @Security BearerAuth
func (h *EventHandler) ListK8sEvents(c *gin.Context) {
	clusterName := c.Param("clusterName")

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Call repository method through type assertion
	// Using reflection to access unexported type
	auditEventRepoValue := reflect.ValueOf(h.eventRepo)
	if auditEventRepoValue.Kind() == reflect.Ptr && auditEventRepoValue.Elem().Kind() == reflect.Struct {
		// Get the concrete repository implementation
		auditEventRepo := auditEventRepoValue.Interface().(repository.AuditEventRepository)

		// Type assert to access unexported methods (we know the underlying type)
		impl, ok := auditEventRepo.((interface {
			ListK8sEvents(context.Context, string, int, int) ([]*models.AuditEvent, int64, error)
		}))
		if !ok {
			handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, fmt.Errorf("invalid repository type"), "Invalid repository type")
			return
		}

		events, total, err := impl.ListK8sEvents(c.Request.Context(), clusterName, pageSize, offset)
		if err != nil {
			logrus.Errorf("Failed to list K8s audit events: %v", err)
			handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to list K8s audit events")
			return
		}

		// Build pagination metadata
		totalPages := (int(total) + pageSize - 1) / pageSize
		pagination := handlers.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		}

		c.JSON(http.StatusOK, gin.H{
			"data":       events,
			"pagination": pagination,
		})
		return
	}

	handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, fmt.Errorf("invalid repository type"), "Invalid repository type")
}

// T029: GetK8sStatistics retrieves K8s audit statistics
// @Summary Get K8s audit statistics
// @Description Get aggregated statistics about K8s audit events
// @Tags audit
// @Accept json
// @Produce json
// @Param clusterName query string false "Filter by K8s cluster name"
// @Success 200 {object} object{data=repository.K8sAuditStatistics}
// @Failure 500 {object} handlers.ErrorResponse
// @Router /audit/events/k8s/statistics [get]
// @Security BearerAuth
func (h *EventHandler) GetK8sStatistics(c *gin.Context) {
	clusterName := c.Query("clusterName")

	// Call repository method through type assertion
	// Using reflection to access unexported type
	auditEventRepoValue := reflect.ValueOf(h.eventRepo)
	if auditEventRepoValue.Kind() == reflect.Ptr && auditEventRepoValue.Elem().Kind() == reflect.Struct {
		// Get the concrete repository implementation
		auditEventRepo := auditEventRepoValue.Interface().(repository.AuditEventRepository)

		// Type assert to access unexported methods (we know the underlying type)
		impl, ok := auditEventRepo.((interface {
			GetK8sStatistics(context.Context, string) (*repository.K8sAuditStatistics, error)
		}))
		if !ok {
			handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, fmt.Errorf("invalid repository type"), "Invalid repository type")
			return
		}

		stats, err := impl.GetK8sStatistics(c.Request.Context(), clusterName)
		if err != nil {
			logrus.Errorf("Failed to get K8s audit statistics: %v", err)
			handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to get K8s audit statistics")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": stats,
		})
		return
	}

	handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, fmt.Errorf("invalid repository type"), "Invalid repository type")
	return
}