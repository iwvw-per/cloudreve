package admin

import (
	"context"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const (
	abuseStatusCondition   = "status"
	abuseReporterCondition = "reporter_id"
	abuseReportedCondition = "reported_id"
)

type (
	// GetAbuseReportResponse 举报详情响应。
	GetAbuseReportResponse struct {
		Report         *ent.AbuseReport `json:"report"`
		ReporterHashID string           `json:"reporter_hash_id"`
		ReportedHashID string           `json:"reported_hash_id"`
		ShareHashID    string           `json:"share_hash_id"`
	}
	ListAbuseReportResponse struct {
		Pagination *inventory.PaginationResults `json:"pagination"`
		Reports    []GetAbuseReportResponse     `json:"reports"`
	}
)

func buildAbuseReportResponse(r *ent.AbuseReport, hasher hashid.Encoder) GetAbuseReportResponse {
	resp := GetAbuseReportResponse{Report: r}
	if r.Edges.Reporter != nil {
		resp.ReporterHashID = hashid.EncodeUserID(hasher, r.Edges.Reporter.ID)
	}
	if r.Edges.Reported != nil {
		resp.ReportedHashID = hashid.EncodeUserID(hasher, r.Edges.Reported.ID)
	}
	if r.Edges.Share != nil {
		resp.ShareHashID = hashid.EncodeShareID(hasher, r.Edges.Share.ID)
	}
	return resp
}

// AbuseReports 举报列表。
func (s *AdminListService) AbuseReports(c *gin.Context) (*ListAbuseReportResponse, error) {
	dep := dependency.FromContext(c)
	abuseClient := dep.AbuseReportClient()
	hasher := dep.HashIDEncoder()

	ctx := context.WithValue(c, inventory.LoadAbuseReporter{}, true)
	ctx = context.WithValue(ctx, inventory.LoadAbuseReported{}, true)
	ctx = context.WithValue(ctx, inventory.LoadAbuseShare{}, true)

	args := &inventory.ListAbuseReportArgs{
		PaginationArgs: &inventory.PaginationArgs{
			PageSize: s.PageSize,
			OrderBy:  s.OrderBy,
			Order:    inventory.OrderDirection(s.OrderDirection),
		},
	}
	if s.Conditions != nil {
		if s.Conditions[abuseStatusCondition] != "" {
			st := types.AbuseStatus(s.Conditions[abuseStatusCondition])
			args.Status = &st
		}
		if s.Conditions[abuseReporterCondition] != "" {
			args.ReporterID, _ = strconv.Atoi(s.Conditions[abuseReporterCondition])
		}
		if s.Conditions[abuseReportedCondition] != "" {
			args.ReportedID, _ = strconv.Atoi(s.Conditions[abuseReportedCondition])
		}
	}

	targetOffset := (s.Page - 1) * s.PageSize
	if targetOffset < 0 {
		targetOffset = 0
	}

	var (
		reports []*ent.AbuseReport
		total   int
	)
	args.UseCursorPagination = true
	for {
		res, err := abuseClient.List(ctx, args)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to list abuse reports", err)
		}
		total = res.TotalItems
		reports = append(reports, res.Reports...)
		if res.NextPageToken == "" || len(reports) >= targetOffset+s.PageSize {
			break
		}
		args.PageToken = res.NextPageToken
	}

	start := targetOffset
	if start > len(reports) {
		start = len(reports)
	}
	end := start + s.PageSize
	if end > len(reports) {
		end = len(reports)
	}

	return &ListAbuseReportResponse{
		Pagination: &inventory.PaginationResults{TotalItems: total, PageSize: s.PageSize},
		Reports: lo.Map(reports[start:end], func(r *ent.AbuseReport, _ int) GetAbuseReportResponse {
			return buildAbuseReportResponse(r, hasher)
		}),
	}, nil
}

type (
	// SingleAbuseReportService 举报详情服务。
	SingleAbuseReportService struct {
		ID int
	}
	SingleAbuseReportParamCtx struct{}
)

// Get 获取举报详情。
func (s *SingleAbuseReportService) Get(c *gin.Context) (*GetAbuseReportResponse, error) {
	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()

	ctx := context.WithValue(c, inventory.LoadAbuseReporter{}, true)
	ctx = context.WithValue(ctx, inventory.LoadAbuseReported{}, true)
	ctx = context.WithValue(ctx, inventory.LoadAbuseShare{}, true)

	report, err := dep.AbuseReportClient().GetByID(ctx, s.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get abuse report", err)
	}

	resp := buildAbuseReportResponse(report, hasher)
	return &resp, nil
}

type (
	// UpdateAbuseReportStatusService 标记举报处理状态。
	UpdateAbuseReportStatusService struct {
		ID     int               `json:"-"`
		Status types.AbuseStatus `json:"status" binding:"required"`
	}
	UpdateAbuseReportStatusParamCtx struct{}
)

// Update 更新举报状态。
func (s *UpdateAbuseReportStatusService) Update(c *gin.Context) (*GetAbuseReportResponse, error) {
	switch s.Status {
	case types.AbuseStatusPending, types.AbuseStatusResolved, types.AbuseStatusIgnored:
	default:
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid abuse report status", nil)
	}

	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()
	ctx := context.WithValue(c, inventory.LoadAbuseReporter{}, true)
	ctx = context.WithValue(ctx, inventory.LoadAbuseReported{}, true)
	ctx = context.WithValue(ctx, inventory.LoadAbuseShare{}, true)

	if _, err := dep.AbuseReportClient().UpdateStatus(c, s.ID, s.Status); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update abuse report", err)
	}

	report, err := dep.AbuseReportClient().GetByID(ctx, s.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get abuse report", err)
	}

	resp := buildAbuseReportResponse(report, hasher)
	return &resp, nil
}

type (
	// BatchAbuseReportService 批量删除举报。
	BatchAbuseReportService struct {
		IDs []int `json:"ids" binding:"required"`
	}
	BatchAbuseReportParamCtx struct{}
)

// Delete 批量删除举报。
func (s *BatchAbuseReportService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	if err := dep.AbuseReportClient().Delete(c, s.IDs); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete abuse reports", err)
	}
	return nil
}

type (
	// CreateAbuseReportService 用户举报分享。
	CreateAbuseReportService struct {
		ShareID     int    `json:"-"`
		Reason      string `json:"reason"`
		Description string `json:"description"`
	}
	CreateAbuseReportParamCtx struct{}
)

// Create 记录一条针对分享的举报。
func (s *CreateAbuseReportService) Create(c *gin.Context) error {
	dep := dependency.FromContext(c)

	reporter := inventory.UserFromContext(c)
	if reporter == nil || inventory.IsAnonymousUser(reporter) {
		return serializer.NewError(serializer.CodeCheckLogin, "Login required", nil)
	}
	if s.ShareID == 0 {
		return serializer.NewError(serializer.CodeParamErr, "Share ID is required", nil)
	}

	ctx := context.WithValue(c, inventory.LoadShareUser{}, true)
	share, err := dep.ShareClient().GetByID(ctx, s.ShareID)
	if err != nil {
		return serializer.NewError(serializer.CodeNotFound, "Share not found", err)
	}

	reportedID := 0
	if share.Edges.User != nil {
		reportedID = share.Edges.User.ID
	}

	if _, err := dep.AbuseReportClient().Create(c, &inventory.CreateAbuseReportParams{
		ReporterID:  reporter.ID,
		ReportedID:  reportedID,
		ShareID:     s.ShareID,
		Reason:      s.Reason,
		Description: s.Description,
	}); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to create abuse report", err)
	}

	return nil
}
