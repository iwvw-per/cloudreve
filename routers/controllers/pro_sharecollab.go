package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	sharesvc "github.com/cloudreve/Cloudreve/v4/service/share"
	"github.com/gin-gonic/gin"
)

// ShareUpload 经分享链接上传文件
func ShareUpload(c *gin.Context) {
	service := ParametersFromContext[*explorer.ShareUploadService](c, explorer.ShareUploadParamCtx{})
	service.ShareID = hashid.FromContext(c)
	res, err := service.Upload(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// ShareModify 经分享链接修改分享内文件（重命名/更新内容）
func ShareModify(c *gin.Context) {
	service := ParametersFromContext[*explorer.ShareModifyService](c, explorer.ShareModifyParamCtx{})
	service.ShareID = hashid.FromContext(c)
	res, err := service.Modify(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// ShareDelete 经分享链接删除分享内文件
func ShareDelete(c *gin.Context) {
	service := ParametersFromContext[*explorer.ShareDeleteService](c, explorer.ShareDeleteParamCtx{})
	service.ShareID = hashid.FromContext(c)
	err := service.Delete(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{})
}

// BuyShare 用积分购买付费分享访问权
func BuyShare(c *gin.Context) {
	service := ParametersFromContext[*sharesvc.BuyShareService](c, sharesvc.BuyShareParamCtx{})
	service.ShareID = hashid.FromContext(c)
	res, err := service.Buy(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}
