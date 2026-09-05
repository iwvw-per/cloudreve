package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	explorersvc "github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

func init() {
	RegisterProPublicRoute(func(v4 *gin.RouterGroup, _ dependency.Dep) {
		file := v4.Group("file")
		{
			file.PATCH("permission",
				middleware.LoginRequired(),
				middleware.RequiredScopes(types.ScopeFilesWrite),
				controllers.FromJSON[explorersvc.SetFilePermissionService](explorersvc.SetFilePermissionParamCtx{}),
				controllers.ProSetFilePermission,
			)
		}
	})
}
