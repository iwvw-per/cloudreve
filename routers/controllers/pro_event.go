package controllers

import (
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
)

// AdminListEvents 操作日志列表
func AdminListEvents(c *gin.Context) {
	service := ParametersFromContext[*admin.AdminListService](c, admin.AdminListServiceParamsCtx{})
	res, err := service.Events(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// AdminGetEvent 操作日志详情
func AdminGetEvent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, serializer.Err(c, serializer.NewError(serializer.CodeParamErr, "Invalid event ID", err)))
		return
	}
	service := &admin.SingleAuditLogService{ID: id}
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// AdminCleanupEvent 清理操作日志
func AdminCleanupEvent(c *gin.Context) {
	service := ParametersFromContext[*admin.CleanupEventService](c, admin.CleanupEventParameterCtx{})
	err := service.Cleanup(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

// AdminBatchDeleteEvent 批量删除操作日志
func AdminBatchDeleteEvent(c *gin.Context) {
	service := ParametersFromContext[*admin.BatchEventService](c, admin.BatchEventParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}