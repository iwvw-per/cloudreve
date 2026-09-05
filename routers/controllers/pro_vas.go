package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	adminsvc "github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
)

// ProAdminListProduct 商品列表
func ProAdminListProduct(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.ListProductService](c, adminsvc.ListProductParamCtx{})
	res, err := service.List(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProAdminCreateProduct 新建商品
func ProAdminCreateProduct(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.UpsertProductService](c, adminsvc.UpsertProductParamCtx{})
	res, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProAdminUpdateProduct 更新商品
func ProAdminUpdateProduct(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.UpsertProductService](c, adminsvc.UpsertProductParamCtx{})
	res, err := service.Update(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProAdminGetProduct 商品详情
func ProAdminGetProduct(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.SingleProductService](c, adminsvc.SingleProductParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProAdminDeleteProduct 删除商品
func ProAdminDeleteProduct(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.SingleProductService](c, adminsvc.SingleProductParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

// ProAdminCreateGiftCode 批量生成兑换码
func ProAdminCreateGiftCode(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.CreateGiftCodeService](c, adminsvc.CreateGiftCodeParamCtx{})
	res, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProAdminListGiftCode 兑换码列表
func ProAdminListGiftCode(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.ListGiftCodeService](c, adminsvc.ListGiftCodeParamCtx{})
	res, err := service.List(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProAdminBatchDeleteGiftCode 批量删除兑换码
func ProAdminBatchDeleteGiftCode(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.BatchDeleteGiftCodeService](c, adminsvc.BatchDeleteGiftCodeParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

// ProAdminListOrder 订单列表
func ProAdminListOrder(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.ListOrderService](c, adminsvc.ListOrderParamCtx{})
	res, err := service.List(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProAdminDeleteOrder 删除订单
func ProAdminDeleteOrder(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.SingleOrderService](c, adminsvc.SingleOrderParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

// ProAdminCleanupOrder 清理订单
func ProAdminCleanupOrder(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.CleanupOrderService](c, adminsvc.CleanupOrderParamCtx{})
	err := service.Cleanup(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{})
}

// ProAdminAdjustCredit 调整用户积分
func ProAdminAdjustCredit(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.AdjustCreditService](c, adminsvc.AdjustCreditParamCtx{})
	res, err := service.Adjust(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}
