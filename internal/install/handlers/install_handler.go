package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/install/models"
	"github.com/ysicing/tiga/internal/install/services"
)

// T025-T029: API 处理器

// InstallHandler 安装处理器
type InstallHandler struct {
	installService  *services.InstallService
	installComplete chan<- struct{}
}

// NewInstallHandler 创建安装处理器
func NewInstallHandler(configPath string, installComplete chan<- struct{}) *InstallHandler {
	return &InstallHandler{
		installService:  services.NewInstallService(configPath),
		installComplete: installComplete,
	}
}

// Status 检查初始化状态
// GET /api/install/status
func (h *InstallHandler) Status(c *gin.Context) {
	installed := h.installService.IsInstalled()

	redirectTo := "/install"
	if installed {
		redirectTo = "/login"
	}

	c.JSON(http.StatusOK, models.StatusResponse{
		Installed:  installed,
		RedirectTo: redirectTo,
	})
}

// CheckDB 检查数据库连接
// POST /api/install/check-db
func (h *InstallHandler) CheckDB(c *gin.Context) {
	var req models.CheckDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid database configuration",
			"details": err.Error(),
		})
		return
	}

	resp, err := h.installService.CheckDatabase(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid database configuration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ValidateAdmin 验证管理员账户
// POST /api/install/validate-admin
func (h *InstallHandler) ValidateAdmin(c *gin.Context) {
	var req models.AdminAccount
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid admin account data",
			"details": err.Error(),
		})
		return
	}

	resp := h.installService.ValidateAdmin(req)
	c.JSON(http.StatusOK, resp)
}

// ValidateSettings 验证系统设置
// POST /api/install/validate-settings
func (h *InstallHandler) ValidateSettings(c *gin.Context) {
	var req models.SystemSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid settings data",
			"details": err.Error(),
		})
		return
	}

	resp := h.installService.ValidateSettings(req)
	c.JSON(http.StatusOK, resp)
}

// Finalize 完成初始化
// POST /api/install/finalize
func (h *InstallHandler) Finalize(c *gin.Context) {
	// 检查是否已初始化
	if h.installService.IsInstalled() {
		c.JSON(http.StatusForbidden, models.FinalizeResponse{
			Success: false,
			Error:   "Installation already completed",
		})
		return
	}

	var req models.FinalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid finalize request",
			"details": err.Error(),
		})
		return
	}

	resp, err := h.installService.Finalize(req)
	if err != nil {
		// 根据错误类型返回不同状态码
		if err.Error() == "installation already completed" {
			c.JSON(http.StatusForbidden, resp)
			return
		}
		if err.Error() == "existing data found" {
			c.JSON(http.StatusConflict, resp)
			return
		}

		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	c.JSON(http.StatusOK, resp)

	// Signal installation completion (non-blocking, only if channel is provided)
	if h.installComplete != nil {
		go func() {
			h.installComplete <- struct{}{}
		}()
	}
}

// RegisterRoutes 注册路由
func (h *InstallHandler) RegisterRoutes(r *gin.RouterGroup) {
	install := r.Group("/install")
	{
		install.GET("/status", h.Status)
		install.POST("/check-db", h.CheckDB)
		install.POST("/validate-admin", h.ValidateAdmin)
		install.POST("/validate-settings", h.ValidateSettings)
		install.POST("/finalize", h.Finalize)
	}
}
