package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	adminsvc "github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
)

// 注册操作日志/审计日志相关管理后台路由。
func init() {
	RegisterProAdminRoute(func(admin *gin.RouterGroup, _ dependency.Dep) {
		event := admin.Group("event")
		{
			// 操作日志列表
			event.POST("",
				controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
				controllers.AdminListEvents,
			)
			// 操作日志详情
			event.GET(":id",
				controllers.AdminGetEvent,
			)
			// 清理操作日志
			event.POST("cleanup",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromJSON[adminsvc.CleanupEventService](adminsvc.CleanupEventParameterCtx{}),
				controllers.AdminCleanupEvent,
			)
			// 批量删除操作日志
			event.POST("batch/delete",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromJSON[adminsvc.BatchEventService](adminsvc.BatchEventParamCtx{}),
				controllers.AdminBatchDeleteEvent,
			)
		}
	})
}