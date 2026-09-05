package controllers

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// ListAccountsRequest 前端上报的已保存账号 token 列表，后端据此校验并返回账号概要。
// 每个 token 可以是 access token 或 refresh token，用于解析对应用户，不互相影响。
type ListAccountsRequest struct {
	Tokens []string `json:"tokens"`
}

// AccountSummary 账号切换所需的账号概要，仅暴露切换 UI 所需字段。
type AccountSummary struct {
	UserID   int    `json:"user_id"`
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
}

// ProListAccounts 校验前端持有的多个账号 token，返回每个账号的概要。
// 多账号切换为前端本地多 token 管理：此接口仅做无侵入的解析与概要返回，
// 不引入新的全局状态，也不依赖请求头中的当前账号。
func ProListAccounts(c *gin.Context) {
	req := &ListAccountsRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(200, serializer.ParamErr(c, "Invalid request body", err))
		return
	}

	dep := dependency.FromContext(c)
	idEncoder := dep.HashIDEncoder()
	userClient := dep.UserClient()

	seen := make(map[int]struct{})
	accounts := make([]AccountSummary, 0, len(req.Tokens))
	for _, token := range req.Tokens {
		claims, err := dep.TokenAuth().Claims(c, token)
		if err != nil {
			continue
		}

		uid, err := idEncoder.Decode(claims.Subject, hashid.UserID)
		if err != nil {
			continue
		}

		if _, ok := seen[uid]; ok {
			continue
		}

		u, err := userClient.GetActiveByID(c, uid)
		if err != nil {
			continue
		}

		seen[uid] = struct{}{}
		accounts = append(accounts, AccountSummary{
			UserID:   uid,
			ID:       hashid.EncodeUserID(idEncoder, uid),
			Nickname: u.Nick,
			Email:    u.Email,
		})
	}

	c.JSON(200, serializer.Response{Data: accounts})
}
