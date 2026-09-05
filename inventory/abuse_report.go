package inventory

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/abusereport"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
)

type (
	// LoadAbuseReporter / LoadAbuseReported / LoadAbuseShare are eager-loading ctx keys.
	LoadAbuseReporter struct{}
	LoadAbuseReported struct{}
	LoadAbuseShare    struct{}
)

type (
	AbuseReportClient interface {
		TxOperator
		// Create creates a new abuse report.
		Create(ctx context.Context, a *CreateAbuseReportParams) (*ent.AbuseReport, error)
		// GetByID returns an abuse report by its id.
		GetByID(ctx context.Context, id int) (*ent.AbuseReport, error)
		// List returns a page of abuse reports.
		List(ctx context.Context, args *ListAbuseReportArgs) (*ListAbuseReportResult, error)
		// UpdateStatus sets the resolution status of a report.
		UpdateStatus(ctx context.Context, id int, status types.AbuseStatus) (*ent.AbuseReport, error)
		// Delete removes abuse reports.
		Delete(ctx context.Context, ids []int) error
		// Count counts abuse reports (optionally by status).
		Count(ctx context.Context, status *types.AbuseStatus) (int, error)
	}

	CreateAbuseReportParams struct {
		ReporterID   int
		ReportedID   int
		ShareID      int
		FolderPath   string
		Reason       string
		Description  string
		RawContent   *types.AuditContent
	}

	ListAbuseReportArgs struct {
		*PaginationArgs
		ReportIDs  []int
		ReporterID int
		ReportedID int
		Status     *types.AbuseStatus
	}
	ListAbuseReportResult struct {
		*PaginationResults
		Reports []*ent.AbuseReport
	}
)

func NewAbuseReportClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) AbuseReportClient {
	return &abuseReportClient{client: client, hasher: hasher, maxSQlParam: sqlParamLimit(dbType)}
}

type abuseReportClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *abuseReportClient) SetClient(newClient *ent.Client) TxOperator {
	return &abuseReportClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *abuseReportClient) GetClient() *ent.Client {
	return c.client
}

func (c *abuseReportClient) Create(ctx context.Context, a *CreateAbuseReportParams) (*ent.AbuseReport, error) {
	q := c.client.AbuseReport.
		Create().
		SetStatus(abusereport.Status(types.AbuseStatusPending))
	if a.ReporterID != 0 {
		q.SetReporterUser(a.ReporterID)
	}
	if a.ReportedID != 0 {
		q.SetReportedUser(a.ReportedID)
	}
	if a.ShareID != 0 {
		q.SetShareReports(a.ShareID)
	}
	if a.FolderPath != "" {
		q.SetFolderPath(a.FolderPath)
	}
	if a.Reason != "" {
		q.SetReason(a.Reason)
	}
	if a.Description != "" {
		q.SetDescription(a.Description)
	}
	if a.RawContent != nil {
		q.SetRawContent(a.RawContent)
	}
	return q.Save(ctx)
}

func (c *abuseReportClient) GetByID(ctx context.Context, id int) (*ent.AbuseReport, error) {
	q := c.client.AbuseReport.Query().Where(abusereport.ID(id))
	if _, ok := ctx.Value(LoadAbuseReporter{}).(bool); ok {
		q = q.WithReporter()
	}
	if _, ok := ctx.Value(LoadAbuseReported{}).(bool); ok {
		q = q.WithReported()
	}
	if _, ok := ctx.Value(LoadAbuseShare{}).(bool); ok {
		q = q.WithShare()
	}
	return q.Only(ctx)
}

func (c *abuseReportClient) List(ctx context.Context, args *ListAbuseReportArgs) (*ListAbuseReportResult, error) {
	q := c.client.AbuseReport.Query()
	if len(args.ReportIDs) > 0 {
		q = q.Where(abusereport.IDIn(args.ReportIDs...))
	}
	if args.ReporterID != 0 {
		q = q.Where(abusereport.ReporterUser(args.ReporterID))
	}
	if args.ReportedID != 0 {
		q = q.Where(abusereport.ReportedUser(args.ReportedID))
	}
	if args.Status != nil {
		q = q.Where(abusereport.StatusEQ(abusereport.Status(*args.Status)))
	}
	if _, ok := ctx.Value(LoadAbuseReporter{}).(bool); ok {
		q = q.WithReporter()
	}
	if _, ok := ctx.Value(LoadAbuseReported{}).(bool); ok {
		q = q.WithReported()
	}
	if _, ok := ctx.Value(LoadAbuseShare{}).(bool); ok {
		q = q.WithShare()
	}

	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 1)
	if args.UseCursorPagination && args.PageToken != "" {
		token, err := pageTokenFromString(args.PageToken, c.hasher, hashid.AbuseReportID)
		if err != nil {
			return nil, err
		}
		if token.ID != 0 {
			q = q.Where(abusereport.IDLT(token.ID))
		}
	}

	reports, err := q.Limit(pageSize).All(ctx)
	if err != nil {
		return nil, err
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	res := &ListAbuseReportResult{Reports: reports, PaginationResults: &PaginationResults{TotalItems: total, PageSize: pageSize}}
	if len(reports) >= pageSize {
		last := reports[len(reports)-1]
		token := &PageToken{ID: last.ID, Time: &last.CreatedAt}
		if s, err := token.Encode(c.hasher, hashid.EncodeAbuseReportID); err == nil {
			res.NextPageToken = s
		}
	}
	return res, nil
}

func (c *abuseReportClient) UpdateStatus(ctx context.Context, id int, status types.AbuseStatus) (*ent.AbuseReport, error) {
	return c.client.AbuseReport.UpdateOneID(id).SetStatus(abusereport.Status(status)).Save(ctx)
}

func (c *abuseReportClient) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := c.client.AbuseReport.Delete().Where(abusereport.IDIn(ids...)).Exec(schema.SkipSoftDelete(ctx))
	return err
}

func (c *abuseReportClient) Count(ctx context.Context, status *types.AbuseStatus) (int, error) {
	q := c.client.AbuseReport.Query()
	if status != nil {
		q = q.Where(abusereport.StatusEQ(abusereport.Status(*status)))
	}
	return q.Count(ctx)
}