package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	"github.com/gin-gonic/gin"
)

func init() {
	RegisterProAdminRoute(registerProAdminEmailFilterRoutes)
}

// registerProAdminEmailFilterRoutes 注册邮箱域名过滤配置回读接口。
func registerProAdminEmailFilterRoutes(admin *gin.RouterGroup, _ dependency.Dep) {
	admin.GET("emailFilter",
		middleware.RequiredScopes(types.ScopeAdminRead),
		controllers.ProAdminGetEmailFilter,
	)
}