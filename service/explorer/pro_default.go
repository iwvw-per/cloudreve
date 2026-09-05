package explorer

import (
	"context"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// GetGroupDefaultShares 返回当前用户所属用户组配置的默认分享（未过期的）。
// 支持分享的数字 id 与 hash id 两种存储形式。
func GetGroupDefaultShares(c *gin.Context, user *ent.User) ([]*ent.Share, error) {
	if user == nil || user.Edges.Group == nil || user.Edges.Group.Settings == nil {
		return nil, nil
	}

	rawIDs := user.Edges.Group.Settings.DefaultShares
	if len(rawIDs) == 0 {
		return nil, nil
	}

	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()

	var shareIDs []int
	for _, raw := range rawIDs {
		if id, err := hasher.Decode(raw, hashid.ShareID); err == nil {
			shareIDs = append(shareIDs, id)
			continue
		}

		if id, err := strconv.Atoi(raw); err == nil {
			shareIDs = append(shareIDs, id)
		}
	}

	if len(shareIDs) == 0 {
		return nil, nil
	}

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	shares, err := dep.ShareClient().GetByIDs(ctx, shareIDs)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to load default shares", err)
	}

	// 过滤已过期/不可用的默认分享
	return lo.Filter(shares, func(s *ent.Share, _ int) bool {
		return inventory.IsValidShare(s) == nil
	}), nil
}

// MergeDefaultSharesToPinned 把用户所属组的默认分享合并到已加载的分享列表中。
// 默认分享置于列表最前，且按 share ID 与已存在项去重。
func MergeDefaultSharesToPinned(c *gin.Context, user *ent.User, shares []*ent.Share) ([]*ent.Share, error) {
	defaultShares, err := GetGroupDefaultShares(c, user)
	if err != nil {
		return nil, err
	}

	existed := lo.SliceToMap(shares, func(s *ent.Share) (int, struct{}) {
		return s.ID, struct{}{}
	})

	merged := make([]*ent.Share, 0, len(defaultShares)+len(shares))
	for _, ds := range defaultShares {
		if _, ok := existed[ds.ID]; ok {
			continue
		}
		merged = append(merged, ds)
	}
	merged = append(merged, shares...)

	return merged, nil
}
