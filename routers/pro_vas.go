package routers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/middleware"
	"github.com/cloudreve/Cloudreve/v4/routers/controllers"
	adminsvc "github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
)

func init() {
	RegisterProAdminRoute(func(admin *gin.RouterGroup, _ dependency.Dep) {
		// 商品管理
		product := admin.Group("product")
		{
			// 商品列表
			product.POST("",
				controllers.FromJSON[adminsvc.ListProductService](adminsvc.ListProductParamCtx{}),
				controllers.ProAdminListProduct,
			)
			// 新建商品
			product.PUT("",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromJSON[adminsvc.UpsertProductService](adminsvc.UpsertProductParamCtx{}),
				controllers.ProAdminCreateProduct,
			)
			// 商品详情
			product.GET(":id",
				controllers.FromUri[adminsvc.SingleProductService](adminsvc.SingleProductParamCtx{}),
				controllers.ProAdminGetProduct,
			)
			// 更新商品
			product.PUT(":id",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromJSON[adminsvc.UpsertProductService](adminsvc.UpsertProductParamCtx{}),
				controllers.ProAdminUpdateProduct,
			)
			// 删除商品
			product.DELETE(":id",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromUri[adminsvc.SingleProductService](adminsvc.SingleProductParamCtx{}),
				controllers.ProAdminDeleteProduct,
			)
		}

		// 兑换码管理
		giftcode := admin.Group("giftcode")
		{
			// 批量生成兑换码
			giftcode.POST("",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromJSON[adminsvc.CreateGiftCodeService](adminsvc.CreateGiftCodeParamCtx{}),
				controllers.ProAdminCreateGiftCode,
			)
			// 兑换码列表
			giftcode.POST("list",
				controllers.FromJSON[adminsvc.ListGiftCodeService](adminsvc.ListGiftCodeParamCtx{}),
				controllers.ProAdminListGiftCode,
			)
			// 批量删除兑换码
			giftcode.POST("batch/delete",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromJSON[adminsvc.BatchDeleteGiftCodeService](adminsvc.BatchDeleteGiftCodeParamCtx{}),
				controllers.ProAdminBatchDeleteGiftCode,
			)
		}

		// 订单（支付）管理
		payment := admin.Group("payment")
		{
			// 订单列表
			payment.POST("",
				controllers.FromJSON[adminsvc.ListOrderService](adminsvc.ListOrderParamCtx{}),
				controllers.ProAdminListOrder,
			)
			// 删除订单
			payment.DELETE(":id",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromUri[adminsvc.SingleOrderService](adminsvc.SingleOrderParamCtx{}),
				controllers.ProAdminDeleteOrder,
			)
			// 清理订单
			payment.POST("cleanup",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromJSON[adminsvc.CleanupOrderService](adminsvc.CleanupOrderParamCtx{}),
				controllers.ProAdminCleanupOrder,
			)
		}

		// 用户积分管理
		user := admin.Group("user")
		{
			// 调整用户积分
			user.POST(":id/credits",
				middleware.RequiredScopes(types.ScopeAdminWrite),
				controllers.FromJSON[adminsvc.AdjustCreditService](adminsvc.AdjustCreditParamCtx{}),
				controllers.ProAdminAdjustCredit,
			)
		}
	})
}
