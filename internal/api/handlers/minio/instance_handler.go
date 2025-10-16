package minio

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
)

type InstanceHandler struct {
	instanceRepo repository.InstanceRepository
}

func NewMinioInstanceHandler(instanceRepo repository.InstanceRepository) *InstanceHandler {
	return &InstanceHandler{instanceRepo: instanceRepo}
}

// Create
type createReq struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Host        string                 `json:"host" binding:"required"`
	Port        int                    `json:"port" binding:"required"`
	UseSSL      bool                   `json:"use_ssl"`
	AccessKey   string                 `json:"access_key" binding:"required"`
	SecretKey   string                 `json:"secret_key" binding:"required"`
	OwnerID     string                 `json:"owner_id" binding:"required,uuid"`
	Labels      map[string]interface{} `json:"labels"`
}

func (h *InstanceHandler) Create(c *gin.Context) {
	var req createReq
	if !basehandlers.BindJSON(c, &req) {
		return
	}
	ownerID, err := basehandlers.ParseUUID(req.OwnerID)
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	instance := &models.Instance{
		Name:        req.Name,
		Description: req.Description,
		Type:        "minio",
		OwnerID:     ownerID,
		Connection:  models.JSONB{"host": req.Host, "port": req.Port},
		Config:      models.JSONB{"access_key": req.AccessKey, "secret_key": req.SecretKey, "use_ssl": req.UseSSL},
		Labels:      req.Labels,
		Status:      "pending",
		Health:      "unknown",
	}
	if err := h.instanceRepo.Create(c.Request.Context(), instance); err != nil {
		basehandlers.RespondInternalError(c, err)
		return
	}
	basehandlers.RespondCreated(c, instance)
}

// List
func (h *InstanceHandler) List(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		basehandlers.RespondInternalError(c, fmt.Errorf("db not found in context"))
		return
	}
	var instances []models.Instance
	if err := db.Where("type = ?", "minio").Order("created_at DESC").Find(&instances).Error; err != nil {
		basehandlers.RespondInternalError(c, err)
		return
	}
	basehandlers.RespondSuccess(c, instances)
}

// Get
func (h *InstanceHandler) Get(c *gin.Context) {
	id, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}
	inst, err := h.instanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		basehandlers.RespondNotFound(c, err)
		return
	}
	basehandlers.RespondSuccess(c, inst)
}

// Update (only description and labels for now)
type updateReq struct {
	Description *string                `json:"description"`
	Labels      map[string]interface{} `json:"labels"`
}

func (h *InstanceHandler) Update(c *gin.Context) {
	id, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}
	var req updateReq
	if !basehandlers.BindJSON(c, &req) {
		return
	}
	fields := map[string]interface{}{}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Labels != nil {
		fields["labels"] = req.Labels
	}
	if len(fields) > 0 {
		if err := h.instanceRepo.UpdateFields(c.Request.Context(), id, fields); err != nil {
			basehandlers.RespondInternalError(c, err)
			return
		}
	}
	inst, err := h.instanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		basehandlers.RespondNotFound(c, err)
		return
	}
	basehandlers.RespondSuccess(c, inst)
}

// Delete
func (h *InstanceHandler) Delete(c *gin.Context) {
	id, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}
	if err := h.instanceRepo.Delete(c.Request.Context(), id); err != nil {
		basehandlers.RespondNotFound(c, err)
		return
	}
	basehandlers.RespondNoContent(c)
}

// Test connection
func (h *InstanceHandler) Test(c *gin.Context) {
	id, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}
	inst, err := h.instanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		basehandlers.RespondNotFound(c, err)
		return
	}
	if inst.Type != "minio" {
		basehandlers.RespondBadRequest(c, fmt.Errorf("instance is not MinIO type"))
		return
	}
	mgr := managers.NewMinIOManager()
	if err := mgr.Initialize(c.Request.Context(), inst); err != nil {
		basehandlers.RespondInternalError(c, err)
		return
	}
	if err := mgr.Connect(c.Request.Context()); err != nil {
		basehandlers.RespondInternalError(c, err)
		return
	}
	defer mgr.Disconnect(c.Request.Context())
	status, _ := mgr.HealthCheck(c.Request.Context())
	basehandlers.RespondSuccess(c, status)
}

func getDB(c *gin.Context) *gorm.DB {
	v, ok := c.Get("db")
	if !ok {
		return nil
	}
	db, _ := v.(*gorm.DB)
	return db
}
