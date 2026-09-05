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
	RegisterProPublicRoute(registerProMigrateRoutes)
}

// registerProMigrateRoutes 注册存储策略迁移相关路由。
func registerProMigrateRoutes(v4 *gin.RouterGroup, _ dependency.Dep) {
	wf := v4.Group("workflow")
	wf.Use(middleware.LoginRequired())
	wf.Use(middleware.RequiredScopes(types.ScopeWorkflowRead))
	{
		// 创建存储策略迁移任务
		wf.POST("relocate",
			middleware.RequiredScopes(types.ScopeWorkflowWrite),
			controllers.FromJSON[explorersvc.RelocateWorkflowService](explorersvc.CreateRelocateParamCtx{}),
			controllers.CreateRelocateTask,
		)
	}
}
