package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// Event holds the schema definition for the audit log / activity log entity.
type Event struct {
	ent.Schema
}

// Fields of the Event.
func (Event) Fields() []ent.Field {
	return []ent.Field{
		field.Int("type").
			Comment("Audit log type, aligned with types.AuditType"),
		field.String("correlation_id").
			Optional().
			MaxLen(64),
		field.String("ip").
			Optional().
			MaxLen(64),
		field.String("user_agent").
			Optional().
			MaxLen(1024),
		field.JSON("content", &types.AuditContent{}).
			Optional(),
		field.Int("user_events").Optional(),
		field.Int("file_events").Optional(),
		field.Int("entity_events").Optional(),
		field.Int("share_events").Optional(),
	}
}

// Edges of the Event.
func (Event) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("events").
			Field("user_events").
			Unique(),
		edge.From("file", File.Type).
			Ref("events").
			Field("file_events").
			Unique(),
		edge.From("entity", Entity.Type).
			Ref("events").
			Field("entity_events").
			Unique(),
		edge.From("share", Share.Type).
			Ref("events").
			Field("share_events").
			Unique(),
	}
}

func (Event) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}
