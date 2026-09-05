package admin

import (
	"context"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gin-gonic/gin"
)

// GetProductResponse 商品详情响应。
type GetProductResponse struct {
	*ent.Product
	HashID string `json:"hash_id,omitempty"`
}

// ListProductResponse 商品列表响应。
type ListProductResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Products   []GetProductResponse         `json:"products"`
}

// ListProductService 商品列表服务。
type (
	ListProductService struct {
		PageSize  int    `json:"page_size" binding:"min=1,max=100"`
		PageToken string `json:"page_token"`
		Type      string `json:"type"`
	}
	ListProductParamCtx struct{}
)

// List 获取商品列表。
func (s *ListProductService) List(c *gin.Context) (*ListProductResponse, error) {
	dep := dependency.FromContext(c)
	productClient := dep.ProductClient()
	hasher := dep.HashIDEncoder()

	args := &inventory.ListProductArgs{
		PaginationArgs: &inventory.PaginationArgs{
			PageSize:            s.PageSize,
			PageToken:           s.PageToken,
			UseCursorPagination: s.PageToken != "",
		},
	}
	if s.Type != "" {
		args.ProductTypes = []types.ProductType{types.ProductType(s.Type)}
	}

	res, err := productClient.List(c, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list products", err)
	}

	products := make([]GetProductResponse, 0, len(res.Products))
	for _, p := range res.Products {
		products = append(products, GetProductResponse{
			Product: p,
			HashID:  hashid.EncodeProductID(hasher, p.ID),
		})
	}

	return &ListProductResponse{
		Pagination: res.PaginationResults,
		Products:   products,
	}, nil
}

// UpsertProductService 商品新建/更新服务。
type (
	UpsertProductService struct {
		ID        int                 `json:"id"`
		Name      string              `json:"name" binding:"required"`
		Type      types.ProductType   `json:"type" binding:"required"`
		Price     int                 `json:"price"`
		Highlight bool                `json:"highlight"`
		Enabled   bool                `json:"enabled"`
		Props     *types.ProductProps `json:"props"`
	}
	UpsertProductParamCtx struct{}
)

func (s *UpsertProductService) buildParams() *inventory.CreateProductParams {
	return &inventory.CreateProductParams{
		Name:      s.Name,
		Type:      s.Type,
		Price:     s.Price,
		Highlight: s.Highlight,
		Enabled:   s.Enabled,
		Props:     s.Props,
	}
}

// Create 新建商品。
func (s *UpsertProductService) Create(c *gin.Context) (*GetProductResponse, error) {
	dep := dependency.FromContext(c)
	productClient := dep.ProductClient()
	hasher := dep.HashIDEncoder()

	p, err := productClient.Create(c, s.buildParams())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create product", err)
	}

	return &GetProductResponse{
		Product: p,
		HashID:  hashid.EncodeProductID(hasher, p.ID),
	}, nil
}

// Update 更新商品。
func (s *UpsertProductService) Update(c *gin.Context) (*GetProductResponse, error) {
	dep := dependency.FromContext(c)
	productClient := dep.ProductClient()
	hasher := dep.HashIDEncoder()

	if s.ID == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Product ID is required", nil)
	}

	p, err := productClient.Update(c, s.ID, s.buildParams())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update product", err)
	}

	return &GetProductResponse{
		Product: p,
		HashID:  hashid.EncodeProductID(hasher, p.ID),
	}, nil
}

// SingleProductService 商品ID服务。
type (
	SingleProductService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleProductParamCtx struct{}
)

// Get 获取商品详情。
func (s *SingleProductService) Get(c *gin.Context) (*GetProductResponse, error) {
	dep := dependency.FromContext(c)
	productClient := dep.ProductClient()
	hasher := dep.HashIDEncoder()

	p, err := productClient.GetByID(c, s.ID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Product not found", err)
	}

	return &GetProductResponse{
		Product: p,
		HashID:  hashid.EncodeProductID(hasher, p.ID),
	}, nil
}

// Delete 删除商品。
func (s *SingleProductService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	productClient := dep.ProductClient()

	if err := productClient.Delete(c, []int{s.ID}); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete product", err)
	}

	return nil
}

// ListGiftCodeResponse 兑换码列表响应。
type ListGiftCodeResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Codes      []GetGiftCodeResponse        `json:"gift_codes"`
}

// GetGiftCodeResponse 兑换码详情响应。
type GetGiftCodeResponse struct {
	*ent.GiftCode
	HashID string `json:"hash_id,omitempty"`
}

// CreateGiftCodeService 批量生成兑换码服务。
type (
	CreateGiftCodeService struct {
		Count         int `json:"count" binding:"required,min=1,max=1000"`
		LinkedProduct int `json:"linked_product"`
		ProductQty    int `json:"product_qty"`
	}
	CreateGiftCodeParamCtx struct{}
)

const giftCodeLength = 16

// Create 批量生成兑换码。
func (s *CreateGiftCodeService) Create(c *gin.Context) (*ListGiftCodeResponse, error) {
	dep := dependency.FromContext(c)
	giftCodeClient := dep.GiftCodeClient()
	hasher := dep.HashIDEncoder()

	codes := make([]*inventory.CreateGiftCodeParams, 0, s.Count)
	seen := make(map[string]struct{}, s.Count)
	for i := 0; i < s.Count; i++ {
		var code string
		for {
			code = strings.ToUpper(util.RandStringRunesCrypto(giftCodeLength))
			if _, dup := seen[code]; !dup {
				break
			}
		}
		seen[code] = struct{}{}

		props := &types.GiftCodeProps{
			LinkedProduct: s.LinkedProduct,
			ProductQty:    s.ProductQty,
		}
		codes = append(codes, &inventory.CreateGiftCodeParams{
			Code:  code,
			Props: props,
		})
	}

	created, err := giftCodeClient.Create(c, codes)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create gift codes", err)
	}

	result := make([]GetGiftCodeResponse, 0, len(created))
	for _, gc := range created {
		result = append(result, GetGiftCodeResponse{
			GiftCode: gc,
			HashID:   hashid.EncodeGiftCodeID(hasher, gc.ID),
		})
	}

	return &ListGiftCodeResponse{Codes: result}, nil
}

// ListGiftCodeService 兑换码列表服务。
type (
	ListGiftCodeService struct {
		PageSize   int    `json:"page_size" binding:"min=1,max=100"`
		PageToken  string `json:"page_token"`
		UsedOnly   bool   `json:"used_only"`
		UnusedOnly bool   `json:"unused_only"`
	}
	ListGiftCodeParamCtx struct{}
)

// List 获取兑换码列表。
func (s *ListGiftCodeService) List(c *gin.Context) (*ListGiftCodeResponse, error) {
	dep := dependency.FromContext(c)
	giftCodeClient := dep.GiftCodeClient()
	hasher := dep.HashIDEncoder()

	res, err := giftCodeClient.List(c, &inventory.ListGiftCodeArgs{
		PaginationArgs: &inventory.PaginationArgs{
			PageSize:            s.PageSize,
			PageToken:           s.PageToken,
			UseCursorPagination: s.PageToken != "",
		},
		UsedOnly:   s.UsedOnly,
		UnusedOnly: s.UnusedOnly,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list gift codes", err)
	}

	codes := make([]GetGiftCodeResponse, 0, len(res.Codes))
	for _, gc := range res.Codes {
		codes = append(codes, GetGiftCodeResponse{
			GiftCode: gc,
			HashID:   hashid.EncodeGiftCodeID(hasher, gc.ID),
		})
	}

	return &ListGiftCodeResponse{
		Pagination: res.PaginationResults,
		Codes:      codes,
	}, nil
}

// BatchDeleteGiftCodeService 批量删除兑换码服务。
type (
	BatchDeleteGiftCodeService struct {
		IDs []int `json:"ids" binding:"required,min=1"`
	}
	BatchDeleteGiftCodeParamCtx struct{}
)

// Delete 批量删除兑换码。
func (s *BatchDeleteGiftCodeService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	giftCodeClient := dep.GiftCodeClient()

	if err := giftCodeClient.Delete(c, s.IDs); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete gift codes", err)
	}

	return nil
}

// ListOrderResponse 订单列表响应。
type ListOrderResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Orders     []GetOrderResponse           `json:"orders"`
}

// GetOrderResponse 订单详情响应。
type GetOrderResponse struct {
	*ent.Order
	HashID     string `json:"hash_id,omitempty"`
	UserHashID string `json:"user_hash_id,omitempty"`
}

// ListOrderService 订单列表服务。
type (
	ListOrderService struct {
		PageSize    int    `json:"page_size" binding:"min=1,max=100"`
		PageToken   string `json:"page_token"`
		ProductType string `json:"product_type"`
	}
	ListOrderParamCtx struct{}
)

// List 获取订单列表。
func (s *ListOrderService) List(c *gin.Context) (*ListOrderResponse, error) {
	dep := dependency.FromContext(c)
	orderClient := dep.OrderClient()
	hasher := dep.HashIDEncoder()

	ctx := context.WithValue(c, inventory.LoadOrderUser{}, true)
	args := &inventory.ListOrderArgs{
		PaginationArgs: &inventory.PaginationArgs{
			PageSize:            s.PageSize,
			PageToken:           s.PageToken,
			UseCursorPagination: s.PageToken != "",
		},
	}
	if s.ProductType != "" {
		args.ProductTypes = []types.ProductType{types.ProductType(s.ProductType)}
	}

	res, err := orderClient.List(ctx, args)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list orders", err)
	}

	orders := make([]GetOrderResponse, 0, len(res.Orders))
	for _, o := range res.Orders {
		orderRes := GetOrderResponse{
			Order:  o,
			HashID: hashid.EncodeOrderID(hasher, o.ID),
		}
		if o.Edges.User != nil {
			orderRes.UserHashID = hashid.EncodeUserID(hasher, o.Edges.User.ID)
		}
		orders = append(orders, orderRes)
	}

	return &ListOrderResponse{
		Pagination: res.PaginationResults,
		Orders:     orders,
	}, nil
}

// SingleOrderService 订单ID服务。
type (
	SingleOrderService struct {
		ID int `uri:"id" json:"id" binding:"required"`
	}
	SingleOrderParamCtx struct{}
)

// Delete 删除订单。
func (s *SingleOrderService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	orderClient := dep.OrderClient()

	if err := orderClient.Delete(c, []int{s.ID}); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete order", err)
	}

	return nil
}

// CleanupOrderService 订单清理服务。
type (
	CleanupOrderService struct {
		Status   string `json:"status" binding:"required"`
		NotAfter string `json:"not_after"`
	}
	CleanupOrderParamCtx struct{}
)

// Cleanup 清理指定状态的订单。
func (s *CleanupOrderService) Cleanup(c *gin.Context) error {
	dep := dependency.FromContext(c)
	orderClient := dep.OrderClient()

	var notAfter *time.Time
	if s.NotAfter != "" {
		t, err := time.Parse(time.RFC3339, s.NotAfter)
		if err != nil {
			return serializer.NewError(serializer.CodeParamErr, "Invalid not_after time", err)
		}
		notAfter = &t
	}

	ids := make([]int, 0)
	pageToken := ""
	for {
		res, err := orderClient.List(c, &inventory.ListOrderArgs{
			PaginationArgs: &inventory.PaginationArgs{
				PageSize:            100,
				PageToken:           pageToken,
				UseCursorPagination: true,
			},
		})
		if err != nil {
			return serializer.NewError(serializer.CodeDBError, "Failed to list orders", err)
		}

		for _, o := range res.Orders {
			if types.OrderStatus(o.Status) != types.OrderStatus(s.Status) {
				continue
			}
			if notAfter != nil && !o.CreatedAt.Before(*notAfter) {
				continue
			}
			ids = append(ids, o.ID)
		}

		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}

	if err := orderClient.Delete(c, ids); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to cleanup orders", err)
	}

	return nil
}

// AdjustCreditService 用户积分调整服务。UID 从路径参数 id 读取，Amount/Reason 来自请求体。
type (
	AdjustCreditService struct {
		Amount int    `json:"amount" binding:"required"`
		Reason string `json:"reason"`
	}
	AdjustCreditParamCtx struct{}
)

// Adjust 调整用户积分。
func (s *AdjustCreditService) Adjust(c *gin.Context) (*GetUserResponse, error) {
	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()
	userClient := dep.UserClient()

	if s.Amount == 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "Amount is required", nil)
	}

	uid, err := hasher.Decode(c.Param("id"), hashid.UserID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid user id", err)
	}

	u, err := userClient.GetByID(c, uid)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeUserNotFound, "User not found", err)
	}

	newCredit := u.Credit + s.Amount
	if newCredit < 0 {
		return nil, serializer.NewError(serializer.CodeInsufficientCredit, "Insufficient credit", nil)
	}

	updated, err := u.Update().SetCredit(newCredit).Save(c)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to update user credit", err)
	}

	return &GetUserResponse{
		User:         updated,
		HashID:       hashid.EncodeUserID(hasher, updated.ID),
		TwoFAEnabled: updated.TwoFactorSecret != "",
	}, nil
}
