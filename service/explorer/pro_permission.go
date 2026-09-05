package explorer

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

type FileAccessAction int

const (
	FileAccessActionView FileAccessAction = iota + 1
	FileAccessActionDownload
	FileAccessActionWrite
)

func filePropsOf(f fs.File) *types.FileProps {
	if f == nil || f.IsNil() {
		return nil
	}
	if dbf, ok := f.(*dbfs.File); ok && dbf.Model != nil {
		return dbf.Model.Props
	}
	return nil
}

func CheckFileAccess(ctx context.Context, f fs.File, user *ent.User, action FileAccessAction) error {
	props := filePropsOf(f)
	if props == nil || props.Permissions == nil {
		return nil
	}

	rule := props.Permissions
	if user == nil || inventory.IsAnonymousUser(user) {
		if rule.Anonymous <= 0 {
			return nil
		}
		if int(action) > rule.Anonymous {
			return serializer.NewError(serializer.CodeNoPermissionErr, "no permission to access this file", nil)
		}
		return nil
	}

	for _, uid := range rule.DenyUsers {
		if uid == user.ID {
			return serializer.NewError(serializer.CodeNoPermissionErr, "no permission to access this file", nil)
		}
	}

	groupID := 0
	if user.Edges.Group != nil {
		groupID = user.Edges.Group.ID
	}
	for _, gid := range rule.DenyGroups {
		if gid == groupID {
			return serializer.NewError(serializer.CodeNoPermissionErr, "no permission to access this file", nil)
		}
	}

	allowedByUser := false
	for _, uid := range rule.AllowUsers {
		if uid == user.ID {
			allowedByUser = true
			break
		}
	}

	allowedByGroup := true
	if len(rule.AllowGroups) > 0 {
		allowedByGroup = false
		for _, gid := range rule.AllowGroups {
			if gid == groupID {
				allowedByGroup = true
				break
			}
		}
	}

	if len(rule.AllowUsers) > 0 && !allowedByUser && !allowedByGroup {
		return serializer.NewError(serializer.CodeNoPermissionErr, "no permission to access this file", nil)
	}
	if !allowedByGroup {
		return serializer.NewError(serializer.CodeNoPermissionErr, "no permission to access this file", nil)
	}

	return nil
}

type (
	SetFilePermissionParamCtx struct{}
	SetFilePermissionService  struct {
		Uri         string `json:"uri" binding:"required"`
		AllowUsers  []int  `json:"allow_users"`
		DenyUsers   []int  `json:"deny_users"`
		AllowGroups []int  `json:"allow_groups"`
		DenyGroups  []int  `json:"deny_groups"`
		Anonymous   int    `json:"anonymous" binding:"min=0,max=3"`
	}
)

func (s *SetFilePermissionService) Set(c *gin.Context) (*FileResponse, error) {
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

	if file.OwnerID() != user.ID && !user.Edges.Group.Permissions.Enabled(int(types.GroupPermissionIsAdmin)) {
		return nil, fs.ErrOwnerOnly
	}

	props := filePropsOf(file)
	if props == nil {
		props = &types.FileProps{}
	}

	rule := &types.FileAccessRule{
		AllowUsers:  s.AllowUsers,
		DenyUsers:   s.DenyUsers,
		AllowGroups: s.AllowGroups,
		DenyGroups:  s.DenyGroups,
		Anonymous:   s.Anonymous,
	}
	if len(rule.AllowUsers) == 0 && len(rule.DenyUsers) == 0 && len(rule.AllowGroups) == 0 && len(rule.DenyGroups) == 0 && rule.Anonymous == 0 {
		props.Permissions = nil
	} else {
		props.Permissions = rule
	}

	dbf, ok := file.(*dbfs.File)
	if !ok {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "unsupported file type", nil)
	}
	if _, err := dep.FileClient().UpdateProps(c, dbf.Model, props); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "failed to update file permission", err)
	}

	updated, err := m.Get(c, uri, dbfs.WithNotRoot())
	if err != nil {
		return nil, fmt.Errorf("failed to get updated file: %w", err)
	}

	return BuildFileResponse(c, user, updated, dep.HashIDEncoder(), nil), nil
}
