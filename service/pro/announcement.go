package pro

import (
	"context"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	entsetting "github.com/cloudreve/Cloudreve/v4/ent/setting"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	pkgsetting "github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gin-gonic/gin"
)

const (
	AnnouncementKey        = "announcement"
	AnnouncementEnabledKey = "announcement_enabled"
)

// Announcement 站点公告序列化结构。
type Announcement struct {
	Enabled   bool      `json:"enabled"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetAnnouncementService 读取当前生效的站点公告。
type (
	GetAnnouncementService struct{}
	GetAnnouncementParamCtx struct{}
)

func (s *GetAnnouncementService) Get(c *gin.Context) (*Announcement, error) {
	return getAnnouncement(c, dependency.FromContext(c))
}

// UpdateAnnouncementService 保存站点公告。
type (
	UpdateAnnouncementService struct {
		Enabled bool   `json:"enabled"`
		Content string `json:"content"`
	}
	UpdateAnnouncementParamCtx struct{}
)

func (s *UpdateAnnouncementService) Update(c *gin.Context) (*Announcement, error) {
	dep := dependency.FromContext(c)

	if err := upsertAnnouncementSetting(c, dep, AnnouncementKey, s.Content); err != nil {
		return nil, err
	}

	enabled := "0"
	if s.Enabled {
		enabled = "1"
	}
	if err := upsertAnnouncementSetting(c, dep, AnnouncementEnabledKey, enabled); err != nil {
		return nil, err
	}

	kv := dep.KV()
	if err := kv.Delete(pkgsetting.KvSettingPrefix, AnnouncementKey, AnnouncementEnabledKey); err != nil {
		return nil, serializer.NewError(serializer.CodeInternalSetting, "Failed to clear cache", err)
	}

	return getAnnouncement(c, dep)
}

func getAnnouncement(c context.Context, dep dependency.Dep) (*Announcement, error) {
	sc := dep.SettingClient()

	content := ""
	if v, err := sc.Get(c, AnnouncementKey); err == nil {
		content = v
	} else if !ent.IsNotFound(err) {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to read announcement", err)
	}

	enabled := false
	if v, err := sc.Get(c, AnnouncementEnabledKey); err == nil {
		enabled = pkgsetting.IsTrueValue(v)
	} else if !ent.IsNotFound(err) {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to read announcement", err)
	}

	updatedAt, err := announcementUpdatedAt(c, dep)
	if err != nil {
		return nil, err
	}

	return &Announcement{
		Enabled:   enabled,
		Content:   content,
		UpdatedAt: updatedAt,
	}, nil
}

func announcementUpdatedAt(c context.Context, dep dependency.Dep) (time.Time, error) {
	row, err := dep.DBClient().Setting.Query().Where(entsetting.Name(AnnouncementKey)).Only(c)
	if ent.IsNotFound(err) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, serializer.NewError(serializer.CodeDBError, "Failed to read announcement", err)
	}
	return row.UpdatedAt, nil
}

func upsertAnnouncementSetting(c context.Context, dep dependency.Dep, name, value string) error {
	err := dep.DBClient().Setting.Create().
		SetName(name).
		SetValue(value).
		OnConflictColumns(entsetting.FieldName).
		UpdateNewValues().
		Exec(c)
	if err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to save announcement", err)
	}
	return nil
}