package minio

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"

	mrepo "github.com/ysicing/tiga/internal/repository/minio"
	msvc "github.com/ysicing/tiga/internal/services/minio"
)

type ShareHandler struct {
	instanceRepo repository.InstanceRepository
}

func NewShareHandler(instanceRepo repository.InstanceRepository) *ShareHandler {
	return &ShareHandler{instanceRepo: instanceRepo}
}

type createShareRequest struct {
	InstanceID string `json:"instance_id" binding:"required,uuid"`
	Bucket     string `json:"bucket" binding:"required"`
	Key        string `json:"key" binding:"required"`
	// expiry: one of 1h,1d,7d,30d
	Expiry string `json:"expiry" binding:"required,oneof=1h 1d 7d 30d"`
}

// CreateShare: POST /api/v1/minio/shares
func (h *ShareHandler) CreateShare(c *gin.Context) {
	var req createShareRequest
	if !handlers.BindJSON(c, &req) {
		return
	}
	instanceID, err := handlers.ParseUUID(req.InstanceID)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	d := time.Hour
	switch req.Expiry {
	case "1h":
		d = time.Hour
	case "1d":
		d = 24 * time.Hour
	case "7d":
		d = 7 * 24 * time.Hour
	case "30d":
		d = 30 * 24 * time.Hour
	}

	// Use ShareService to generate link and persist
	db := getDB(c)
	svc := msvc.NewShareService(&h.instanceRepo, mrepo.NewShareRepository(db))
	var createdBy *uuid.UUID
	link, url, err := svc.CreateShareLink(c.Request.Context(), instanceID, req.Bucket, req.Key, d, createdBy)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	// Audit
	logger := msvc.NewAuditLogger(mrepo.NewAuditRepository(getDB(c)))
	_ = logger.LogOperation(c.Request.Context(), instanceID, "share", "object", req.Key, "create", "success", "", nil, "", c.ClientIP(), models.JSONB{"bucket": req.Bucket, "expiry": req.Expiry})
	handlers.RespondCreated(c, gin.H{"id": link.ID, "instance_id": link.InstanceID, "bucket": link.BucketName, "key": link.ObjectKey, "url": url, "expires_at": link.ExpiresAt})
}

// ListShares: GET /api/v1/minio/shares
func (h *ShareHandler) ListShares(c *gin.Context) {
	// Optional instance_id filter to scope results
	instanceIDStr := c.Query("instance_id")
	db := getDB(c)
	repo := mrepo.NewShareRepository(db)
	if instanceIDStr == "" {
		items, err := repo.ListAll(c.Request.Context())
		if err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		handlers.RespondSuccess(c, items)
		return
	}
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}
	svc := msvc.NewShareService(&h.instanceRepo, repo)
	items, err := svc.ListShares(c.Request.Context(), instanceID, nil)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	handlers.RespondSuccess(c, items)
}

// RevokeShare: DELETE /api/v1/minio/shares/:id
func (h *ShareHandler) RevokeShare(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("id is required"))
		return
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}
	db := getDB(c)
	svc := msvc.NewShareService(&h.instanceRepo, mrepo.NewShareRepository(db))
	if err := svc.RevokeShare(c.Request.Context(), uid); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	// Audit
	logger := msvc.NewAuditLogger(mrepo.NewAuditRepository(getDB(c)))
	_ = logger.LogOperation(c.Request.Context(), uuid.Nil, "share", "share", id, "revoke", "success", "", nil, "", c.ClientIP(), nil)
	handlers.RespondSuccess(c, gin.H{"message": "share revoked"})
}
