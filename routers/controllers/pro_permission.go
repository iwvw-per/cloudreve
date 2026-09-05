package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	explorersvc "github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

func ProSetFilePermission(c *gin.Context) {
	service := ParametersFromContext[*explorersvc.SetFilePermissionService](c, explorersvc.SetFilePermissionParamCtx{})
	res, err := service.Set(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}
