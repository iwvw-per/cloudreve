package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	"github.com/gin-gonic/gin"
)

func init() {
	RegisterProPublicRoute(registerProPaymentRoutes)
}

// registerProPaymentRoutes 注册支付渠道异步通知回调路由（用户侧公开接口）。
func registerProPaymentRoutes(v4 *gin.RouterGroup, _ dependency.Dep) {
	payment := v4.Group("payment")
	{
		payment.POST("callback/alipay", controllers.ProAlipayCallback)
		payment.POST("callback/wechat", controllers.ProWechatCallback)
		payment.POST("callback/payjs", controllers.ProPayJSCallback)
		payment.POST("callback/custom", controllers.ProCustomCallback)
	}
}
