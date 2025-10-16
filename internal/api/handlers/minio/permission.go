package minio

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"

	mrepo "github.com/ysicing/tiga/internal/repository/minio"
	msvc "github.com/ysicing/tiga/internal/services/minio"
)

type PermissionHandler struct {
	instanceRepo repository.InstanceRepository
}

func NewPermissionHandler(instanceRepo repository.InstanceRepository) *PermissionHandler {
	return &PermissionHandler{instanceRepo: instanceRepo}
}

type GrantPermissionRequest struct {
	InstanceID string `json:"instance_id" binding:"required,uuid"`
	User       string `json:"user" binding:"required"`
	Bucket     string `json:"bucket" binding:"required"`
	Prefix     string `json:"prefix"`
	Permission string `json:"permission" binding:"required,oneof=readonly writeonly readwrite"`
}

// GrantPermission: POST /api/v1/minio/permissions
func (h *PermissionHandler) GrantPermission(c *gin.Context) {
	var req GrantPermissionRequest
	if !handlers.BindJSON(c, &req) {
		return
	}

	instanceID, err := handlers.ParseUUID(req.InstanceID)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}
	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	db := getDB(c)
	permSvc := msvc.NewPermissionService(&h.instanceRepo, mrepo.NewUserRepository(db), mrepo.NewPermissionRepository(db))
	var grantedBy *uuid.UUID
	// auth not wired; left nil
	policyName, err := permSvc.GrantPermission(c.Request.Context(), instance.ID, req.User, req.Bucket, req.Prefix, req.Permission, grantedBy, "")
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	// Audit
	logger := msvc.NewAuditLogger(mrepo.NewAuditRepository(db))
	_ = logger.LogOperation(c.Request.Context(), instance.ID, "permission", "bucket", req.Bucket, "grant", "success", "", nil, "", c.ClientIP(), models.JSONB{"user": req.User, "prefix": req.Prefix, "perm": req.Permission})
	handlers.RespondCreated(c, gin.H{"id": policyName, "instance_id": instance.ID.String(), "user": req.User, "bucket": req.Bucket, "prefix": req.Prefix, "permission": req.Permission})
}

// ListPermissions: GET /api/v1/minio/permissions?instance_id=...&user=...&bucket=...
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	instanceIDStr := c.Query("instance_id")
	if instanceIDStr == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance_id is required"))
		return
	}
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	user := c.Query("user")
	bucket := c.Query("bucket")

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}
	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	db := getDB(c)
	permSvc := msvc.NewPermissionService(&h.instanceRepo, mrepo.NewUserRepository(db), mrepo.NewPermissionRepository(db))
	var userID *uuid.UUID
	if user != "" {
		if u, err := mrepo.NewUserRepository(db).GetByUsername(c.Request.Context(), instance.ID, user); err == nil {
			userID = &u.ID
		}
	}
	perms, err := permSvc.ListPermissions(c.Request.Context(), instance.ID, userID, &bucket)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	handlers.RespondSuccess(c, perms)
}

// RevokePermission: DELETE /api/v1/minio/permissions/:id?instance_id=...&user=...
func (h *PermissionHandler) RevokePermission(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("id is required"))
		return
	}
	instanceIDStr := c.Query("instance_id")
	if instanceIDStr == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance_id is required"))
		return
	}
	user := c.Query("user")
	if user == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("user is required"))
		return
	}

	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}
	if instance.Type != "minio" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}

	db := getDB(c)
	permSvc := msvc.NewPermissionService(&h.instanceRepo, mrepo.NewUserRepository(db), mrepo.NewPermissionRepository(db))
	if err := permSvc.RevokePermission(c.Request.Context(), instance.ID, user, id); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	// Audit
	logger := msvc.NewAuditLogger(mrepo.NewAuditRepository(db))
	_ = logger.LogOperation(c.Request.Context(), instance.ID, "permission", "policy", id, "revoke", "success", "", nil, "", c.ClientIP(), models.JSONB{"user": user})
	handlers.RespondSuccess(c, gin.H{"message": "permission revoked"})
}
