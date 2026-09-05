package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/service/callback"
	"github.com/gin-gonic/gin"
)

// writePaymentReply 写出各支付渠道约定的回调应答。
func writePaymentReply(c *gin.Context, result callback.NotifyResult) {
	if result.ReplyXML {
		c.Data(200, "application/xml; charset=utf-8", []byte(result.Reply))
		return
	}
	c.String(200, result.Reply)
}

// ProAlipayCallback 支付宝当面付异步通知回调。
func ProAlipayCallback(c *gin.Context) {
	service := &callback.PaymentService{}
	result := service.HandleAlipay(c)
	if result.Reply != "success" {
		logging.FromContext(c).Warning("Alipay callback failed: %s", result.Reply)
	}
	writePaymentReply(c, result)
}

// ProWechatCallback 微信直连 Native 异步通知回调。
func ProWechatCallback(c *gin.Context) {
	service := &callback.PaymentService{}
	result := service.HandleWechat(c)
	writePaymentReply(c, result)
}

// ProPayJSCallback PAYJS 异步通知回调。
func ProPayJSCallback(c *gin.Context) {
	service := &callback.PaymentService{}
	result := service.HandlePayJS(c)
	if result.Reply != "success" {
		logging.FromContext(c).Warning("PAYJS callback failed: %s", result.Reply)
	}
	writePaymentReply(c, result)
}

// ProCustomCallback 自定义支付渠道异步通知回调。
func ProCustomCallback(c *gin.Context) {
	service := &callback.PaymentService{}
	result := service.HandleCustom(c)
	if result.Reply != "success" {
		logging.FromContext(c).Warning("Custom payment callback failed: %s", result.Reply)
	}
	writePaymentReply(c, result)
}
