package version

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/version"
)

// GetVersion 获取服务端版本信息
// @Summary 获取服务端版本信息
// @Description 返回服务端的版本号、构建时间和 commit ID
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} version.Info "版本信息"
// @Router /api/v1/version [get]
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, version.GetInfo())
}
