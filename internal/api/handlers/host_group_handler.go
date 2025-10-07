package handlers

import (
	"net/http"

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

// ListGroups lists all unique group names from host nodes
// @Summary List all host groups
// @Description Get a list of all unique group names used by host nodes
// @Tags VMs
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 500 {object} map[string]interface{} "失败"
// @Router /api/v1/vms/host-groups [get]
func (h *HostGroupHandler) ListGroups(c *gin.Context) {
	var groupNames []string

	// Query distinct group names from host_nodes, ordered alphabetically
	// Ensure "默认分组" appears first if it exists
	err := h.db.Model(&models.HostNode{}).
		Distinct("group_name").
		Where("group_name != ''").
		Order("CASE WHEN group_name = '默认分组' THEN 0 ELSE 1 END, group_name ASC").
		Pluck("group_name", &groupNames).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to list groups",
		})
		return
	}

	// If no groups exist, return default group
	if len(groupNames) == 0 {
		groupNames = []string{"默认分组"}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": groupNames,
			"total": len(groupNames),
		},
	})
}
