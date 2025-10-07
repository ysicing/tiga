package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// AppConfigResponse 应用配置响应
type AppConfigResponse struct {
	AppName     string `json:"app_name"`
	AppSubtitle string `json:"app_subtitle"`
}

// ConfigFile 配置文件结构
type ConfigFile struct {
	Server struct {
		AppName     string `yaml:"app_name"`
		AppSubtitle string `yaml:"app_subtitle"`
	} `yaml:"server"`
}

// GetAppConfig 获取应用配置（登录页使用，无需认证）
func GetAppConfig(configPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 默认值
		response := AppConfigResponse{
			AppName:     "DevOps Platform",
			AppSubtitle: "Access your DevOps dashboard",
		}

		// 尝试读取配置文件
		data, err := os.ReadFile(configPath)
		if err == nil {
			var config ConfigFile
			if err := yaml.Unmarshal(data, &config); err == nil {
				if config.Server.AppName != "" {
					response.AppName = config.Server.AppName
				}
				if config.Server.AppSubtitle != "" {
					response.AppSubtitle = config.Server.AppSubtitle
				}
			}
		}

		c.JSON(http.StatusOK, response)
	}
}
