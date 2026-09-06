package pro

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// FulfillOrder 依据商品类型与商品属性完成履约。
// productClient/userClient/eventClient 应为事务作用域客户端（或普通客户端），由调用方决定。
func FulfillOrder(ctx context.Context, dep dependency.Dep, productClient inventory.ProductClient,
	userClient inventory.UserClient, eventClient inventory.EventClient, order *ent.Order, provider types.PaymentProvider) error {
	product, err := productClient.GetByID(ctx, order.ProductID)
	if err != nil {
		return fmt.Errorf("product %d not found: %w", order.ProductID, err)
	}

	user, err := userClient.GetByID(ctx, order.UserOrders)
	if err != nil {
		return fmt.Errorf("user %d not found: %w", order.UserOrders, err)
	}

	props := product.Props
	if props == nil {
		props = &types.ProductProps{}
	}

	switch types.ProductType(order.ProductType) {
	case types.ProductTypeStorage:
		size := props.Size * int64(order.Quantity)
		if size <= 0 {
			return fmt.Errorf("product %d has no size", product.ID)
		}
		newExpire := int64(0)
		if props.DurationDays > 0 {
			base := user.ExtraStorageExpire
			if base < time.Now().Unix() {
				base = time.Now().Unix()
			}
			newExpire = base + int64(props.DurationDays)*86400
		}
		if _, err := user.Update().
			SetExtraStorage(user.ExtraStorage + size).
			SetExtraStorageExpire(newExpire).
			Save(ctx); err != nil {
			return fmt.Errorf("failed to add %d extra storage to user %d: %w", size, user.ID, err)
		}
		writeStorageAddedEvent(ctx, eventClient, order, product, size)
	case types.ProductTypeGroup:
		if props.GroupID == 0 {
			return fmt.Errorf("product %d has no group_id", order.ID)
		}
		if user.GroupUsers == 1 {
			return fmt.Errorf("admin user cannot be downgraded by group purchase")
		}
		from := user.GroupUsers
		if _, err := user.Update().SetGroupID(props.GroupID).Save(ctx); err != nil {
			return fmt.Errorf("failed to upgrade user %d to group %d: %w", user.ID, props.GroupID, err)
		}
		writeGroupChangedEvent(ctx, eventClient, order, from, props.GroupID)
	case types.ProductTypeCredit:
		amount := props.CreditAmount * order.Quantity
		if amount == 0 {
			return fmt.Errorf("product %d has no credit_amount", product.ID)
		}
		if _, err := user.Update().SetCredit(user.Credit + amount).Save(ctx); err != nil {
			return fmt.Errorf("failed to add %d credits to user %d: %w", amount, user.ID, err)
		}
		writePointsChangeEvent(ctx, eventClient, order, amount)
	default:
		return fmt.Errorf("unknown product type %q for order %s", order.ProductType, order.OrderNo)
	}

	return nil
}

func writeStorageAddedEvent(ctx context.Context, eventClient inventory.EventClient, order *ent.Order, product *ent.Product, size int64) {
	_, _ = eventClient.Create(ctx, &inventory.CreateEventParams{
		Type:   types.AuditTypeStorageAdded,
		UserID: order.UserOrders,
		Content: &types.AuditContent{
			PaymentID:   order.ID,
			Sku:         product.Name,
			StorageSize: size,
		},
	})
}

func writeGroupChangedEvent(ctx context.Context, eventClient inventory.EventClient, order *ent.Order, from, to int) {
	_, _ = eventClient.Create(ctx, &inventory.CreateEventParams{
		Type:   types.AuditTypeGroupChanged,
		UserID: order.UserOrders,
		Content: &types.AuditContent{
			PaymentID:   order.ID,
			GroupIDFrom: from,
			GroupID:     to,
		},
	})
}

func writePointsChangeEvent(ctx context.Context, eventClient inventory.EventClient, order *ent.Order, amount int) {
	_, _ = eventClient.Create(ctx, &inventory.CreateEventParams{
		Type:   types.AuditTypePointsChange,
		UserID: order.UserOrders,
		Content: &types.AuditContent{
			PaymentID:    order.ID,
			PointsChange: amount,
		},
	})
}
