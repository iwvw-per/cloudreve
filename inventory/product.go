package inventory

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/product"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
)

type (
	ProductClient interface {
		TxOperator
		// Create creates a new product.
		Create(ctx context.Context, p *CreateProductParams) (*ent.Product, error)
		// GetByID returns a product by its id.
		GetByID(ctx context.Context, id int) (*ent.Product, error)
		// GetByIDs returns products by their ids.
		GetByIDs(ctx context.Context, ids []int) ([]*ent.Product, error)
		// List returns a page of products.
		List(ctx context.Context, args *ListProductArgs) (*ListProductResult, error)
		// Update updates an existing product.
		Update(ctx context.Context, id int, p *CreateProductParams) (*ent.Product, error)
		// Delete removes products.
		Delete(ctx context.Context, ids []int) error
		// Count counts products of the given types.
		Count(ctx context.Context, productTypes []types.ProductType) (int, error)
	}

	CreateProductParams struct {
		Name       string
		Type       types.ProductType
		Price      int
		Highlight  bool
		Enabled    bool
		Props      *types.ProductProps
	}

	ListProductArgs struct {
		*PaginationArgs
		ProductIDs []int
		ProductTypes []types.ProductType
		EnabledOnly bool
	}
	ListProductResult struct {
		*PaginationResults
		Products []*ent.Product
	}
)

func NewProductClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) ProductClient {
	return &productClient{client: client, hasher: hasher, maxSQlParam: sqlParamLimit(dbType)}
}

type productClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *productClient) SetClient(newClient *ent.Client) TxOperator {
	return &productClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *productClient) GetClient() *ent.Client {
	return c.client
}

func (c *productClient) Create(ctx context.Context, p *CreateProductParams) (*ent.Product, error) {
	return c.buildMutation(c.client.Product.Create(), p).Save(ctx)
}

func (c *productClient) buildMutation(q *ent.ProductCreate, p *CreateProductParams) *ent.ProductCreate {
	q.SetName(p.Name).
		SetType(product.Type(p.Type)).
		SetPrice(p.Price).
		SetHighlight(p.Highlight).
		SetEnabled(p.Enabled)
	if p.Props != nil {
		q.SetProps(p.Props)
	}
	return q
}

func (c *productClient) GetByID(ctx context.Context, id int) (*ent.Product, error) {
	return c.client.Product.Query().Where(product.ID(id)).Only(ctx)
}

func (c *productClient) GetByIDs(ctx context.Context, ids []int) ([]*ent.Product, error) {
	if len(ids) == 0 {
		return []*ent.Product{}, nil
	}
	return c.client.Product.Query().Where(product.IDIn(ids...)).All(ctx)
}

func (c *productClient) List(ctx context.Context, args *ListProductArgs) (*ListProductResult, error) {
	q := c.client.Product.Query()
	if len(args.ProductIDs) > 0 {
		q = q.Where(product.IDIn(args.ProductIDs...))
	}
	if len(args.ProductTypes) > 0 {
		vals := make([]product.Type, 0, len(args.ProductTypes))
		for _, t := range args.ProductTypes {
			vals = append(vals, product.Type(t))
		}
		q = q.Where(product.TypeIn(vals...))
	}
	if args.EnabledOnly {
		q = q.Where(product.Enabled(true))
	}

	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 1)
	if args.UseCursorPagination && args.PageToken != "" {
		token, err := pageTokenFromString(args.PageToken, c.hasher, hashid.ProductID)
		if err != nil {
			return nil, err
		}
		if token.ID != 0 {
			q = q.Where(product.IDLT(token.ID))
		}
	}

	products, err := q.Limit(pageSize).All(ctx)
	if err != nil {
		return nil, err
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	res := &ListProductResult{Products: products, PaginationResults: &PaginationResults{TotalItems: total, PageSize: pageSize}}
	if len(products) >= pageSize {
		last := products[len(products)-1]
		token := &PageToken{ID: last.ID, Time: &last.CreatedAt}
		if s, err := token.Encode(c.hasher, hashid.EncodeProductID); err == nil {
			res.NextPageToken = s
		}
	}
	return res, nil
}

func (c *productClient) Update(ctx context.Context, id int, p *CreateProductParams) (*ent.Product, error) {
	return c.client.Product.UpdateOneID(id).
		SetName(p.Name).
		SetType(product.Type(p.Type)).
		SetPrice(p.Price).
		SetHighlight(p.Highlight).
		SetEnabled(p.Enabled).
		SetProps(p.Props).
		Save(ctx)
}

func (c *productClient) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := c.client.Product.Delete().Where(product.IDIn(ids...)).Exec(schema.SkipSoftDelete(ctx))
	return err
}

func (c *productClient) Count(ctx context.Context, productTypes []types.ProductType) (int, error) {
	q := c.client.Product.Query()
	if len(productTypes) > 0 {
		vals := make([]product.Type, 0, len(productTypes))
		for _, t := range productTypes {
			vals = append(vals, product.Type(t))
		}
		q = q.Where(product.TypeIn(vals...))
	}
	return q.Count(ctx)
}