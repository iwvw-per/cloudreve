package inventory

import (
	"context"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/order"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
)

type (
	// LoadOrderUser is an eager-loading ctx key for the Order user edge.
	LoadOrderUser struct{}
)

type (
	OrderClient interface {
		TxOperator
		// Create creates a new order.
		Create(ctx context.Context, o *CreateOrderParams) (*ent.Order, error)
		// GetByID returns an order by its id.
		GetByID(ctx context.Context, id int) (*ent.Order, error)
		// GetByOrderNo returns an order by its unique order number.
		GetByOrderNo(ctx context.Context, orderNo string) (*ent.Order, error)
		// List returns a page of orders.
		List(ctx context.Context, args *ListOrderArgs) (*ListOrderResult, error)
		// UpdateStatus updates the order status/provider.
		UpdateStatus(ctx context.Context, id int, status types.OrderStatus, provider types.PaymentProvider) (*ent.Order, error)
		// MarkFulfilled marks an order as fulfilled.
		MarkFulfilled(ctx context.Context, id int) (*ent.Order, error)
		// Delete removes orders.
		Delete(ctx context.Context, ids []int) error
		// CountByUser counts orders of a user (optionally in time range).
		CountByUser(ctx context.Context, uid int, start, end *time.Time) (int, error)
	}

	CreateOrderParams struct {
		OrderNo     string
		UserID      int
		ProductType types.ProductType
		ProductID   int
		Quantity    int
		Amount      int
		Provider    types.PaymentProvider
		Content     *types.GiftCodeProps
	}

	ListOrderArgs struct {
		*PaginationArgs
		UserID     int
		ProductID  int
		OrderIDs   []int
		ProductTypes []types.ProductType
	}
	ListOrderResult struct {
		*PaginationResults
		Orders []*ent.Order
	}
)

func NewOrderClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) OrderClient {
	return &orderClient{client: client, hasher: hasher, maxSQlParam: sqlParamLimit(dbType)}
}

type orderClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *orderClient) SetClient(newClient *ent.Client) TxOperator {
	return &orderClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *orderClient) GetClient() *ent.Client {
	return c.client
}

func (c *orderClient) Create(ctx context.Context, o *CreateOrderParams) (*ent.Order, error) {
	q := c.client.Order.
		Create().
		SetOrderNo(o.OrderNo).
		SetProductType(order.ProductType(o.ProductType)).
		SetQuantity(o.Quantity).
		SetAmount(o.Amount).
		SetStatus(order.Status(types.OrderStatusUnpaid)).
		SetUserOrders(o.UserID)
	if o.ProductID != 0 {
		q.SetProductID(o.ProductID)
	}
	if o.Provider != "" {
		q.SetProvider(string(o.Provider))
	}
	if o.Content != nil {
		q.SetContent(o.Content)
	}
	return q.Save(ctx)
}

func (c *orderClient) GetByID(ctx context.Context, id int) (*ent.Order, error) {
	q := c.client.Order.Query().Where(order.ID(id))
	if _, ok := ctx.Value(LoadOrderUser{}).(bool); ok {
		q = q.WithUser()
	}
	return q.Only(ctx)
}

func (c *orderClient) GetByOrderNo(ctx context.Context, orderNo string) (*ent.Order, error) {
	return c.client.Order.Query().Where(order.OrderNo(orderNo)).Only(ctx)
}

func (c *orderClient) List(ctx context.Context, args *ListOrderArgs) (*ListOrderResult, error) {
	q := c.client.Order.Query()
	if args.UserID != 0 {
		q = q.Where(order.UserOrders(args.UserID))
	}
	if args.ProductID != 0 {
		q = q.Where(order.ProductID(args.ProductID))
	}
	if len(args.OrderIDs) > 0 {
		q = q.Where(order.IDIn(args.OrderIDs...))
	}
	if len(args.ProductTypes) > 0 {
		vals := make([]order.ProductType, 0, len(args.ProductTypes))
		for _, t := range args.ProductTypes {
			vals = append(vals, order.ProductType(t))
		}
		q = q.Where(order.ProductTypeIn(vals...))
	}
	if _, ok := ctx.Value(LoadOrderUser{}).(bool); ok {
		q = q.WithUser()
	}

	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 1)
	if args.UseCursorPagination && args.PageToken != "" {
		token, err := pageTokenFromString(args.PageToken, c.hasher, hashid.OrderID)
		if err != nil {
			return nil, err
		}
		if token.ID != 0 {
			q = q.Where(order.IDLT(token.ID))
		}
	}

	orders, err := q.Limit(pageSize).All(ctx)
	if err != nil {
		return nil, err
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	res := &ListOrderResult{Orders: orders, PaginationResults: &PaginationResults{TotalItems: total, PageSize: pageSize}}
	if len(orders) >= pageSize {
		last := orders[len(orders)-1]
		token := &PageToken{ID: last.ID, Time: &last.CreatedAt}
		if s, err := token.Encode(c.hasher, hashid.EncodeOrderID); err == nil {
			res.NextPageToken = s
		}
	}
	return res, nil
}

func (c *orderClient) UpdateStatus(ctx context.Context, id int, status types.OrderStatus, provider types.PaymentProvider) (*ent.Order, error) {
	q := c.client.Order.UpdateOneID(id).SetStatus(order.Status(status))
	if provider != "" {
		q.SetProvider(string(provider))
	}
	return q.Save(ctx)
}

func (c *orderClient) MarkFulfilled(ctx context.Context, id int) (*ent.Order, error) {
	return c.client.Order.UpdateOneID(id).SetStatus(order.Status(types.OrderStatusFulfilled)).Save(ctx)
}

func (c *orderClient) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := c.client.Order.Delete().Where(order.IDIn(ids...)).Exec(schema.SkipSoftDelete(ctx))
	return err
}

func (c *orderClient) CountByUser(ctx context.Context, uid int, start, end *time.Time) (int, error) {
	q := c.client.Order.Query().Where(order.UserOrders(uid))
	if start != nil && end != nil {
		q = q.Where(order.CreatedAtGTE(*start), order.CreatedAtLT(*end))
	}
	return q.Count(ctx)
}