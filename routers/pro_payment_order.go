package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	explorersvc "github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

// 注册用户侧支付下单路由。
func init() {
	RegisterProPublicRoute(func(v4 *gin.RouterGroup, _ dependency.Dep) {
		payment := v4.Group("payment")
		{
			// 创建订单
			payment.POST("order",
				middleware.LoginRequired(),
				middleware.RequiredScopes(types.ScopeSharesRead),
				controllers.FromJSON[explorersvc.CreateOrderService](explorersvc.CreateOrderParamCtx{}),
				controllers.CreateOrder,
			)
		}
	})
}
