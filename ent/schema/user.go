package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").
			MaxLen(100).
			Unique(),
		field.String("nick").
			MaxLen(100),
		field.String("password").
			Optional().
			Sensitive(),
		field.Enum("status").
			Values("active", "inactive", "manual_banned", "sys_banned").
			Default("active"),
		field.Int64("storage").
			Default(0),
		field.Int64("extra_storage").
			Default(0).
			Comment("Additional storage capacity purchased via storage packs"),
		field.Int64("extra_storage_expire").
			Default(0).
			Comment("Unix timestamp when extra_storage expires, 0 means never"),
		field.String("two_factor_secret").
			Sensitive().
			Optional(),
		field.String("avatar").
			Optional(),
		field.JSON("settings", &types.UserSetting{}).
			Default(&types.UserSetting{}).
			Optional(),
		field.Int("group_users"),
		field.Int64("group_expires").
			Default(0).
			Comment("Unix timestamp when current group expires, 0 means never"),
		field.Int("previous_group").
			Default(0).
			Comment("Group ID to fall back to when current group expires"),
		field.Int("credit").
			Default(0).
			Comment("User credit points balance"),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", Group.Type).
			Ref("users").
			Field("group_users").
			Unique().
			Required(),
		edge.To("files", File.Type),
		edge.To("dav_accounts", DavAccount.Type),
		edge.To("shares", Share.Type),
		edge.To("passkey", Passkey.Type),
		edge.To("tasks", Task.Type),
		edge.To("fsevents", FsEvent.Type),
		edge.To("entities", Entity.Type),
		edge.To("oauth_grants", OAuthGrant.Type),
		edge.To("events", Event.Type),
		edge.To("orders", Order.Type),
		edge.To("gift_codes", GiftCode.Type),
		edge.To("abuse_reports_made", AbuseReport.Type),
		edge.To("abuse_reports_received", AbuseReport.Type),
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
