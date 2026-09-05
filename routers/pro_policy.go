package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

// 多存储策略（PRO）：用户侧策略列表与目录偏好策略设置。
func init() {
	RegisterProPublicRoute(func(v4 *gin.RouterGroup, _ dependency.Dep) {
		// 当前用户组可用的存储策略列表
		v4.GET("user/policies", middleware.LoginRequired(), controllers.ProGetUserPolicies)
		// 设置目录/文件的偏好存储策略
		v4.PATCH(
			"file/policy",
			middleware.LoginRequired(),
			middleware.RequiredScopes(types.ScopeFilesWrite),
			controllers.FromJSON[explorer.SetFilePolicyService](explorer.SetFilePolicyParameterCtx{}),
			controllers.ProSetFilePolicy,
		)
	})
}
