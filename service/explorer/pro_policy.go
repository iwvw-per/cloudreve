package explorer

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// groupPolicyIDs 返回用户组可用的存储策略 ID 列表。
// 组配置了多存储策略（AvailablePolicyIDs）时返回该列表；
// 未配置时回退到默认单策略边（storage_policies），保持旧行为。
func groupPolicyIDs(group *ent.Group) []int {
	if group == nil {
		return nil
	}
	if len(group.Settings.AvailablePolicyIDs) > 0 {
		return group.Settings.AvailablePolicyIDs
	}
	if group.Edges.StoragePolicies != nil && group.Edges.StoragePolicies.ID > 0 {
		return []int{group.Edges.StoragePolicies.ID}
	}
	return nil
}

// loadGroupWithPolicy 加载用户组并带上 storage_policies 边。
func loadGroupWithPolicy(ctx context.Context, dep dependency.Dep, groupID int) (*ent.Group, error) {
	gctx := context.WithValue(ctx, inventory.LoadGroupPolicy{}, true)
	return dep.GroupClient().GetByID(gctx, groupID)
}

// resolveUploadPolicyID 校验请求的存储策略是否在当前用户组可用范围内，
// 可用则返回该策略 ID，否则返回 0（回退用户组默认策略）。
func resolveUploadPolicyID(ctx context.Context, dep dependency.Dep, user *ent.User, requested int) int {
	if user == nil || user.Edges.Group == nil || requested == 0 {
		return 0
	}
	group, err := loadGroupWithPolicy(ctx, dep, user.Edges.Group.ID)
	if err != nil || group == nil {
		return 0
	}
	if lo.Contains(groupPolicyIDs(group), requested) {
		return requested
	}
	return 0
}

// walkUpDirectoryPreferredPolicy 在目标目录的父目录链中向上查找第一个设置了
// PreferredStoragePolicyID 的目录，返回其策略 ID。仅对用户自己的文件系统生效，
// 找不到时返回 0。
func walkUpDirectoryPreferredPolicy(ctx context.Context, dep dependency.Dep, user *ent.User, uri *fs.URI) int {
	if user == nil || uri.FileSystem() != constants.FileSystemMy {
		return 0
	}
	fileClient := dep.FileClient()
	root, err := fileClient.Root(ctx, user)
	if err != nil || root == nil {
		return 0
	}

	current := root
	for _, elem := range uri.DirUri().Elements() {
		child, err := fileClient.GetChildFile(ctx, current, user.ID, elem, false)
		if err != nil || child == nil {
			return 0
		}
		current = child
	}

	for depth := 0; depth < 64 && current != nil; depth++ {
		if current.Props != nil && current.Props.PreferredStoragePolicyID != 0 {
			return current.Props.PreferredStoragePolicyID
		}
		parent, err := fileClient.GetParentFile(ctx, current, false)
		if err != nil {
			break
		}
		current = parent
	}
	return 0
}

type (
	UserPoliciesParameterCtx struct{}
)

// UserPolicyResponse 用户侧可用的存储策略。id 为 hashid（与前端 StoragePolicy 一致，
// 用于上传时指定 policy_id），storage_policy_id 为原始数据库 ID（用于设置目录偏好策略）。
type UserPolicyResponse struct {
	ID               string           `json:"id"`
	StoragePolicyID  int              `json:"storage_policy_id"`
	Name             string           `json:"name"`
	AllowedSuffix    []string         `json:"allowed_suffix,omitempty"`
	DeniedSuffix     []string         `json:"denied_suffix,omitempty"`
	Type             types.PolicyType `json:"type"`
	MaxSize          int64            `json:"max_size"`
	Relay            bool             `json:"relay,omitempty"`
	ChunkConcurrency int              `json:"chunk_concurrency,omitempty"`
	Encryption       bool             `json:"encryption,omitempty"`
}

// UserPolicies 返回当前登录用户组可用的存储策略列表。
func UserPolicies(c *gin.Context) ([]*UserPolicyResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	if user == nil || user.Edges.Group == nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "user group not loaded", nil)
	}

	group, err := loadGroupWithPolicy(c, dep, user.Edges.Group.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get user group", err)
	}

	ids := groupPolicyIDs(group)
	hasher := dep.HashIDEncoder()
	res := make([]*UserPolicyResponse, 0, len(ids))
	for _, id := range ids {
		p, err := dep.StoragePolicyClient().GetPolicyByID(c, id)
		if err != nil || p == nil || p.Settings == nil {
			continue
		}
		sp := BuildStoragePolicy(p, hasher)
		res = append(res, &UserPolicyResponse{
			ID:               sp.ID,
			StoragePolicyID:  p.ID,
			Name:             sp.Name,
			AllowedSuffix:    sp.AllowedSuffix,
			DeniedSuffix:     sp.DeniedSuffix,
			Type:             sp.Type,
			MaxSize:          sp.MaxSize,
			Relay:            sp.Relay,
			ChunkConcurrency: sp.ChunkConcurrency,
			Encryption:       sp.Encryption,
		})
	}
	return res, nil
}

type (
	SetFilePolicyParameterCtx struct{}
	SetFilePolicyService  struct {
		Uri            string `json:"uri" binding:"required"`
		StoragePolicyID int    `json:"storage_policy_id"`
	}
)

// Set 设置目录/文件的偏好存储策略。StoragePolicyID 为 0 时清除偏好设置。
func (s *SetFilePolicyService) Set(c *gin.Context) (*FileResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	m := manager.NewFileManager(dep, user)
	defer m.Recycle()

	uri, err := fs.NewUriFromString(s.Uri)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

	file, err := m.Get(c, uri, dbfs.WithNotRoot())
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	if file == nil || file.IsNil() {
		return nil, serializer.NewError(serializer.CodeNotFound, "file not found", nil)
	}

	if user.Edges.Group == nil || (file.OwnerID() != user.ID && !user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin))) {
		return nil, fs.ErrOwnerOnly
	}

	dbf, ok := file.(*dbfs.File)
	if !ok {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "unsupported file type", nil)
	}

	props := dbf.Model.Props
	if props == nil {
		props = &types.FileProps{}
	}
	props.PreferredStoragePolicyID = s.StoragePolicyID
	if _, err := dep.FileClient().UpdateProps(c, dbf.Model, props); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "failed to update preferred storage policy", err)
	}

	updated, err := m.Get(c, uri, dbfs.WithNotRoot())
	if err != nil {
		return nil, fmt.Errorf("failed to get updated file: %w", err)
	}

	return BuildFileResponse(c, user, updated, dep.HashIDEncoder(), nil), nil
}
