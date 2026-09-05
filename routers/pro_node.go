package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	"github.com/gin-gonic/gin"
)

func init() {
	RegisterProPublicRoute(registerProPublicNodeRoutes)
}

// registerProPublicNodeRoutes 注册用户侧可用节点列表接口。
func registerProPublicNodeRoutes(v4 *gin.RouterGroup, _ dependency.Dep) {
	v4.GET("user/nodes",
		middleware.LoginRequired(),
		controllers.ProUserListNodes,
	)
}
