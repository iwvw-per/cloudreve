package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
)

// AbuseReport holds the schema definition for an abuse report.
type AbuseReport struct {
	ent.Schema
}

// Fields of the AbuseReport.
func (AbuseReport) Fields() []ent.Field {
	return []ent.Field{
		field.String("folder_path").
			Optional(),
		field.String("reason").
			Optional(),
		field.Text("description").
			Optional(),
		field.Enum("status").
			Values(string(types.AbuseStatusPending), string(types.AbuseStatusResolved), string(types.AbuseStatusIgnored)).
			Default(string(types.AbuseStatusPending)),
		field.JSON("raw_content", &types.AuditContent{}).
			Optional(),
		field.Int("reporter_user").Optional(),
		field.Int("reported_user").Optional(),
		field.Int("share_reports").Optional(),
	}
}

// Edges of the AbuseReport.
func (AbuseReport) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("reporter", User.Type).
			Ref("abuse_reports_made").
			Field("reporter_user").
			Unique(),
		edge.From("reported", User.Type).
			Ref("abuse_reports_received").
			Field("reported_user").
			Unique(),
		edge.From("share", Share.Type).
			Ref("abuse_reports").
			Field("share_reports").
			Unique(),
	}
}

func (AbuseReport) Mixin() []ent.Mixin {
	return []ent.Mixin{
		CommonMixin{},
	}
}