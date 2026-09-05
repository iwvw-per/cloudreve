package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	"github.com/gin-gonic/gin"
)

// 多账号切换：为前端会话层提供无侵入的账号列表校验接口。
// 前端本地管理多个账号各自的 token，切换时仅替换 Authorization 头，
// 后端鉴权核心不受影响。此路由用于切换前/后校验前端上报的账号 token 并返回账号概要。
func init() {
	RegisterProPublicRoute(func(v4 *gin.RouterGroup, _ dependency.Dep) {
		accounts := v4.Group("session")
		{
			accounts.POST("accounts", controllers.ProListAccounts)
		}
	})
}
