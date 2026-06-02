package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AuditLog struct {
	ent.Schema
}

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("access_request_id"),
		field.Enum("action").
			Values(
				"REQUEST_CREATED",
				"REQUEST_APPROVED",
				"REQUEST_REJECTED",
				"REQUEST_UNDER_REVIEW",
				"COMMENT_ADDED",
				"SCREENSHOT_UPLOADED",
				"USER_CREATED",
				"USER_ROLE_CHANGED",
			),
		field.String("actor_email").
			NotEmpty().
			Comment("Email of who performed the action"),
		field.String("actor_role").
			NotEmpty().
			Comment("Role of actor at time of action — immutable snapshot"),
		field.Text("metadata").
			Optional().
			Nillable().
			Comment("JSON blob for extra context — flexible for future actions"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (AuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("access_request", AccessRequest.Type).
			Ref("audit_logs").
			Unique().
			Required().
			Field("access_request_id"),
	}
}

func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("actor_email"),
		index.Fields("action"),
		index.Fields("access_request_id"),
	}
}
