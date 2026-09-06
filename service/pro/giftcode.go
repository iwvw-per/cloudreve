package pro

import (
	"errors"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/order"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

// RedeemGiftCodeService 用户侧兑换码兑付服务。
type (
	RedeemGiftCodeService struct {
		Code string `json:"code" binding:"required"`
	}
	RedeemGiftCodeParamCtx struct{}
	RedeemGiftCodeResponse  struct {
		Product *ent.Product `json:"product"`
		User    *ent.User    `json:"user"`
	}
)

// Redeem 校验兑换码并执行履约：查询码 -> 校验未使用 -> 按关联商品履约 -> 标记已用，整体在同一事务内。
func (s *RedeemGiftCodeService) Redeem(c *gin.Context) (*RedeemGiftCodeResponse, error) {
	dep := dependency.FromContext(c)
	user := inventory.UserFromContext(c)
	if user == nil || inventory.IsAnonymousUser(user) {
		return nil, serializer.NewError(serializer.CodeCheckLogin, "Login required", nil)
	}

	giftCodeClient := dep.GiftCodeClient()
	productClient := dep.ProductClient()
	userClient := dep.UserClient()
	eventClient := dep.EventClient()

	txGiftCode, tx, ctx, err := inventory.WithTx(c, giftCodeClient)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create transaction", err)
	}
	txProduct, _ := inventory.InheritTx(ctx, productClient)
	txUser, _ := inventory.InheritTx(ctx, userClient)
	txEvent, _ := inventory.InheritTx(ctx, eventClient)

	code, err := txGiftCode.GetByCode(ctx, s.Code)
	if err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeInvalidGiftCode, "Invalid gift code", err)
	}
	if code.UsedBy != 0 && code.UsedAt != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeInvalidGiftCode, "Gift code already used", nil)
	}

	// 关联商品校验
	props := code.Props
	if props == nil || props.LinkedProduct == 0 {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeInvalidGiftCode, "Gift code has no linked product", nil)
	}
	product, err := txProduct.GetByID(ctx, props.LinkedProduct)
	if err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeInvalidGiftCode, "Linked product not found", err)
	}

	// 构建一次订单记录（履约共用逻辑），便于审计与追溯。
	qty := props.ProductQty
	if qty <= 0 {
		qty = 1
	}
	order, err := txGiftCode.GetClient().Order.Create().
		SetOrderNo(fmt.Sprintf("GC%s%d", code.Code[:min(8, len(code.Code))], code.ID)).
		SetProductType(order.ProductType(string(product.Type))).
		SetProductID(product.ID).
		SetQuantity(qty).
		SetAmount(0).
		SetStatus(order.Status(string(types.OrderStatusFulfilled))).
		SetProvider(string(types.PaymentProviderCredits)).
		SetContent(props).
		SetUserOrders(user.ID).
		Save(ctx)
	if err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create order record", err)
	}

	if err := FulfillOrder(ctx, dep, txProduct, txUser, txEvent, order, types.PaymentProviderCredits); err != nil {
		inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeFulfillAdminGroup, "Failed to fulfill gift code", err)
	}

	if _, err := txGiftCode.Redeem(ctx, code.ID, user.ID); err != nil {
		inventory.Rollback(tx)
		if errors.Is(err, inventory.ErrGiftCodeUsed) {
			return nil, serializer.NewError(serializer.CodeInvalidGiftCode, "Gift code already used", err)
		}
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to redeem gift code", err)
	}

	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit redemption", err)
	}

	// 审计
	_, _ = eventClient.Create(c, &inventory.CreateEventParams{
		Type:   types.AuditTypeRedeemGiftCode,
		UserID: user.ID,
		Content: &types.AuditContent{
			Sku:    product.Name,
			Reason: "redeem gift code",
		},
	})

	freshUser, err := userClient.GetByID(c, user.ID)
	if err != nil {
		freshUser = user
	}
	return &RedeemGiftCodeResponse{Product: product, User: freshUser}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
