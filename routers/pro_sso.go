package routers

import (
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	authsvc "github.com/cloudreve/Cloudreve/v4/service/auth"
	"github.com/gin-gonic/gin"
)

// RegisterSSORoutes registers the third-party SSO (Logto / OIDC / QQ) routes.
// Start triggers the OIDC/OAuth authorization redirect, callback exchanges the
// authorization code for tokens and signs the user in.
//
// It is invoked from the `session` group in router.go so that the routes share
// the same session / current-user middleware as the rest of the auth endpoints.
func RegisterSSORoutes(session *gin.RouterGroup) {
	sso := session.Group("sso/:provider")
	{
		sso.GET("start", ssoStart)
		sso.GET("callback", ssoCallback)
	}
}

// ssoStart initiates the SSO sign-in and returns the provider authorization URL.
func ssoStart(c *gin.Context) {
	provider := authsvc.Provider(c.Param("provider"))
	if !authsvc.IsSupportedProvider(provider) {
		c.JSON(200, serializer.Err(c, authsvc.UnsupportedProviderError()))
		return
	}

	svc := authsvc.NewSSOService(provider)
	res, err := svc.Start(c)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}

// ssoCallback completes the SSO login after the provider redirects the browser.
func ssoCallback(c *gin.Context) {
	provider := authsvc.Provider(c.Param("provider"))
	if !authsvc.IsSupportedProvider(provider) {
		c.JSON(200, serializer.Err(c, authsvc.UnsupportedProviderError()))
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	svc := authsvc.NewSSOService(provider)
	res, err := svc.Callback(c, code, state)
	if err != nil {
		c.JSON(200, serializer.Err(c, err))
		return
	}

	c.JSON(200, serializer.Response{Data: res})
}