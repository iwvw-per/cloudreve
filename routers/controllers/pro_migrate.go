package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
)

// CreateRelocateTask 创建存储策略迁移任务
func CreateRelocateTask(c *gin.Context) {
	service := ParametersFromContext[*explorer.RelocateWorkflowService](c, explorer.CreateRelocateParamCtx{})
	resp, err := service.CreateRelocateTask(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		c.Abort()
		return
	}

	if resp != nil {
		c.JSON(200, serializer.Response{
			Data: resp,
		})
	}
}