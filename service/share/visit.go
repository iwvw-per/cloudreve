package share

import (
	"context"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster/routes"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/cloudreve/Cloudreve/v4/service/explorer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type (
	ShortLinkRedirectService struct {
		ID       string `uri:"id" binding:"required"`
		Password string `uri:"password"`
	}
	ShortLinkRedirectParamCtx struct{}
)

func (s *ShortLinkRedirectService) RedirectTo(c *gin.Context) string {
	shareLongUrl := routes.MasterShareLongUrl(s.ID, s.Password)

	shortLinkQuery := c.Request.URL.Query() // Query in ShortLink, adapt to Cloudreve V3
	shareLongUrlQuery := shareLongUrl.Query()

	userSpecifiedPath := shortLinkQuery.Get("path")
	if userSpecifiedPath != "" {
		masterPath := shareLongUrlQuery.Get("path")
		masterPath += "/" + strings.TrimPrefix(userSpecifiedPath, "/")

		shareLongUrlQuery.Set("path", masterPath)
	}

	shortLinkQuery.Del("path") // 防止用户指定的 Path 就是空字符串
	for k, vals := range shortLinkQuery {
		shareLongUrlQuery[k] = append(shareLongUrlQuery[k], vals...)
	}

	shareLongUrl.RawQuery = shareLongUrlQuery.Encode()
	return shareLongUrl.String()
}

type (
	ShareInfoService struct {
		Password      string `form:"password"`
		CountViews    bool   `form:"count_views"`
		OwnerExtended bool   `form:"owner_extended"`
	}
	ShareInfoParamCtx struct{}
)

func (s *ShareInfoService) Get(c *gin.Context) (*explorer.Share, error) {
	dep := dependency.FromContext(c)
	u := inventory.UserFromContext(c)
	shareClient := dep.ShareClient()

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	share, err := shareClient.GetByID(ctx, hashid.FromContext(c))
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, serializer.NewError(serializer.CodeNotFound, "Share not found", nil)
		}
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get share", err)
	}

	if err := inventory.IsValidShare(share); err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Share link expired", err)
	}

	if s.CountViews {
		_ = shareClient.Viewed(c, share)
	}

	unlocked := true
	// Share requires password
	if share.Password != "" && s.Password != share.Password && share.Edges.User.ID != u.ID {
		unlocked = false
	}

	base := dep.SettingProvider().SiteURL(c)
	res := explorer.BuildShare(c, share, base, dep.HashIDEncoder(), u, share.Edges.User, share.Edges.File.Name,
		types.FileType(share.Edges.File.Type), unlocked, false)

	if s.OwnerExtended && share.Edges.User.ID == u.ID {
		// Add more information about the shared file
		m := manager.NewFileManager(dep, u)
		defer m.Recycle()

		shareUri, err := fs.NewUriFromString(fs.NewShareUri(res.ID, s.Password))
		if err != nil {
			return nil, serializer.NewError(serializer.CodeInternalSetting, "Invalid share url", err)
		}

		root, err := m.Get(c, shareUri)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeNotFound, "File not found", err)
		}

		res.SourceUri = root.Uri(true).String()
	}

	return res, nil

}

type (
	ListShareService struct {
		PageSize       int    `form:"page_size" binding:"required,min=10,max=100"`
		OrderBy        string `uri:"order_by" form:"order_by" json:"order_by"`
		OrderDirection string `uri:"order_direction" form:"order_direction" json:"order_direction"`
		NextPageToken  string `form:"next_page_token"`
	}
	ListShareParamCtx struct{}
)

func (s *ListShareService) List(c *gin.Context) (*ListShareResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	shareClient := dep.ShareClient()

	args := &inventory.ListShareArgs{
		PaginationArgs: &inventory.PaginationArgs{
			UseCursorPagination: true,
			PageToken:           s.NextPageToken,
			PageSize:            s.PageSize,
			Order:               inventory.OrderDirection(s.OrderDirection),
			OrderBy:             s.OrderBy,
		},
		UserID: user.ID,
	}

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	res, err := shareClient.List(ctx, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shares", err)
	}

	// 合并用户所属组的默认固定分享（置于列表最前、去重）
	res.Shares, err = explorer.MergeDefaultSharesToPinned(c, user, res.Shares)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to merge default shares", err)
	}

	base := dep.SettingProvider().SiteURL(ctx)
	return BuildListShareResponse(ctx, res, hasher, base, user, true), nil
}

func (s *ListShareService) ListInUserProfile(c *gin.Context, uid int) (*ListShareResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	hasher := dep.HashIDEncoder()
	shareClient := dep.ShareClient()

	targetUser, err := dep.UserClient().GetActiveByID(c, uid)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user", err)
	}

	if targetUser.Settings != nil && targetUser.Settings.ShareLinksInProfile == types.ProfileHideShare {
		return nil, serializer.NewError(serializer.CodeParamErr, "User has disabled share links in profile", nil)
	}

	publicOnly := targetUser.Settings == nil || targetUser.Settings.ShareLinksInProfile == types.ProfilePublicShareOnly
	args := &inventory.ListShareArgs{
		PaginationArgs: &inventory.PaginationArgs{
			UseCursorPagination: true,
			PageToken:           s.NextPageToken,
			PageSize:            s.PageSize,
			Order:               inventory.OrderDirection(s.OrderDirection),
			OrderBy:             s.OrderBy,
		},
		UserID:     uid,
		PublicOnly: publicOnly,
	}

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadFileMetadata{}, true)
	res, err := shareClient.List(ctx, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list shares", err)
	}

	base := dep.SettingProvider().SiteURL(ctx)
	return BuildListShareResponse(ctx, res, hasher, base, user, false), nil
}

type (
	// BuyShareService 用积分购买付费分享的访问权。
	BuyShareService struct {
		ShareID int
	}
	BuyShareParamCtx struct{}
	// BuyShareResponse 购买结果。
	BuyShareResponse struct {
		Purchased bool `json:"purchased"`
		Price     int  `json:"price"`
		Credit    int  `json:"credit"`
	}
)

// Buy 校验访问者积分并扣减，记录购买者并写审计日志。
func (s *BuyShareService) Buy(c *gin.Context) (*BuyShareResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	if user == nil || inventory.IsAnonymousUser(user) {
		return nil, serializer.NewError(serializer.CodeCheckLogin, "Login required", nil)
	}

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
	share, err := dep.ShareClient().GetByID(ctx, s.ShareID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "share not found", err)
	}
	if err := inventory.IsValidShare(share); err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "share link expired", err)
	}
	if share.Edges.User == nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "share owner not found", nil)
	}
	if share.Edges.User.ID == user.ID {
		return nil, serializer.NewError(serializer.CodeParamErr, "cannot purchase your own share", nil)
	}

	props := share.Props
	if props == nil || props.Price <= 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "this share is free", nil)
	}

	if lo.Contains(props.PurchasedUsers, user.ID) {
		return &BuyShareResponse{Purchased: true, Price: props.Price, Credit: user.Credit}, nil
	}

	if user.Credit < props.Price {
		return nil, serializer.NewError(serializer.CodeInsufficientCredit, "Insufficient credits", nil)
	}

	// 扣减积分
	newCredit := user.Credit - props.Price
	if _, err := dep.UserClient().GetClient().User.UpdateOneID(user.ID).SetCredit(newCredit).Save(ctx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to deduct credits", err)
	}

	// 记录购买者
	newProps := *props
	newProps.PurchasedUsers = lo.Uniq(append(append([]int{}, props.PurchasedUsers...), user.ID))
	if _, err := dep.ShareClient().Upsert(ctx, &inventory.CreateShareParams{
		Existed: share,
		Props:   &newProps,
	}); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to record purchase", err)
	}

	// 审计
	admin.RecordEvent(c, &inventory.CreateEventParams{
		Type:    types.AuditTypePointsChange,
		UserID:  user.ID,
		ShareID: share.ID,
		Content: &types.AuditContent{
			PointsChange: -props.Price,
			Reason:       "purchase share access",
		},
	})

	return &BuyShareResponse{Purchased: true, Price: props.Price, Credit: newCredit}, nil
}
