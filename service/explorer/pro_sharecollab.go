package explorer

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/admin"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// ShareWriteType 分享内写操作类型。
type ShareWriteType int

const (
	ShareWriteUpload ShareWriteType = iota
	ShareWriteModify
	ShareWriteDelete
)

// CheckShareCollabWrite 校验当前用户是否可在分享内执行指定写操作。
func CheckShareCollabWrite(c *gin.Context, share *ent.Share, writeType ShareWriteType) error {
	props := share.Props
	if props == nil {
		return serializer.NewError(serializer.CodeNoPermissionErr, "share does not allow write operations", nil)
	}

	u := inventory.UserFromContext(c)
	anonymous := u == nil || inventory.IsAnonymousUser(u)

	switch writeType {
	case ShareWriteUpload:
		if anonymous {
			if !props.AllowAnonymousUpload {
				return serializer.NewError(serializer.CodeNoPermissionErr, "anonymous upload is not allowed for this share", nil)
			}
		} else if !props.AllowUpload {
			return serializer.NewError(serializer.CodeNoPermissionErr, "upload is not allowed for this share", nil)
		}
	case ShareWriteModify:
		if anonymous || !props.AllowModify {
			return serializer.NewError(serializer.CodeNoPermissionErr, "modify is not allowed for this share", nil)
		}
	case ShareWriteDelete:
		if anonymous || !props.AllowDelete {
			return serializer.NewError(serializer.CodeNoPermissionErr, "delete is not allowed for this share", nil)
		}
	default:
		return serializer.NewError(serializer.CodeNoPermissionErr, "unsupported write operation", nil)
	}

	return nil
}

// HasPurchasedShare 判断当前用户是否已购买分享访问权（所有者视为已购买）。
func HasPurchasedShare(c *gin.Context, share *ent.Share) bool {
	u := inventory.UserFromContext(c)
	if u == nil || inventory.IsAnonymousUser(u) {
		return false
	}
	if share.Edges.User != nil && share.Edges.User.ID == u.ID {
		return true
	}
	if share.Props == nil || len(share.Props.PurchasedUsers) == 0 {
		return false
	}
	return lo.Contains(share.Props.PurchasedUsers, u.ID)
}

// CheckShareAccessOrPurchased 校验付费分享是否已购买；免费分享直接通过。
func CheckShareAccessOrPurchased(c *gin.Context, share *ent.Share) error {
	if share.Props == nil || share.Props.Price <= 0 {
		return nil
	}
	if HasPurchasedShare(c, share) {
		return nil
	}
	return serializer.NewError(serializer.CodePurchaseRequired, "You need to purchase this share before accessing it", nil)
}

// LoadShareForWrite 加载分享并校验当前用户的写权限。
func LoadShareForWrite(c *gin.Context, shareID int, writeType ShareWriteType) (*ent.Share, error) {
	dep := dependency.FromContext(c)

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadShareFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadUserGroup{}, true)
	share, err := dep.ShareClient().GetByID(ctx, shareID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "share not found", err)
	}
	if err := inventory.IsValidShare(share); err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "share link expired", err)
	}

	u := inventory.UserFromContext(c)
	if u != nil && share.Edges.User != nil && share.Edges.User.ID == u.ID {
		return share, nil
	}

	if err := CheckShareAccessOrPurchased(c, share); err != nil {
		return nil, err
	}
	if err := CheckShareCollabWrite(c, share, writeType); err != nil {
		return nil, err
	}

	return share, nil
}

// resolveOwnerRootUri 计算分享根文件在所有者 "my" 文件系统下的 URI，
// 后续写操作均以所有者身份落到其真实目录上。
func resolveOwnerRootUri(c *gin.Context, dep dependency.Dep, share *ent.Share) (*fs.URI, error) {
	owner := share.Edges.User
	if owner == nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "share owner not found", nil)
	}
	sourceFile := share.Edges.File
	if sourceFile == nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "share source file not found", nil)
	}

	names := []string{sourceFile.Name}
	current := sourceFile
	for {
		parent, err := dep.FileClient().GetParentFile(c, current, false)
		if err != nil {
			if ent.IsNotFound(err) {
				break
			}
			return nil, serializer.NewError(serializer.CodeDBError, "failed to resolve share root", err)
		}
		if parent.Name == inventory.RootFolderName {
			break
		}
		names = append([]string{parent.Name}, names...)
		current = parent
	}

	base, err := fs.NewUriFromString(fs.NewMyUri(hashid.EncodeUserID(dep.HashIDEncoder(), owner.ID)))
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "failed to build owner uri", err)
	}
	return base.Join(names...), nil
}

// shareUriToOwnerUri 将分享内 URI（cloudreve://<shareId>@share/<path>）转换为
// 所有者文件系统下的目标 URI。
func shareUriToOwnerUri(c *gin.Context, dep dependency.Dep, share *ent.Share, shareUri *fs.URI) (*fs.URI, error) {
	root, err := resolveOwnerRootUri(c, dep, share)
	if err != nil {
		return nil, err
	}
	return root.JoinRaw(shareUri.PathTrimmed()), nil
}

// validateShareUri 校验请求中的分享 URI 与路由中的分享 ID 一致。
func validateShareUri(hasher hashid.Encoder, share *ent.Share, uri *fs.URI) error {
	if uri == nil || uri.FileSystem() != constants.FileSystemShare {
		return serializer.NewError(serializer.CodeParamErr, "invalid share uri", nil)
	}
	if uri.ID("") != hashid.EncodeShareID(hasher, share.ID) {
		return serializer.NewError(serializer.CodeParamErr, "share uri does not match share", nil)
	}
	return nil
}

type (
	ShareUploadParamCtx struct{}
	// ShareUploadService 经分享链接上传文件：请求体为文件原始内容，目标路径由 uri 指定。
	ShareUploadService struct {
		ShareID      int
		Uri          string `form:"uri" json:"uri" binding:"required"`
		LastModified int64  `form:"last_modified" json:"last_modified"`
	}
)

func (s *ShareUploadService) Upload(c *gin.Context) (*FileResponse, error) {
	dep := dependency.FromContext(c)
	share, err := LoadShareForWrite(c, s.ShareID, ShareWriteUpload)
	if err != nil {
		return nil, err
	}

	shareUri, err := fs.NewUriFromString(s.Uri)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}
	if err := validateShareUri(dep.HashIDEncoder(), share, shareUri); err != nil {
		return nil, err
	}

	targetUri, err := shareUriToOwnerUri(c, dep, share, shareUri)
	if err != nil {
		return nil, err
	}

	rc, fileSize, err := request.SniffContentLength(c.Request)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "invalid content length", err)
	}

	props := &fs.UploadProps{
		Uri:  targetUri,
		Size: fileSize,
	}
	if s.LastModified > 0 {
		lastModified := time.UnixMilli(s.LastModified)
		props.LastModified = &lastModified
	}

	fileData := &fs.UploadRequest{
		Props: props,
		File:  rc,
		Mode:  fs.ModeOverwrite,
	}

	m := manager.NewFileManager(dep, share.Edges.User)
	defer m.Recycle()

	file, err := m.Update(c, fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to share: %w", err)
	}

	u := inventory.UserFromContext(c)
	admin.RecordEvent(c, &inventory.CreateEventParams{
		Type:    types.AuditTypeEntityUploaded,
		UserID:  inventory.UserIDFromContext(c),
		ShareID: share.ID,
		Content: &types.AuditContent{
			OriginalName: shareUri.Name(),
		},
	})

	return BuildFileResponse(c, u, file, dep.HashIDEncoder(), nil), nil
}

type (
	ShareModifyParamCtx struct{}
	// ShareModifyService 经分享链接修改分享内文件：
	// 提供 new_name 时执行重命名，否则将请求体作为文件新内容写入 uri。
	ShareModifyService struct {
		ShareID int
		Uri     string `form:"uri" json:"uri" binding:"required"`
		NewName string `form:"new_name" json:"new_name"`
	}
)

func (s *ShareModifyService) Modify(c *gin.Context) (*FileResponse, error) {
	if s.NewName != "" {
		return s.rename(c)
	}
	return s.updateContent(c)
}

func (s *ShareModifyService) rename(c *gin.Context) (*FileResponse, error) {
	dep := dependency.FromContext(c)
	share, err := LoadShareForWrite(c, s.ShareID, ShareWriteModify)
	if err != nil {
		return nil, err
	}

	shareUri, err := fs.NewUriFromString(s.Uri)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}
	if err := validateShareUri(dep.HashIDEncoder(), share, shareUri); err != nil {
		return nil, err
	}

	targetUri, err := shareUriToOwnerUri(c, dep, share, shareUri)
	if err != nil {
		return nil, err
	}

	m := manager.NewFileManager(dep, share.Edges.User)
	defer m.Recycle()

	file, err := m.Rename(c, targetUri, s.NewName)
	if err != nil {
		return nil, fmt.Errorf("failed to rename shared file: %w", err)
	}

	u := inventory.UserFromContext(c)
	admin.RecordEvent(c, &inventory.CreateEventParams{
		Type:    types.AuditTypeFileRename,
		UserID:  inventory.UserIDFromContext(c),
		ShareID: share.ID,
		Content: &types.AuditContent{
			OriginalName: shareUri.Name(),
			NewName:      s.NewName,
		},
	})

	return BuildFileResponse(c, u, file, dep.HashIDEncoder(), nil), nil
}

func (s *ShareModifyService) updateContent(c *gin.Context) (*FileResponse, error) {
	dep := dependency.FromContext(c)
	share, err := LoadShareForWrite(c, s.ShareID, ShareWriteModify)
	if err != nil {
		return nil, err
	}

	shareUri, err := fs.NewUriFromString(s.Uri)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}
	if err := validateShareUri(dep.HashIDEncoder(), share, shareUri); err != nil {
		return nil, err
	}

	targetUri, err := shareUriToOwnerUri(c, dep, share, shareUri)
	if err != nil {
		return nil, err
	}

	rc, fileSize, err := request.SniffContentLength(c.Request)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "invalid content length", err)
	}
	if fileSize > dep.SettingProvider().MaxOnlineEditSize(c) {
		return nil, fs.ErrFileSizeTooBig
	}

	fileData := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:  targetUri,
			Size: fileSize,
		},
		File: rc,
		Mode: fs.ModeOverwrite,
	}

	m := manager.NewFileManager(dep, share.Edges.User)
	defer m.Recycle()

	file, err := m.Update(c, fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to update shared file: %w", err)
	}

	u := inventory.UserFromContext(c)
	admin.RecordEvent(c, &inventory.CreateEventParams{
		Type:    types.AuditTypeEntityUploaded,
		UserID:  inventory.UserIDFromContext(c),
		ShareID: share.ID,
		Content: &types.AuditContent{
			OriginalName: shareUri.Name(),
		},
	})

	return BuildFileResponse(c, u, file, dep.HashIDEncoder(), nil), nil
}

type (
	ShareDeleteParamCtx struct{}
	// ShareDeleteService 经分享链接删除分享内文件。
	ShareDeleteService struct {
		ShareID int
		Uris    []string `json:"uris" binding:"required,min=1"`
	}
)

func (s *ShareDeleteService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	share, err := LoadShareForWrite(c, s.ShareID, ShareWriteDelete)
	if err != nil {
		return err
	}

	uris, err := fs.NewUriFromStrings(s.Uris...)
	if err != nil {
		return serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}
	if len(uris) > 100 {
		return serializer.NewError(serializer.CodeParamErr, "too many uris", nil)
	}

	targets := make([]*fs.URI, 0, len(uris))
	for _, u := range uris {
		if err := validateShareUri(dep.HashIDEncoder(), share, u); err != nil {
			return err
		}
		target, err := shareUriToOwnerUri(c, dep, share, u)
		if err != nil {
			return err
		}
		targets = append(targets, target)
	}

	m := manager.NewFileManager(dep, share.Edges.User)
	defer m.Recycle()

	if err := m.Delete(c, targets); err != nil {
		return fmt.Errorf("failed to delete shared file: %w", err)
	}

	admin.RecordEvent(c, &inventory.CreateEventParams{
		Type:    types.AuditTypeDeleteFile,
		UserID:  inventory.UserIDFromContext(c),
		ShareID: share.ID,
		Content: &types.AuditContent{
			OriginalName: s.Uris[0],
		},
	})

	return nil
}
