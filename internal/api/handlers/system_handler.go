package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// SystemHandler handles system configuration operations
type SystemHandler struct {
	db *gorm.DB
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(db *gorm.DB) *SystemHandler {
	return &SystemHandler{
		db: db,
	}
}

// getConfigValue retrieves a configuration value from system_configs table
func (h *SystemHandler) getConfigValue(key string) (interface{}, error) {
	var config models.SystemConfig
	if err := h.db.Where("key = ?", key).First(&config).Error; err != nil {
		return nil, err
	}

	// Extract value from JSONB
	if val, ok := config.Value["value"]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("value not found in config")
}

// setConfigValue sets a configuration value in system_configs table
func (h *SystemHandler) setConfigValue(key string, value interface{}, valueType string) error {
	var config models.SystemConfig
	err := h.db.Where("key = ?", key).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		// Create new config entry
		config = models.SystemConfig{
			Key:       key,
			Value:     models.JSONB{"value": value},
			ValueType: valueType,
		}
		return h.db.Create(&config).Error
	}

	if err != nil {
		return err
	}

	// Update existing entry
	config.Value = models.JSONB{"value": value}
	return h.db.Save(&config).Error
}

// GetPublicConfig godoc
// @Summary Get public system configuration
// @Description Get public system configuration (no auth required)
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} PublicConfigResponse
// @Router /api/system/config [get]
func (h *SystemHandler) GetPublicConfig(c *gin.Context) {
	appName, _ := h.getConfigValue("app_name")
	appSubtitle, _ := h.getConfigValue("app_subtitle")
	language, _ := h.getConfigValue("language")
	enableAnalytics, _ := h.getConfigValue("enable_analytics")

	response := PublicConfigResponse{
		AppName:         getString(appName, "Tiga"),
		AppSubtitle:     getString(appSubtitle, ""),
		Language:        getString(language, "zh-CN"),
		EnableAnalytics: getBool(enableAnalytics, false),
	}

	c.JSON(http.StatusOK, response)
}

// GetSystemConfig godoc
// @Summary Get full system configuration (admin only)
// @Description Get full system configuration including all settings
// @Tags system
// @Security Bearer
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/system/config/full [get]
func (h *SystemHandler) GetSystemConfig(c *gin.Context) {
	appName, _ := h.getConfigValue("app_name")
	appSubtitle, _ := h.getConfigValue("app_subtitle")
	domain, _ := h.getConfigValue("domain")
	httpPort, _ := h.getConfigValue("http_port")
	language, _ := h.getConfigValue("language")
	enableAnalytics, _ := h.getConfigValue("enable_analytics")
	installLock, _ := h.getConfigValue("install_lock")

	config := map[string]interface{}{
		"app_name":         getString(appName, "Tiga"),
		"app_subtitle":     getString(appSubtitle, ""),
		"domain":           getString(domain, ""),
		"http_port":        getInt(httpPort, 8080),
		"language":         getString(language, "zh-CN"),
		"enable_analytics": getBool(enableAnalytics, false),
		"install_lock":     getBool(installLock, false),
	}

	c.JSON(http.StatusOK, config)
}

// UpdateSystemConfig godoc
// @Summary Update system configuration (admin only)
// @Description Update system configuration settings
// @Tags system
// @Security Bearer
// @Accept json
// @Produce json
// @Param config body UpdateSystemConfigRequest true "System configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/system/config [put]
func (h *SystemHandler) UpdateSystemConfig(c *gin.Context) {
	var req UpdateSystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Update fields if provided
	if req.AppName != nil {
		if err := h.setConfigValue("app_name", *req.AppName, "string"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update app_name",
			})
			return
		}
	}
	if req.AppSubtitle != nil {
		if err := h.setConfigValue("app_subtitle", *req.AppSubtitle, "string"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update app_subtitle",
			})
			return
		}
	}
	if req.Language != nil {
		if err := h.setConfigValue("language", *req.Language, "string"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update language",
			})
			return
		}
	}
	if req.EnableAnalytics != nil {
		if err := h.setConfigValue("enable_analytics", *req.EnableAnalytics, "boolean"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update enable_analytics",
			})
			return
		}
	}

	// Return updated config
	h.GetSystemConfig(c)
}

// Helper functions to safely extract typed values
func getString(val interface{}, defaultVal string) string {
	if val == nil {
		return defaultVal
	}
	if str, ok := val.(string); ok {
		return str
	}
	return defaultVal
}

func getBool(val interface{}, defaultVal bool) bool {
	if val == nil {
		return defaultVal
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return defaultVal
}

func getInt(val interface{}, defaultVal int) int {
	if val == nil {
		return defaultVal
	}
	if i, ok := val.(float64); ok {
		return int(i)
	}
	if i, ok := val.(int); ok {
		return i
	}
	return defaultVal
}

// PublicConfigResponse represents public system configuration
type PublicConfigResponse struct {
	AppName         string `json:"app_name"`
	AppSubtitle     string `json:"app_subtitle"`
	Language        string `json:"language"`
	EnableAnalytics bool   `json:"enable_analytics"`
}

// UpdateSystemConfigRequest represents system configuration update request
type UpdateSystemConfigRequest struct {
	AppName         *string `json:"app_name,omitempty"`
	AppSubtitle     *string `json:"app_subtitle,omitempty"`
	Language        *string `json:"language,omitempty"`
	EnableAnalytics *bool   `json:"enable_analytics,omitempty"`
}
