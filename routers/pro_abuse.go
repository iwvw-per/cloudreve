package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	adminsvc "github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
)

func init() {
	RegisterProAdminRoute(registerProAdminAbuseRoutes)
	RegisterProPublicRoute(registerProPublicAbuseRoutes)
}

// registerProAdminAbuseRoutes 注册滥用举报管理后台路由。
func registerProAdminAbuseRoutes(admin *gin.RouterGroup, _ dependency.Dep) {
	abuse := admin.Group("abuse")
	{
		// 举报列表
		abuse.POST("",
			controllers.FromJSON[adminsvc.AdminListService](adminsvc.AdminListServiceParamsCtx{}),
			controllers.AdminListAbuseReports,
		)
		// 举报详情
		abuse.GET(":id",
			controllers.AdminGetAbuseReport,
		)
		// 标记举报处理
		abuse.PATCH(":id",
			middleware.RequiredScopes(types.ScopeAdminWrite),
			controllers.FromJSON[adminsvc.UpdateAbuseReportStatusService](adminsvc.UpdateAbuseReportStatusParamCtx{}),
			controllers.AdminUpdateAbuseReport,
		)
		// 批量删除举报
		abuse.POST("batch/delete",
			middleware.RequiredScopes(types.ScopeAdminWrite),
			controllers.FromJSON[adminsvc.BatchAbuseReportService](adminsvc.BatchAbuseReportParamCtx{}),
			controllers.AdminBatchDeleteAbuseReport,
		)
	}
}

// registerProPublicAbuseRoutes 注册用户侧举报分享的公开接口。
func registerProPublicAbuseRoutes(v4 *gin.RouterGroup, _ dependency.Dep) {
	v4.POST("share/report/:id",
		middleware.LoginRequired(),
		middleware.RequiredScopes(types.ScopeSharesRead),
		middleware.HashID(hashid.ShareID),
		controllers.FromJSON[adminsvc.CreateAbuseReportService](adminsvc.CreateAbuseReportParamCtx{}),
		controllers.ReportShareAbuse,
	)
}