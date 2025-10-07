package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/models"
)

var unlogPath = []string{
	"/healthz",
	"/assets/",
	"/favicon.ico",
	"/metrics",
}

func Logger() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Skip logging for health check and static asset paths
			for _, path := range unlogPath {
				if param.Path == path || strings.HasPrefix(param.Path, path) {
					return ""
				}
			}
			name := "-"
			if v, ok := param.Keys["user"]; ok {
				if user, ok := v.(models.User); ok {
					name = user.Key()
				}
			}

			cluster := "-"
			if clusterName, exists := param.Keys[ClusterNameKey]; exists {
				cluster = clusterName.(string)
			}

			return fmt.Sprintf("%s - %s \"%s %s\" %d %s %s %s\n",
				param.ClientIP,
				param.TimeStamp.Format("2006-01-02 15:04:05"),
				param.Method,
				param.Path,
				param.StatusCode,
				param.Latency,
				cluster,
				name,
			)
		},
	})
}
