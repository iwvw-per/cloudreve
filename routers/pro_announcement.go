package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	prosvc "github.com/cloudreve/Cloudreve/v4/service/pro"
	"github.com/gin-gonic/gin"
)

// 站点公告管理后台与用户侧路由注册。
func init() {
	RegisterProAdminRoute(func(admin *gin.RouterGroup, _ dependency.Dep) {
		// 管理端读取站点公告
		admin.GET("announcement",
			controllers.FromUri[prosvc.GetAnnouncementService](prosvc.GetAnnouncementParamCtx{}),
			controllers.ProAdminGetAnnouncement,
		)
		// 管理端保存站点公告
		admin.PUT("announcement",
			middleware.RequiredScopes(types.ScopeAdminWrite),
			controllers.FromJSON[prosvc.UpdateAnnouncementService](prosvc.UpdateAnnouncementParamCtx{}),
			controllers.ProAdminUpdateAnnouncement,
		)
	})

	RegisterProPublicRoute(func(v4 *gin.RouterGroup, _ dependency.Dep) {
		// 用户侧读取站点公告
		v4.GET("site/announcement",
			controllers.ProPublicGetAnnouncement,
		)
	})
}