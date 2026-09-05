package explorer

import (
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type (
	CreateOrderParamCtx struct{}
	CreateOrderService  struct {
		ProductID   int                   `json:"product_id" binding:"required"`
		ProductType types.ProductType     `json:"product_type" binding:"required"`
		Quantity    int                   `json:"quantity"`
		Provider    types.PaymentProvider `json:"provider"`
	}
)

// Create 创建一笔待支付订单。金额按商品单价 × 数量计算。
func (s *CreateOrderService) Create(c *gin.Context) (*ent.Order, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	if user == nil || inventory.IsAnonymousUser(user) {
		return nil, serializer.NewError(serializer.CodeCheckLogin, "Login required", nil)
	}
	if s.Quantity <= 0 {
		s.Quantity = 1
	}

	product, err := dep.ProductClient().GetByID(c, s.ProductID)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeNotFound, "Product not found", err)
	}
	if !product.Enabled {
		return nil, serializer.NewError(serializer.CodeParamErr, "Product is disabled", nil)
	}
	if string(product.Type) != string(s.ProductType) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Product type mismatch", nil)
	}

	orderNo := fmt.Sprintf("CR%s%d", uuid.Must(uuid.NewV4()).String()[:8], time.Now().Unix())
	order, err := dep.OrderClient().Create(c, &inventory.CreateOrderParams{
		OrderNo:     orderNo,
		UserID:      user.ID,
		ProductType: s.ProductType,
		ProductID:   s.ProductID,
		Quantity:    s.Quantity,
		Amount:      product.Price * s.Quantity,
		Provider:    s.Provider,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create order", err)
	}

	return order, nil
}
