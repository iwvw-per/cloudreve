package types

import (
	"time"
)

// UserSetting 用户其他配置
type (
	UserSetting struct {
		ProfileOff          bool                     `json:"profile_off,omitempty"`
		PreferredTheme      string                   `json:"preferred_theme,omitempty"`
		VersionRetention    bool                     `json:"version_retention,omitempty"`
		VersionRetentionExt []string                 `json:"version_retention_ext,omitempty"`
		VersionRetentionMax int                      `json:"version_retention_max,omitempty"`
		Pined               []PinedFile              `json:"pined,omitempty"`
		Language            string                   `json:"email_language,omitempty"`
		DisableViewSync     bool                     `json:"disable_view_sync,omitempty"`
		FsViewMap           map[string]ExplorerView  `json:"fs_view_map,omitempty"`
		ShareLinksInProfile ShareLinksInProfileLevel `json:"share_links_in_profile,omitempty"`
	}

	ShareLinksInProfileLevel string

	PinedFile struct {
		Uri  string `json:"uri"`
		Name string `json:"name,omitempty"`
	}

	// GroupSetting 用户组其他配置
	GroupSetting struct {
		CompressSize          int64                  `json:"compress_size,omitempty"` // 可压缩大小
		DecompressSize        int64                  `json:"decompress_size,omitempty"`
		RemoteDownloadOptions map[string]interface{} `json:"remote_download_options,omitempty"` // 离线下载用户组配置
		SourceBatchSize       int                    `json:"source_batch,omitempty"`
		Aria2BatchSize        int                    `json:"aria2_batch,omitempty"`
		MaxWalkedFiles        int                    `json:"max_walked_files,omitempty"`
		TrashRetention        int                    `json:"trash_retention,omitempty"`
		RedirectedSource      bool                   `json:"redirected_source,omitempty"`
		DefaultShares         []string               `json:"default_shares,omitempty"` // 默认固定分享
		AllowedNodes          []int                  `json:"allowed_nodes,omitempty"`  // 可用任务节点
		// AvailablePolicyIDs 该组可用的多个存储策略（PRO 多存储策略）。
		// 与默认 storage_policies 边兼容：为空时回退到边的单一策略。
		AvailablePolicyIDs []int `json:"available_policy_ids,omitempty"`
	}

	// PolicySetting 非公有的存储策略属性
	PolicySetting struct {
		// Upyun访问Token
		Token string `json:"token"`
		// 允许的文件扩展名
		FileType []string `json:"file_type"`
		// IsFileTypeDenyList Whether above list is a deny list.
		IsFileTypeDenyList bool `json:"is_file_type_deny_list,omitempty"`
		// FileRegexp 文件扩展名正则表达式
		NameRegexp string `json:"file_regexp,omitempty"`
		// IsNameRegexp Whether above regexp is a deny list.
		IsNameRegexpDenyList bool `json:"is_name_regexp_deny_list,omitempty"`
		// OauthRedirect Oauth 重定向地址
		OauthRedirect string `json:"od_redirect,omitempty"`
		// CustomProxy whether to use custom-proxy to get file content
		CustomProxy bool `json:"custom_proxy,omitempty"`
		// ProxyServer 反代地址
		ProxyServer string `json:"proxy_server,omitempty"`
		// InternalProxy whether to use Cloudreve internal proxy to get file content
		InternalProxy bool `json:"internal_proxy,omitempty"`
		// OdDriver OneDrive 驱动器定位符
		OdDriver string `json:"od_driver,omitempty"`
		// Region 区域代码
		Region string `json:"region,omitempty"`
		// ServerSideEndpoint 服务端请求使用的 Endpoint，为空时使用 Policy.Server 字段
		ServerSideEndpoint string `json:"server_side_endpoint,omitempty"`
		// 分片上传的分片大小
		ChunkSize int64 `json:"chunk_size,omitempty"`
		// 每秒对存储端的 API 请求上限
		TPSLimit float64 `json:"tps_limit,omitempty"`
		// 每秒 API 请求爆发上限
		TPSLimitBurst int `json:"tps_limit_burst,omitempty"`
		// Set this to `true` to force the request to use path-style addressing,
		// i.e., `http://s3.amazonaws.com/BUCKET/KEY `
		S3ForcePathStyle bool `json:"s3_path_style"`
		// File extensions that support thumbnail generation using native policy API.
		ThumbExts []string `json:"thumb_exts,omitempty"`
		// Whether to support all file extensions for thumbnail generation.
		ThumbSupportAllExts bool `json:"thumb_support_all_exts,omitempty"`
		// ThumbMaxSize indicates the maximum allowed size of a thumbnail. 0 indicates that no limit is set.
		ThumbMaxSize int64 `json:"thumb_max_size,omitempty"`
		// Whether to upload file through server's relay.
		Relay bool `json:"relay,omitempty"`
		// Whether to pre allocate space for file before upload in physical disk.
		PreAllocate bool `json:"pre_allocate,omitempty"`
		// MediaMetaExts file extensions that support media meta generation using native policy API.
		MediaMetaExts []string `json:"media_meta_exts,omitempty"`
		// MediaMetaGeneratorProxy whether to use local proxy to generate media meta.
		MediaMetaGeneratorProxy bool `json:"media_meta_generator_proxy,omitempty"`
		// ThumbGeneratorProxy whether to use local proxy to generate thumbnail.
		ThumbGeneratorProxy bool `json:"thumb_generator_proxy,omitempty"`
		// NativeMediaProcessing whether to use native media processing API from storage provider.
		NativeMediaProcessing bool `json:"native_media_processing"`
		// S3DeleteBatchSize the number of objects to delete in each batch.
		S3DeleteBatchSize int `json:"s3_delete_batch_size,omitempty"`
		// StreamSaver whether to use stream saver to download file in Web.
		StreamSaver bool `json:"stream_saver,omitempty"`
		// UseCname whether to use CNAME for endpoint (OSS).
		UseCname bool `json:"use_cname,omitempty"`
		// CDN domain does not need to be signed.
		SourceAuth bool `json:"source_auth,omitempty"`
		// QiniuUploadCdn whether to use CDN for Qiniu upload.
		QiniuUploadCdn bool `json:"qiniu_upload_cdn,omitempty"`
		// ChunkConcurrency the number of chunks to upload concurrently.
		ChunkConcurrency int `json:"chunk_concurrency,omitempty"`
		// Whether to enable file encryption.
		Encryption bool `json:"encryption,omitempty"`
		// LoadBalancer 负载均衡存储策略配置：子策略 ID → 权重。
		// 仅对 PolicyType load_balance 生效。
		LoadBalancer *LoadBalancerConfig `json:"load_balancer,omitempty"`
	}

	// LoadBalancerConfig 负载均衡存储策略配置。
	LoadBalancerConfig struct {
		// Weights 子存储策略 ID → 权重。
		Weights map[int]int `json:"weights,omitempty"`
	}

	FileType         int
	EntityType       int
	GroupPermission  int
	FilePermission   int
	DavAccountOption int
	NodeCapability   int

	NodeSetting struct {
		Provider            DownloaderProvider `json:"provider,omitempty"`
		*QBittorrentSetting `json:"qbittorrent,omitempty"`
		*Aria2Setting       `json:"aria2,omitempty"`
		// 下载监控间隔
		Interval       int  `json:"interval,omitempty"`
		WaitForSeeding bool `json:"wait_for_seeding,omitempty"`
		// URLValidation controls SSRF policy applied to user-supplied URLs
		// fetched by this node's downloader. nil means the secure default
		// (validation on, no extra allowlist) — existing nodes upgraded in
		// place stay protected without admin action.
		URLValidation *URLValidationSetting `json:"url_validation,omitempty"`
	}

	URLValidationSetting struct {
		// Disabled turns the SSRF check off entirely on this node. Only set
		// this when the downloader runs in a network segment that cannot
		// reach any internal asset (e.g. dedicated egress namespace).
		Disabled bool `json:"disabled,omitempty"`
		// AllowedHosts is a list of hostnames or IP literals that bypass all
		// checks. Exact, case-insensitive match against url.Hostname().
		AllowedHosts []string `json:"allowed_hosts,omitempty"`
		// AllowedCIDRs is a list of CIDR blocks (IPv4 or IPv6) whose IPs are
		// treated as safe even if they would otherwise be rejected (private,
		// link-local, etc.). Use this to whitelist a LAN range like
		// "192.168.10.0/24" so a local NAS can be fetched.
		AllowedCIDRs []string `json:"allowed_cidrs,omitempty"`
	}

	DownloaderProvider string

	QBittorrentSetting struct {
		Server   string         `json:"server,omitempty"`
		User     string         `json:"user,omitempty"`
		Password string         `json:"password,omitempty"`
		Options  map[string]any `json:"options,omitempty"`
		TempPath string         `json:"temp_path,omitempty"`
	}

	Aria2Setting struct {
		Server   string         `json:"server,omitempty"`
		Token    string         `json:"token,omitempty"`
		Options  map[string]any `json:"options,omitempty"`
		TempPath string         `json:"temp_path,omitempty"`
	}

	TaskPublicState struct {
		Error            string          `json:"error,omitempty"`
		ErrorHistory     []string        `json:"error_history,omitempty"`
		ExecutedDuration time.Duration   `json:"executed_duration,omitempty"`
		RetryCount       int             `json:"retry_count,omitempty"`
		ResumeTime       int64           `json:"resume_time,omitempty"`
		SlaveTaskProps   *SlaveTaskProps `json:"slave_task_props,omitempty"`
	}

	SlaveTaskProps struct {
		NodeID            int    `json:"node_id,omitempty"`
		MasterSiteURl     string `json:"master_site_u_rl,omitempty"`
		MasterSiteID      string `json:"master_site_id,omitempty"`
		MasterSiteVersion string `json:"master_site_version,omitempty"`
	}

	EntityProps struct {
		UnlinkOnly      bool             `json:"unlink_only,omitempty"`
		EncryptMetadata *EncryptMetadata `json:"encrypt_metadata,omitempty"`
	}

	Cipher string

	EncryptMetadata struct {
		Algorithm    Cipher `json:"algorithm"`
		Key          []byte `json:"key"`
		KeyPlainText []byte `json:"key_plain_text,omitempty"`
		IV           []byte `json:"iv"`
	}

	DavAccountProps struct {
	}

	PolicyType string

	FileProps struct {
		View *ExplorerView `json:"view,omitempty"`
		// PreferredStoragePolicyID sets a per-directory/per-file storage policy.
		// New uploads under this file inherit the preferred policy.
		PreferredStoragePolicyID int `json:"preferred_storage_policy_id,omitempty"`
		// Permissions holds per-file permission overrides (PRO file permission management).
		Permissions *FileAccessRule `json:"permissions,omitempty"`
	}

	// FileAccessRule 文件级权限管理（PRO）。空字段表示继承默认/未设置。
	FileAccessRule struct {
		// AllowUsers 指定用户（user id）访问白名单
		AllowUsers []int `json:"allow_users,omitempty"`
		// DenyUsers 指定用户（user id）访问黑名单
		DenyUsers []int `json:"deny_users,omitempty"`
		// AllowGroups 指定用户组（group id）访问白名单
		AllowGroups []int `json:"allow_groups,omitempty"`
		// DenyGroups 指定用户组（group id）访问黑名单
		DenyGroups []int `json:"deny_groups,omitempty"`
		// Anonymous 匿名用户访问级别：0=inherit,1=view,2=download,3=write
		Anonymous int `json:"anonymous,omitempty"`
	}

	ExplorerView struct {
		PageSize       int              `json:"page_size" binding:"min=50"`
		Order          string           `json:"order,omitempty" binding:"max=255"`
		OrderDirection string           `json:"order_direction,omitempty" binding:"eq=asc|eq=desc"`
		View           string           `json:"view,omitempty" binding:"eq=list|eq=grid|eq=gallery"`
		Thumbnail      bool             `json:"thumbnail,omitempty"`
		GalleryWidth   int              `json:"gallery_width,omitempty" binding:"min=50,max=500"`
		Columns        []ListViewColumn `json:"columns,omitempty" binding:"max=1000"`
	}

	ListViewColumn struct {
		Type  int             `json:"type" binding:"min=0"`
		Width *int            `json:"width,omitempty"`
		Props *ColumTypeProps `json:"props,omitempty"`
	}

	ColumTypeProps struct {
		MetadataKey   string `json:"metadata_key,omitempty" binding:"max=255"`
		CustomPropsID string `json:"custom_props_id,omitempty" binding:"max=255"`
	}

	ShareProps struct {
		// Whether to share view setting from owner
		ShareView bool `json:"share_view,omitempty"`
		// Whether to automatically show readme file in share view
		ShowReadMe bool `json:"show_read_me,omitempty"`
		// Price in credits to access this share (0 = free)
		Price int `json:"price,omitempty"`
		// AllowUpload allows visitors to upload files via the share link
		AllowUpload bool `json:"allow_upload,omitempty"`
		// AllowModify allows visitors to modify files via the share link
		AllowModify bool `json:"allow_modify,omitempty"`
		// AllowDelete allows visitors to delete files via the share link
		AllowDelete bool `json:"allow_delete,omitempty"`
		// AllowAnonymousUpload allows anonymous visitors to upload via the share link
		AllowAnonymousUpload bool `json:"allow_anonymous_upload,omitempty"`
		// PurchasedUsers lists the user IDs who have purchased access to this share.
		PurchasedUsers []int `json:"purchased_users,omitempty"`
	}

	OAuthClientProps struct {
		Description     string `json:"description,omitempty"`
		Icon            string `json:"icon,omitempty"`
		RefreshTokenTTL int64  `json:"refresh_token_ttl,omitempty"` // in seconds, 0 means default
	}

	FileTypeIconSetting struct {
		Exts      []string `json:"exts"`
		Icon      string   `json:"icon,omitempty"`
		Color     string   `json:"color,omitempty"`
		ColorDark string   `json:"color_dark,omitempty"`
		Img       string   `json:"img,omitempty"`
	}

	// AuditContent 操作日志/审计事件的附加内容，与前端 LogEntry 对齐。
	AuditContent struct {
		Category          int               `json:"category,omitempty"`
		Failed            bool              `json:"failed,omitempty"`
		Error             string            `json:"error,omitempty"`
		UserAgent         string            `json:"user_agent,omitempty"`
		IsSystem          bool              `json:"is_system,omitempty"`
		Reason            string            `json:"reason,omitempty"`
		EmailTo           string            `json:"email_to,omitempty"`
		EmailTitle        string            `json:"email_title,omitempty"`
		OriginalName      string            `json:"original_name,omitempty"`
		NewName           string            `json:"new_name,omitempty"`
		From              string            `json:"from,omitempty"`
		To                string            `json:"to,omitempty"`
		EntityCreateTime  string            `json:"entity_create_time,omitempty"`
		StoragePolicyID   string            `json:"storage_policy_id,omitempty"`
		StoragePolicyName string            `json:"storage_policy_name,omitempty"`
		AccountID         int               `json:"account_id,omitempty"`
		Account           string            `json:"account,omitempty"`
		AccountURI        string            `json:"account_uri,omitempty"`
		PaymentID         int               `json:"payment_id,omitempty"`
		PointsChange      int               `json:"points_change,omitempty"`
		Sku               string            `json:"sku,omitempty"`
		StorageSize       int64             `json:"storage_size,omitempty"`
		Expire            string            `json:"expire,omitempty"`
		GroupID           int               `json:"group_id,omitempty"`
		GroupIDFrom       int               `json:"group_id_from,omitempty"`
		DirectLinkID      string            `json:"direct_link_id,omitempty"`
		OpenIDProvider    int               `json:"openid_provider,omitempty"`
		Sub               string            `json:"sub,omitempty"`
		Name              string            `json:"name,omitempty"`
		PasskeyID         int               `json:"passkey_id,omitempty"`
		Exts              map[string]string `json:"exts,omitempty"`
	}

	// ProductProps 增值服务商品属性（存储套餐/会员套餐/积分商品）。
	ProductProps struct {
		// Size 存储套餐容量（字节）
		Size int64 `json:"size,omitempty"`
		// DurationDays 有效期（天），0 表示永久
		DurationDays int `json:"duration_days,omitempty"`
		// GroupID 会员套餐升级到的用户组
		GroupID int `json:"group_id,omitempty"`
		// CreditAmount 积分商品对应的积分数量
		CreditAmount int `json:"credit_amount,omitempty"`
		// Description 商品描述，多行
		Description []string `json:"description,omitempty"`
		// PriceCredits 可用积分购买的价格，0 表示不能用积分购买
		PriceCredits int `json:"price_credits,omitempty"`
	}

	// GiftCodeProps 兑换码附加属性。
	GiftCodeProps struct {
		// LinkedProduct 对应商品
		LinkedProduct int `json:"linked_product,omitempty"`
		// ProductQty 商品数量（积分类为积分数量，其他为时长倍数）
		ProductQty int `json:"product_qty,omitempty"`
	}
)

const (
	GroupPermissionIsAdmin = GroupPermission(iota)
	GroupPermissionIsAnonymous
	GroupPermissionShare
	GroupPermissionWebDAV
	GroupPermissionArchiveDownload
	GroupPermissionArchiveTask
	GroupPermissionWebDAVProxy
	GroupPermissionShareDownload
	GroupPermissionShareFree
	GroupPermissionRemoteDownload
	GroupPermissionFolderDirectLink
	GroupPermissionRedirectedSource // not used
	GroupPermissionAdvanceDelete
	GroupPermissionEscalateAnonymity
	GroupPermissionMigratePolicy
	GroupPermissionAllowSelectNode
	GroupPermissionIgnoreFileOwnership // not used
	GroupPermissionUniqueRedirectDirectLink
)

const (
	NodeCapabilityNone NodeCapability = iota
	NodeCapabilityCreateArchive
	NodeCapabilityExtractArchive
	NodeCapabilityRemoteDownload
	NodeCapability_CommunityPlaceholder
)

const (
	FileTypeFile FileType = iota
	FileTypeFolder
)

const (
	EntityTypeVersion EntityType = iota
	EntityTypeThumbnail
	EntityTypeLivePhoto
)

func FileTypeFromString(s string) FileType {
	switch s {
	case "file":
		return FileTypeFile
	case "folder":
		return FileTypeFolder
	}
	return -1
}

const (
	DavAccountReadOnly DavAccountOption = iota
	DavAccountProxy
	DavAccountDisableSysFiles
)

const (
	PolicyTypeLocal       = "local"
	PolicyTypeQiniu       = "qiniu"
	PolicyTypeUpyun       = "upyun"
	PolicyTypeOss         = "oss"
	PolicyTypeCos         = "cos"
	PolicyTypeS3          = "s3"
	PolicyTypeKs3         = "ks3"
	PolicyTypeOd          = "onedrive"
	PolicyTypeRemote      = "remote"
	PolicyTypeObs         = "obs"
	PolicyTypeLoadBalance = "load_balance"
)

const (
	DownloaderProviderAria2       = DownloaderProvider("aria2")
	DownloaderProviderQBittorrent = DownloaderProvider("qbittorrent")
)

type (
	ViewerAction string
	ViewerType   string
)

const (
	ViewerActionView = "view"
	ViewerActionEdit = "edit"

	ViewerTypeBuiltin = "builtin"
	ViewerTypeWopi    = "wopi"
	ViewerTypeCustom  = "custom"
)

type (
	Viewer struct {
		ID                      string                             `json:"id"`
		Type                    ViewerType                         `json:"type"`
		DisplayName             string                             `json:"display_name"`
		Exts                    []string                           `json:"exts"`
		Url                     string                             `json:"url,omitempty"`
		Icon                    string                             `json:"icon,omitempty"`
		WopiActions             map[string]map[ViewerAction]string `json:"wopi_actions,omitempty"`
		Props                   map[string]string                  `json:"props,omitempty"`
		MaxSize                 int64                              `json:"max_size,omitempty"`
		Disabled                bool                               `json:"disabled,omitempty"`
		Templates               []NewFileTemplate                  `json:"templates,omitempty"`
		Platform                string                             `json:"platform,omitempty"`
		RequiredGroupPermission []GroupPermission                  `json:"required_group_permission,omitempty"`
	}
	ViewerGroup struct {
		Viewers []Viewer `json:"viewers"`
	}

	DefaultViewerMapping map[string]string

	NewFileTemplate struct {
		Ext         string `json:"ext"`
		DisplayName string `json:"display_name"`
	}
)

type (
	CustomPropsType string
	CustomProps     struct {
		ID      string          `json:"id"`
		Name    string          `json:"name"`
		Type    CustomPropsType `json:"type"`
		Max     int             `json:"max,omitempty"`
		Min     int             `json:"min,omitempty"`
		Default string          `json:"default,omitempty"`
		Options []string        `json:"options,omitempty"`
		Icon    string          `json:"icon,omitempty"`
	}
)

const (
	CustomPropsTypeText        = "text"
	CustomPropsTypeNumber      = "number"
	CustomPropsTypeBoolean     = "boolean"
	CustomPropsTypeSelect      = "select"
	CustomPropsTypeMultiSelect = "multi_select"
	CustomPropsTypeLink        = "link"
	CustomPropsTypeRating      = "rating"
)

const (
	ProfilePublicShareOnly = ShareLinksInProfileLevel("")
	ProfileAllShare        = ShareLinksInProfileLevel("all_share")
	ProfileHideShare       = ShareLinksInProfileLevel("hide_share")
)

const (
	CipherAES256CTR Cipher = "aes-256-ctr"
)

const (
	ScopeProfile               = "profile"
	ScopeEmail                 = "email"
	ScopeOpenID                = "openid"
	ScopeOfflineAccess         = "offline_access"
	ScopeUserInfoRead          = "UserInfo.Read"
	ScopeUserInfoWrite         = "UserInfo.Write"
	ScopeUserSecurityInfoRead  = "UserSecurityInfo.Read"
	ScopeUserSecurityInfoWrite = "UserSecurityInfo.Write"
	ScopeWorkflowRead          = "Workflow.Read"
	ScopeWorkflowWrite         = "Workflow.Write"
	ScopeAdminRead             = "Admin.Read"
	ScopeAdminWrite            = "Admin.Write"
	ScopeFilesRead             = "Files.Read"
	ScopeFilesWrite            = "Files.Write"
	ScopeSharesRead            = "Shares.Read"
	ScopeSharesWrite           = "Shares.Write"
	ScopeFinanceRead           = "Finance.Read"
	ScopeFinanceWrite          = "Finance.Write"
	ScopeDavAccountRead        = "DavAccount.Read"
	ScopeDavAccountWrite       = "DavAccount.Write"
)

// ProductType 增值服务商品类型。
type ProductType string

const (
	ProductTypeStorage ProductType = "storage_pack" // 存储套餐
	ProductTypeGroup   ProductType = "group"        // 会员套餐（升级用户组）
	ProductTypeCredit  ProductType = "credit"       // 积分商品
)

// OrderStatus 订单状态。
type OrderStatus string

const (
	OrderStatusUnpaid    OrderStatus = "unpaid"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusFulfilled OrderStatus = "fulfilled"
	OrderStatusFailed    OrderStatus = "failed"
)

// AbuseStatus 举报处理状态。
type AbuseStatus string

const (
	AbuseStatusPending  AbuseStatus = "pending"
	AbuseStatusResolved AbuseStatus = "resolved"
	AbuseStatusIgnored  AbuseStatus = "ignored"
)

// PaymentProvider 支付渠道。
type PaymentProvider string

const (
	PaymentProviderAlipay  PaymentProvider = "alipay"
	PaymentProviderWechat  PaymentProvider = "wechat"
	PaymentProviderPayJS   PaymentProvider = "payjs"
	PaymentProviderCustom  PaymentProvider = "custom"
	PaymentProviderCredits PaymentProvider = "credits"
)

// AuditType 操作日志类型，与前端 explorer.ts 的 AuditLogType 枚举一致。
type AuditType int

const (
	AuditTypeServerStart           AuditType = 0
	AuditTypeUserSignup            AuditType = 1
	AuditTypeEmailSent             AuditType = 2
	AuditTypeUserActivated         AuditType = 3
	AuditTypeUserLoginFailed       AuditType = 4
	AuditTypeUserLogin             AuditType = 5
	AuditTypeUserTokenRefresh      AuditType = 6
	AuditTypeFileCreate            AuditType = 7
	AuditTypeFileRename            AuditType = 8
	AuditTypeSetFilePermission     AuditType = 9
	AuditTypeEntityUploaded        AuditType = 10
	AuditTypeEntityDownloaded      AuditType = 11
	AuditTypeCopyFrom              AuditType = 12
	AuditTypeCopyTo                AuditType = 13
	AuditTypeMoveTo                AuditType = 14
	AuditTypeDeleteFile            AuditType = 15
	AuditTypeMoveToTrash           AuditType = 16
	AuditTypeShare                 AuditType = 17
	AuditTypeShareLinkViewed       AuditType = 18
	AuditTypeSetCurrentVersion     AuditType = 19
	AuditTypeDeleteVersion         AuditType = 20
	AuditTypeThumbGenerated        AuditType = 21
	AuditTypeLivePhotoUploaded     AuditType = 22
	AuditTypeUpdateMetadata        AuditType = 23
	AuditTypeEditShare             AuditType = 24
	AuditTypeDeleteShare           AuditType = 25
	AuditTypeMount                 AuditType = 26
	AuditTypeRelocate              AuditType = 27
	AuditTypeCreateArchive         AuditType = 28
	AuditTypeExtractArchive        AuditType = 29
	AuditTypeWebdavLoginFailed     AuditType = 30
	AuditTypeWebdavAccountCreate   AuditType = 31
	AuditTypeWebdavAccountUpdate   AuditType = 32
	AuditTypeWebdavAccountDelete   AuditType = 33
	AuditTypePaymentCreated        AuditType = 34
	AuditTypePointsChange          AuditType = 35
	AuditTypePaymentPaid           AuditType = 36
	AuditTypePaymentFulfilled      AuditType = 37
	AuditTypePaymentFulfillFailed  AuditType = 38
	AuditTypeStorageAdded          AuditType = 39
	AuditTypeGroupChanged          AuditType = 40
	AuditTypeUserExceedQuota       AuditType = 41
	AuditTypeUserChanged           AuditType = 42
	AuditTypeGetDirectLink         AuditType = 43
	AuditTypeLinkAccount           AuditType = 44
	AuditTypeUnlinkAccount         AuditType = 45
	AuditTypeChangeNick            AuditType = 46
	AuditTypeChangeAvatar          AuditType = 47
	AuditTypeMembershipUnsubscribe AuditType = 48
	AuditTypeChangePassword        AuditType = 49
	AuditTypeEnable2FA             AuditType = 50
	AuditTypeDisable2FA            AuditType = 51
	AuditTypeAddPasskey            AuditType = 52
	AuditTypeRemovePasskey         AuditType = 53
	AuditTypeRedeemGiftCode        AuditType = 54
	AuditTypeFileImported          AuditType = 55
	AuditTypeUpdateView            AuditType = 56
	AuditTypeDeleteDirectLink      AuditType = 57
	AuditTypeReportAbuse           AuditType = 58
	AuditTypeOAuthGrantCreate      AuditType = 59
	AuditTypeOAuthTokenExchange    AuditType = 60
	AuditTypeOAuthGrantRevoke      AuditType = 61
)
