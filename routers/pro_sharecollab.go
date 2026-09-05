package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	sharesvc "github.com/cloudreve/Cloudreve/v4/service/share"
	"github.com/gin-gonic/gin"
)

func init() {
	RegisterProPublicRoute(registerProPublicShareCollabRoutes)
}

// registerProPublicShareCollabRoutes 注册分享写协作与分享定价（积分购买）公开接口。
func registerProPublicShareCollabRoutes(v4 *gin.RouterGroup, _ dependency.Dep) {
	shareCollab := v4.Group("share")
	{
		// 经分享链接上传文件（匿名上传由 AllowAnonymousUpload 控制）
		shareCollab.POST("upload/:id",
			middleware.HashID(hashid.ShareID),
			controllers.FromQuery[explorer.ShareUploadService](explorer.ShareUploadParamCtx{}),
			controllers.ShareUpload,
		)
		// 经分享链接修改分享内文件（?new_name= 重命名 / 否则请求体为文件新内容）
		shareCollab.POST("modify/:id",
			middleware.LoginRequired(),
			middleware.HashID(hashid.ShareID),
			controllers.FromQuery[explorer.ShareModifyService](explorer.ShareModifyParamCtx{}),
			controllers.ShareModify,
		)
		// 经分享链接删除分享内文件
		shareCollab.DELETE("file/:id",
			middleware.LoginRequired(),
			middleware.HashID(hashid.ShareID),
			controllers.FromJSON[explorer.ShareDeleteService](explorer.ShareDeleteParamCtx{}),
			controllers.ShareDelete,
		)
		// 用积分购买付费分享访问权
		shareCollab.POST("buy/:id",
			middleware.LoginRequired(),
			middleware.HashID(hashid.ShareID),
			controllers.FromJSON[sharesvc.BuyShareService](sharesvc.BuyShareParamCtx{}),
			controllers.BuyShare,
		)
	}
}
