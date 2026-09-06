package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	prosvc "github.com/cloudreve/Cloudreve/v4/service/pro"
	"github.com/gin-gonic/gin"
)

// ProUserGetShop 用户侧商城数据（商品列表 + 积分余额）。
func ProUserGetShop(c *gin.Context) {
	service := &prosvc.GetShopService{}
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}
