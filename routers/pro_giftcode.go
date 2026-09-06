package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	prosvc "github.com/cloudreve/Cloudreve/v4/service/pro"
	"github.com/gin-gonic/gin"
)

// 注册用户侧兑换码兑付路由。
func init() {
	RegisterProPublicRoute(func(v4 *gin.RouterGroup, _ dependency.Dep) {
		v4.POST("payment/redeem",
			middleware.LoginRequired(),
			middleware.RequiredScopes(types.ScopeSharesRead),
			controllers.FromJSON[prosvc.RedeemGiftCodeService](prosvc.RedeemGiftCodeParamCtx{}),
			controllers.RedeemGiftCode,
		)
	})
}
