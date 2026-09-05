package controllers

import (
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	adminsvc "github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
)

// AdminListAbuseReports 举报列表
func AdminListAbuseReports(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.AdminListService](c, adminsvc.AdminListServiceParamsCtx{})
	res, err := service.AbuseReports(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// AdminGetAbuseReport 举报详情
func AdminGetAbuseReport(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, serializer.Err(c, serializer.NewError(serializer.CodeParamErr, "Invalid abuse report ID", err)))
		return
	}
	service := &adminsvc.SingleAbuseReportService{ID: id}
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// AdminUpdateAbuseReport 标记举报处理
func AdminUpdateAbuseReport(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, serializer.Err(c, serializer.NewError(serializer.CodeParamErr, "Invalid abuse report ID", err)))
		return
	}
	service := ParametersFromContext[*adminsvc.UpdateAbuseReportStatusService](c, adminsvc.UpdateAbuseReportStatusParamCtx{})
	service.ID = id
	res, err := service.Update(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// AdminBatchDeleteAbuseReport 批量删除举报
func AdminBatchDeleteAbuseReport(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.BatchAbuseReportService](c, adminsvc.BatchAbuseReportParamCtx{})
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

// ReportShareAbuse 用户举报分享
func ReportShareAbuse(c *gin.Context) {
	service := ParametersFromContext[*adminsvc.CreateAbuseReportService](c, adminsvc.CreateAbuseReportParamCtx{})
	service.ShareID = hashid.FromContext(c)
	err := service.Create(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}