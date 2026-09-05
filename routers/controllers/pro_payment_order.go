package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	explorersvc "github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

// CreateOrder 创建一笔待支付订单
func CreateOrder(c *gin.Context) {
	service := ParametersFromContext[*explorersvc.CreateOrderService](c, explorersvc.CreateOrderParamCtx{})
	res, err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}
