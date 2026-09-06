package pro

import (
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// ShopProduct 商城商品序列化结构。
type ShopProduct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	// Price 现金价格（分）
	Price int `json:"price"`
	// PriceCredits 积分购买价格，0 表示不支持积分购买
	PriceCredits int `json:"price_credits"`
	// CreditAmount 积分类商品的积分面额
	CreditAmount int `json:"credit_amount"`
	// StorageSize 存储类商品容量（字节）
	StorageSize int64 `json:"storage_size"`
	// DurationDays 有效期天数，0 表示永久
	DurationDays int          `json:"duration_days"`
	GroupID      int          `json:"group_id"`
	Description  []string     `json:"description,omitempty"`
	Highlight    bool         `json:"highlight"`
	HashID       string       `json:"hash_id,omitempty"`
	Product      *ent.Product `json:"product,omitempty"`
}

// GetShopService 用户侧商城服务：返回启用的商品列表与当前用户积分余额。
type (
	GetShopService  struct{}
	GetShopParamCtx struct{}
	GetShopResponse struct {
		Products []ShopProduct `json:"products"`
		Credit   int           `json:"credit"`
	}
)

// Get 获取商城数据。
func (s *GetShopService) Get(c *gin.Context) (*GetShopResponse, error) {
	dep := dependency.FromContext(c)
	hasher := dep.HashIDEncoder()

	appSetting := dep.SettingProvider().AppSetting(c)
	products := make([]ShopProduct, 0)

	// 积分/增值服务未启用时，商城返回空商品列表，前端展示未启用提示。
	if appSetting.CreditEnabled {
		res, err := dep.ProductClient().List(c, &inventory.ListProductArgs{
			PaginationArgs: &inventory.PaginationArgs{
				PageSize: 100,
			},
			EnabledOnly: true,
		})
		if err != nil {
			return nil, serializer.NewError(serializer.CodeDBError, "Failed to list products", err)
		}

		products = make([]ShopProduct, 0, len(res.Products))
		for _, p := range res.Products {
			sp := ShopProduct{
				ID:        p.ID,
				Name:      p.Name,
				Type:      string(p.Type),
				Price:     p.Price,
				Highlight: p.Highlight,
				HashID:    hashid.EncodeProductID(hasher, p.ID),
				Product:   p,
			}
			if props := p.Props; props != nil {
				sp.PriceCredits = props.PriceCredits
				sp.CreditAmount = props.CreditAmount
				sp.StorageSize = props.Size
				sp.DurationDays = props.DurationDays
				sp.GroupID = props.GroupID
				sp.Description = props.Description
			}
			products = append(products, sp)
		}
	}

	credit := 0
	if user := inventory.UserFromContext(c); user != nil {
		credit = user.Credit
	}

	return &GetShopResponse{Products: products, Credit: credit}, nil
}
