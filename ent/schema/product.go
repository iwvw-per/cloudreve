package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// Product holds the schema definition for a VAS product (storage pack / membership / credit).
type Product struct {
	ent.Schema
}

// Fields of the Product.
func (Product) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.Enum("type").
			Values(string(types.ProductTypeStorage), string(types.ProductTypeGroup), string(types.ProductTypeCredit)),
		field.Int("price").
			Comment("Price in cents"),
		field.Bool("highlight").
			Default(false),
		field.Bool("enabled").
			Default(true),
		field.JSON("props", &types.ProductProps{}).
			Optional().
			Default(&types.ProductProps{}),
	}
}

func (Product) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}