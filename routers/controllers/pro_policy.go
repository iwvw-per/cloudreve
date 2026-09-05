package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

// ProGetUserPolicies 获取当前登录用户组可用的存储策略列表。
func ProGetUserPolicies(c *gin.Context) {
	res, err := explorer.UserPolicies(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// ProSetFilePolicy 设置目录/文件的偏好存储策略。
func ProSetFilePolicy(c *gin.Context) {
	service := ParametersFromContext[*explorer.SetFilePolicyService](c, explorer.SetFilePolicyParameterCtx{})
	res, err := service.Set(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}
