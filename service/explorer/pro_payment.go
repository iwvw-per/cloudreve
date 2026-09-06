package explorer

import (
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/service/pro"
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
// 当 Provider 为 credits 时，立即从用户积分余额扣减并完成履约。
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

	// 积分支付：立即扣减积分并履约，订单直接完成。
	if s.Provider == types.PaymentProviderCredits {
		return s.payWithCredits(c, dep, user, product)
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

// payWithCredits 使用积分余额即时支付订单：校验余额、创建订单、扣积分并履约。
func (s *CreateOrderService) payWithCredits(c *gin.Context, dep dependency.Dep, user *ent.User, product *ent.Product) (*ent.Order, error) {
	props := product.Props
	if props == nil {
		props = &types.ProductProps{}
	}
	if props.PriceCredits <= 0 {
		return nil, serializer.NewError(serializer.CodeParamErr, "This product cannot be purchased with credits", nil)
	}

	totalCredits := props.PriceCredits * s.Quantity
	if user.Credit < totalCredits {
		return nil, serializer.NewError(serializer.CodeInsufficientCredit, "Insufficient credits", nil)
	}

	orderNo := fmt.Sprintf("CR%s%d", uuid.Must(uuid.NewV4()).String()[:8], time.Now().Unix())
	orderClient := dep.OrderClient()
	productClient := dep.ProductClient()
	userClient := dep.UserClient()
	eventClient := dep.EventClient()

	txOrder, tx, ctx, err := inventory.WithTx(c, orderClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create payment transaction", err)
	}
	txProduct, _ := inventory.InheritTx(ctx, productClient)
	txUser, _ := inventory.InheritTx(ctx, userClient)
	txEvent, _ := inventory.InheritTx(ctx, eventClient)

	order, err := txOrder.Create(ctx, &inventory.CreateOrderParams{
		OrderNo:     orderNo,
		UserID:      user.ID,
		ProductType: s.ProductType,
		ProductID:   s.ProductID,
		Quantity:    s.Quantity,
		Amount:      totalCredits,
		Provider:    types.PaymentProviderCredits,
	})
	if err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create order", err)
	}

	// 扣减积分
	if _, err := txUser.GetClient().User.UpdateOneID(user.ID).SetCredit(user.Credit - totalCredits).Save(ctx); err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to deduct credits", err)
	}

	if _, err := txOrder.UpdateStatus(ctx, order.ID, types.OrderStatusPaid, types.PaymentProviderCredits); err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to mark order paid", err)
	}

	if err := pro.FulfillOrder(ctx, dep, txProduct, txUser, txEvent, order, types.PaymentProviderCredits); err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeFulfillAdminGroup, "Failed to fulfill order", err)
	}

	if _, err := txOrder.MarkFulfilled(ctx, order.ID); err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to mark order fulfilled", err)
	}

	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit payment transaction", err)
	}

	return order, nil
}
