package inventory

import (
	"context"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/giftcode"
	"github.com/cloudreve/Cloudreve/v4/ent/schema"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/conf"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
)

var (
	ErrGiftCodeNotFound = &GiftCodeError{"gift code not found"}
	ErrGiftCodeUsed     = &GiftCodeError{"gift code already used"}
)

type GiftCodeError struct{ message string }

func (e *GiftCodeError) Error() string { return e.message }

type (
	GiftCodeClient interface {
		TxOperator
		// Create creates a batch of gift codes.
		Create(ctx context.Context, codes []*CreateGiftCodeParams) ([]*ent.GiftCode, error)
		// GetByCode returns a gift code by its code string.
		GetByCode(ctx context.Context, code string) (*ent.GiftCode, error)
		// List returns a page of gift codes.
		List(ctx context.Context, args *ListGiftCodeArgs) (*ListGiftCodeResult, error)
		// Redeem marks a gift code as used by the given user.
		Redeem(ctx context.Context, id, uid int) (*ent.GiftCode, error)
		// Delete removes gift codes.
		Delete(ctx context.Context, ids []int) error
		// Count counts gift codes (optionally by usage state).
		Count(ctx context.Context, used *bool) (int, error)
	}

	CreateGiftCodeParams struct {
		Code     string
		Props    *types.GiftCodeProps
	}

	ListGiftCodeArgs struct {
		*PaginationArgs
		CodeIDs    []int
		UsedOnly   bool
		UnusedOnly bool
	}
	ListGiftCodeResult struct {
		*PaginationResults
		Codes []*ent.GiftCode
	}
)

func NewGiftCodeClient(client *ent.Client, dbType conf.DBType, hasher hashid.Encoder) GiftCodeClient {
	return &giftCodeClient{client: client, hasher: hasher, maxSQlParam: sqlParamLimit(dbType)}
}

type giftCodeClient struct {
	maxSQlParam int
	client      *ent.Client
	hasher      hashid.Encoder
}

func (c *giftCodeClient) SetClient(newClient *ent.Client) TxOperator {
	return &giftCodeClient{client: newClient, hasher: c.hasher, maxSQlParam: c.maxSQlParam}
}

func (c *giftCodeClient) GetClient() *ent.Client {
	return c.client
}

func (c *giftCodeClient) Create(ctx context.Context, codes []*CreateGiftCodeParams) ([]*ent.GiftCode, error) {
	stms := make([]*ent.GiftCodeCreate, 0, len(codes))
	for _, cd := range codes {
		q := c.client.GiftCode.Create().SetCode(cd.Code)
		if cd.Props != nil {
			q.SetProps(cd.Props)
		}
		stms = append(stms, q)
	}
	return c.client.GiftCode.CreateBulk(stms...).Save(ctx)
}

func (c *giftCodeClient) GetByCode(ctx context.Context, code string) (*ent.GiftCode, error) {
	return c.client.GiftCode.Query().Where(giftcode.Code(code)).Only(ctx)
}

func (c *giftCodeClient) List(ctx context.Context, args *ListGiftCodeArgs) (*ListGiftCodeResult, error) {
	q := c.client.GiftCode.Query()
	if len(args.CodeIDs) > 0 {
		q = q.Where(giftcode.IDIn(args.CodeIDs...))
	}
	if args.UsedOnly {
		q = q.Where(giftcode.UsedByGT(0))
	}
	if args.UnusedOnly {
		q = q.Where(giftcode.UsedAtIsNil())
	}

	pageSize := capPageSize(c.maxSQlParam, args.PageSize, 1)
	if args.UseCursorPagination && args.PageToken != "" {
		token, err := pageTokenFromString(args.PageToken, c.hasher, hashid.GiftCodeID)
		if err != nil {
			return nil, err
		}
		if token.ID != 0 {
			q = q.Where(giftcode.IDLT(token.ID))
		}
	}

	codes, err := q.Limit(pageSize).All(ctx)
	if err != nil {
		return nil, err
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	res := &ListGiftCodeResult{Codes: codes, PaginationResults: &PaginationResults{TotalItems: total, PageSize: pageSize}}
	if len(codes) >= pageSize {
		last := codes[len(codes)-1]
		token := &PageToken{ID: last.ID, Time: &last.CreatedAt}
		if s, err := token.Encode(c.hasher, hashid.EncodeGiftCodeID); err == nil {
			res.NextPageToken = s
		}
	}
	return res, nil
}

func (c *giftCodeClient) Redeem(ctx context.Context, id, uid int) (*ent.GiftCode, error) {
	now := time.Now()
	code, err := c.client.GiftCode.UpdateOneID(id).
		SetUsedBy(uid).
		SetUsedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return code, nil
}

func (c *giftCodeClient) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := c.client.GiftCode.Delete().Where(giftcode.IDIn(ids...)).Exec(schema.SkipSoftDelete(ctx))
	return err
}

func (c *giftCodeClient) Count(ctx context.Context, used *bool) (int, error) {
	q := c.client.GiftCode.Query()
	if used != nil {
		if *used {
			q = q.Where(giftcode.UsedByGT(0))
		} else {
			q = q.Where(giftcode.UsedAtIsNil())
		}
	}
	return q.Count(ctx)
}