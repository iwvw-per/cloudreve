package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

const (
	filterEmailProviderKey  = "filter_email_provider"
	filterEmailProviderRule = "filter_email_provider_rules"
)

// ProAdminGetEmailFilter 回读邮箱域名过滤配置。
func ProAdminGetEmailFilter(c *gin.Context) {
	dep := dependency.FromContext(c)
	vals, err := dep.SettingClient().Gets(c, []string{filterEmailProviderKey, filterEmailProviderRule})
	if err != nil {
		c.JSON(200, serializer.ErrWithDetails(c, serializer.CodeDBError, "Failed to load email filter settings", err))
		return
	}
	c.JSON(200, serializer.Response{Data: vals})
}