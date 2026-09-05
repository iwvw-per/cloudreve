package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	prosvc "github.com/cloudreve/Cloudreve/v4/service/pro"
	"github.com/gin-gonic/gin"
)

// ProAdminGetAnnouncement 管理端读取站点公告。
func ProAdminGetAnnouncement(c *gin.Context) {
	service := ParametersFromContext[*prosvc.GetAnnouncementService](c, prosvc.GetAnnouncementParamCtx{})
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProAdminUpdateAnnouncement 管理端保存站点公告。
func ProAdminUpdateAnnouncement(c *gin.Context) {
	service := ParametersFromContext[*prosvc.UpdateAnnouncementService](c, prosvc.UpdateAnnouncementParamCtx{})
	res, err := service.Update(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}

// ProPublicGetAnnouncement 用户侧读取站点公告。
func ProPublicGetAnnouncement(c *gin.Context) {
	service := &prosvc.GetAnnouncementService{}
	res, err := service.Get(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}
	c.JSON(200, serializer.Response{Data: res})
}