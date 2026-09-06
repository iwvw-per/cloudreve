package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	"github.com/gin-gonic/gin"
)

// 注册用户侧商城接口。
func init() {
	RegisterProPublicRoute(func(v4 *gin.RouterGroup, _ dependency.Dep) {
v4.GET("shop",
			middleware.LoginRequired(),
			controllers.ProUserGetShop,
		)
	})
}
