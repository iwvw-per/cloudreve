package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/gin-gonic/gin"
)

// proAdminRegistrar 注册 PRO 版管理后台扩展路由的回调。
type proAdminRegistrar func(admin *gin.RouterGroup, dep dependency.Dep)

var proAdminRegistrars []proAdminRegistrar

// RegisterProAdminRoute 供各 PRO 功能子代理在自己的文件中通过 init() 注册路由，
// 避免多个子代理同时编辑 router.go 或 registerProAdminRoutes 造成冲突。
func RegisterProAdminRoute(r proAdminRegistrar) {
	proAdminRegistrars = append(proAdminRegistrars, r)
}

// registerProAdminRoutes 在 admin 分组上执行所有 PRO 扩展路由注册。
func registerProAdminRoutes(admin *gin.RouterGroup, dep dependency.Dep) {
	for _, r := range proAdminRegistrars {
		r(admin, dep)
	}
}

// proPublicRegistrar 注册 PRO 版公开（非 admin）扩展路由的回调。
type proPublicRegistrar func(v4 *gin.RouterGroup, dep dependency.Dep)

var proPublicRegistrars []proPublicRegistrar

// RegisterProPublicRoute 供各 PRO 功能子代理在自己的文件中通过 init() 注册
// 需要公开访问（例如需要登录的用户侧接口）的路由，避免子代理并发编辑 router.go。
func RegisterProPublicRoute(r proPublicRegistrar) {
	proPublicRegistrars = append(proPublicRegistrars, r)
}

// registerProPublicRoutes 在 v4 分组上执行所有 PRO 公开扩展路由注册。
func registerProPublicRoutes(v4 *gin.RouterGroup, dep dependency.Dep) {
	for _, r := range proPublicRegistrars {
		r(v4, dep)
	}
}
