package version

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/config"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
	CommitID  = "unknown"
)

type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	CommitID  string `json:"commitId"`
	HasNew    bool   `json:"hasNewVersion"`
	Release   string `json:"releaseUrl"`
}

func GetVersion(c *gin.Context) {
	versionInfo := VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		CommitID:  CommitID,
	}

	// Get config from context
	disableVersionCheck := false
	if cfg, exists := c.Get("config"); exists {
		if appCfg, ok := cfg.(*config.Config); ok {
			disableVersionCheck = appCfg.Features.DisableVersionCheck
		}
	}

	if !disableVersionCheck {
		r := checkForUpdate(c.Request.Context(), Version)
		versionInfo.HasNew = r.hasNew
		if versionInfo.HasNew {
			versionInfo.Release = r.releaseURL
		}
	}
	c.JSON(http.StatusOK, versionInfo)
}
