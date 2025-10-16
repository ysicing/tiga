package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/config"
)

// ConfigMiddleware injects the application config into gin context
func ConfigMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	}
}

// GetConfig retrieves the config from gin context
func GetConfig(c *gin.Context) *config.Config {
	if cfg, exists := c.Get("config"); exists {
		if appCfg, ok := cfg.(*config.Config); ok {
			return appCfg
		}
	}
	return nil
}
