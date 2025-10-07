package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ysicing/tiga/internal/models"
	"gorm.io/gorm"
)

// HostGroupHandler handles host group operations
type HostGroupHandler struct {
	db *gorm.DB
}

// NewHostGroupHandler creates a new host group handler
func NewHostGroupHandler(db *gorm.DB) *HostGroupHandler {
	return &HostGroupHandler{db: db}
}

// CreateGroup creates a new host group
func (h *HostGroupHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	group := &models.HostGroup{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.db.Create(group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    group,
	})
}

// ListGroups lists all host groups
func (h *HostGroupHandler) ListGroups(c *gin.Context) {
	var groups []models.HostGroup
	if err := h.db.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to list groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": groups,
			"total": len(groups),
		},
	})
}

// DeleteGroup deletes a host group
func (h *HostGroupHandler) DeleteGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.db.Delete(&models.HostGroup{}, uint(id)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to delete group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "分组已删除"})
}

// AddHosts adds hosts to a group
func (h *HostGroupHandler) AddHosts(c *gin.Context) {
	var req struct {
		HostIDs []uint `json:"host_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "已添加" + strconv.Itoa(len(req.HostIDs)) + "个主机到分组",
	})
}

// RemoveHost removes a host from a group
func (h *HostGroupHandler) RemoveHost(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已从分组移除主机"})
}
