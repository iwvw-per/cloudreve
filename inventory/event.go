package inventory

import (
	"context"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/event"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
)

type (
	// LoadEventUser / LoadEventFile / LoadEventEntity / LoadEventShare are
	// eager-loading ctx keys for the Event edges.
	LoadEventUser   struct{}
	LoadEventFile   struct{}
	LoadEventEntity struct{}
	LoadEventShare  struct{}
)

type (
	EventClient interface {
		TxOperator
		// Create records an audit event.
		Create(ctx context.Context, e *CreateEventParams) (*ent.Event, error)
		// GetByID returns the event with given id.
		GetByID(ctx context.Context, id int) (*ent.Event, error)
		// List returns a page of audit events.
		List(ctx context.Context, args *ListEventArgs) (*ListEventResult, error)
		// Delete permanently removes the given events.
		Delete(ctx context.Context, ids []int) error
		// Cleanup removes events that match the filtering criteria.
		Cleanup(ctx context.Context, args *CleanupEventArgs) (int, error)
		// CountByTimeRange counts audit events in the given time range.
		CountByTimeRange(ctx context.Context, start, end *time.Time) (int, error)
	}

	CreateEventParams struct {
		Type          types.AuditType
		CorrelationID string
		IP            string
		UserAgent     string
		Content       *types.AuditContent
		UserID        int
		FileID        int
		EntityID      int
		ShareID       int
	}

	ListEventArgs struct {
		*PaginationArgs
		Types    []int
		UserID   int
		EventIDs []int
	}
	ListEventResult struct {
		*PaginationResults
		Events []*ent.Event
	}

	CleanupEventArgs struct {
		NotAfter *time.Time
		Types    []int
	}
)

func NewEventClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) EventClient {
	return &eventClient{client: client, hasher: hasher, maxSQlParam: sqlParamLimit(dbType)}
}

type eventClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *eventClient) SetClient(newClient *ent.Client) TxOperator {
	return &eventClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *eventClient) GetClient() *ent.Client {
	return c.client
}

func (c *eventClient) Create(ctx context.Context, e *CreateEventParams) (*ent.Event, error) {
	q := c.client.Event.
		Create().
		SetType(int(e.Type))
	if e.CorrelationID != "" {
		q.SetCorrelationID(e.CorrelationID)
	}
	if e.IP != "" {
		q.SetIP(e.IP)
	}
	if e.UserAgent != "" {
		q.SetUserAgent(e.UserAgent)
	}
	if e.Content != nil {
		q.SetContent(e.Content)
	}
	if e.UserID != 0 {
		q.SetUserEvents(e.UserID)
	}
	if e.FileID != 0 {
		q.SetFileEvents(e.FileID)
	}
	if e.EntityID != 0 {
		q.SetEntityEvents(e.EntityID)
	}
	if e.ShareID != 0 {
		q.SetShareEvents(e.ShareID)
	}
	return q.Save(ctx)
}

func (c *eventClient) GetByID(ctx context.Context, id int) (*ent.Event, error) {
	return c.client.Event.Query().Where(event.ID(id)).WithUser().WithFile().WithEntity().WithShare().Only(ctx)
}

func (c *eventClient) CountByTimeRange(ctx context.Context, start, end *time.Time) (int, error) {
	if start == nil || end == nil {
		return c.client.Event.Query().Count(ctx)
	}
	return c.client.Event.Query().Where(event.CreatedAtGTE(*start), event.CreatedAtLT(*end)).Count(ctx)
}

func (c *eventClient) List(ctx context.Context, args *ListEventArgs) (*ListEventResult, error) {
	q := c.client.Event.Query()
	if len(args.Types) > 0 {
		q = q.Where(event.TypeIn(args.Types...))
	}
	if args.UserID != 0 {
		q = q.Where(event.UserEvents(args.UserID))
	}
	if len(args.EventIDs) > 0 {
		q = q.Where(event.IDIn(args.EventIDs...))
	}
	if _, ok := ctx.Value(LoadEventUser{}).(bool); ok {
		q = q.WithUser()
	}
	if _, ok := ctx.Value(LoadEventFile{}).(bool); ok {
		q = q.WithFile()
	}
	if _, ok := ctx.Value(LoadEventEntity{}).(bool); ok {
		q = q.WithEntity()
	}
	if _, ok := ctx.Value(LoadEventShare{}).(bool); ok {
		q = q.WithShare()
	}

	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 1)
	if args.UseCursorPagination && args.PageToken != "" {
		token, err := pageTokenFromString(args.PageToken, c.hasher, hashid.AuditLogID)
		if err != nil {
			return nil, err
		}
		if token.ID != 0 {
			q = q.Where(event.IDLT(token.ID))
		}
	}

	events, err := q.Limit(pageSize).All(ctx)
	if err != nil {
		return nil, err
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	res := &ListEventResult{Events: events, PaginationResults: &PaginationResults{TotalItems: total, PageSize: pageSize}}
	if len(events) >= pageSize {
		last := events[len(events)-1]
		token := &PageToken{ID: last.ID, Time: &last.CreatedAt}
		if s, err := token.Encode(c.hasher, hashid.EncodeEventID); err == nil {
			res.NextPageToken = s
		}
	}
	return res, nil
}

func (c *eventClient) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := c.client.Event.Delete().Where(event.IDIn(ids...)).Exec(schema.SkipSoftDelete(ctx))
	return err
}

func (c *eventClient) Cleanup(ctx context.Context, args *CleanupEventArgs) (int, error) {
	q := c.client.Event.Delete()
	if args.NotAfter != nil {
		q = q.Where(event.CreatedAtLT(*args.NotAfter))
	}
	if len(args.Types) > 0 {
		q = q.Where(event.TypeIn(args.Types...))
	}
	return q.Exec(schema.SkipSoftDelete(ctx))
}