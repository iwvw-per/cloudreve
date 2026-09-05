package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// GiftCode holds the schema definition for an redemption code.
type GiftCode struct {
	ent.Schema
}

// Fields of the GiftCode.
func (GiftCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Unique(),
		field.JSON("props", &types.GiftCodeProps{}).
			Optional().
			Default(&types.GiftCodeProps{}),
		field.Int("used_by").
			Optional(),
		field.Time("used_at").
			Nillable().
			Optional(),
		field.Int("user_codes").Optional(),
	}
}

// Edges of the GiftCode.
func (GiftCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("gift_codes").
			Field("user_codes").
			Unique(),
	}
}

func (GiftCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}