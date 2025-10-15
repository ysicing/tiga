package minio

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	mrepo "github.com/ysicing/tiga/internal/repository/minio"
	msvc "github.com/ysicing/tiga/internal/services/minio"
)

// UserHandler handles MinIO user operations
type UserHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewUserHandler creates a new user handler
func NewUserHandler(instanceRepo repository.InstanceRepository) *UserHandler {
	return &UserHandler{instanceRepo: instanceRepo}
}

// ListUsers handles GET /api/v1/minio/instances/{id}/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
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

	// Use service to list (server-authoritative)
	db := getDB(c)
	userRepo := mrepo.NewUserRepository(db)
	svc := msvc.NewUserService(&h.instanceRepo, userRepo)
	users, err := svc.ListUsers(c.Request.Context(), instance.ID)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, users)
}

type createUserRequest struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

// CreateUser handles POST /api/v1/minio/instances/{id}/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Generate credentials if not provided
	if req.AccessKey == "" {
		req.AccessKey = randHexUser(16)
	}
	if req.SecretKey == "" {
		req.SecretKey = randHexUser(24)
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
	userRepo := mrepo.NewUserRepository(db)
	svc := msvc.NewUserService(&h.instanceRepo, userRepo)
	if _, err := svc.CreateUser(c.Request.Context(), instance.ID, req.AccessKey, req.SecretKey, ""); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	// Audit
	logger := msvc.NewAuditLogger(mrepo.NewAuditRepository(db))
	_ = logger.LogOperation(c.Request.Context(), instance.ID, "user", "user", req.AccessKey, "create", "success", "", nil, "", c.ClientIP(), nil)
	handlers.RespondCreated(c, gin.H{"access_key": req.AccessKey, "secret_key": req.SecretKey})
}

// DeleteUser handles DELETE /api/v1/minio/instances/{id}/users/{username}
func (h *UserHandler) DeleteUser(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}
	username := c.Param("username")
	if username == "" {
		handlers.RespondBadRequest(c, fmt.Errorf("username is required"))
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
	userRepo := mrepo.NewUserRepository(db)
	svc := msvc.NewUserService(&h.instanceRepo, userRepo)
	if err := svc.DeleteUser(c.Request.Context(), instance.ID, username); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	// Audit
	logger := msvc.NewAuditLogger(mrepo.NewAuditRepository(db))
	_ = logger.LogOperation(c.Request.Context(), instance.ID, "user", "user", username, "delete", "success", "", nil, "", c.ClientIP(), nil)

	handlers.RespondSuccess(c, gin.H{"message": "user deleted"})
}

func randHexUser(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// helper to get *gorm.DB from gin context
// getDB provided by instance_handler.go in same package; no duplicate here
