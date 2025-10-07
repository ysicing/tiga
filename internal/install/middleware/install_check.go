package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/install/services"
)

// T030: 初始化检查中间件

// InstallCheckMiddleware 检查系统是否已初始化的中间件
func InstallCheckMiddleware(configPath string) gin.HandlerFunc {
	configService := services.NewConfigService(configPath)

	return func(c *gin.Context) {
		// 允许的路径（不需要初始化检查）
		allowedPaths := map[string]bool{
			"/api/install/status":            true,
			"/api/install/check-db":          true,
			"/api/install/validate-admin":    true,
			"/api/install/validate-settings": true,
			"/api/install/finalize":          true,
			"/install":                       true,
			"/static":                        true,
			"/assets":                        true,
		}

		// 检查当前路径是否在允许列表中
		path := c.Request.URL.Path
		for allowedPath := range allowedPaths {
			if path == allowedPath || len(path) > len(allowedPath) && path[:len(allowedPath)] == allowedPath {
				c.Next()
				return
			}
		}

		// 检查是否已初始化
		if !configService.IsInstalled() {
			// 未初始化，重定向到安装页面
			if c.Request.Header.Get("Accept") == "application/json" {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":       "System not initialized",
					"redirect_to": "/install",
				})
			} else {
				c.Redirect(http.StatusTemporaryRedirect, "/install")
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

// PreventReinstallMiddleware 防止重复初始化的中间件
func PreventReinstallMiddleware(configPath string) gin.HandlerFunc {
	configService := services.NewConfigService(configPath)

	return func(c *gin.Context) {
		// 只在初始化相关路径生效（除了 status 检查）
		path := c.Request.URL.Path
		if path != "/api/install/status" && path != "/install" {
			if configService.IsInstalled() {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Installation already completed",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
