package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	prosvc "github.com/cloudreve/Cloudreve/v4/service/pro"
	"github.com/gin-gonic/gin"
)

// RedeemGiftCode 用户侧兑换码兑付。
func RedeemGiftCode(c *gin.Context) {
	service := ParametersFromContext[*prosvc.RedeemGiftCodeService](c, prosvc.RedeemGiftCodeParamCtx{})
	res, err := service.Redeem(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}