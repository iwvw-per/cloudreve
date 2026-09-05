package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// Order holds the schema definition for a payment order.
type Order struct {
	ent.Schema
}

// Fields of the Order.
func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_no").
			Unique(),
		field.Enum("product_type").
			Values(string(types.ProductTypeStorage), string(types.ProductTypeGroup), string(types.ProductTypeCredit)),
		field.Int("product_id").
			Optional(),
		field.Int("quantity").
			Default(1),
		field.Int("amount").
			Comment("Amount in cents"),
		field.Enum("status").
			Values(string(types.OrderStatusUnpaid), string(types.OrderStatusPaid), string(types.OrderStatusFulfilled), string(types.OrderStatusFailed)).
			Default(string(types.OrderStatusUnpaid)),
		field.String("provider").
			Optional(),
		field.JSON("content", &types.GiftCodeProps{}).
			Optional(),
		field.Int("user_orders").Optional(),
	}
}

// Edges of the Order.
func (Order) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("orders").
			Field("user_orders").
			Unique(),
	}
}

func (Order) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}