package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AccessRequest struct {
	ent.Schema
}

func (AccessRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Int("requester_id"),
		field.Int("manager_id").
			Optional().
			Nillable(),

		field.Int("reviewer_id").
			Optional().
			Nillable(),
		field.String("resource").
			NotEmpty().
			Comment("The system or resource access is being requested for"),
		field.String("reason").
			NotEmpty().
			Comment("Business justification for the request"),
		field.Enum("status").
			Values("PENDING", "APPROVED", "REJECTED", "UNDER_REVIEW").
			Default("PENDING"),
		field.String("screenshot_url").
			Optional().
			Nillable().
			Comment("S3 URL of the uploaded proof screenshot"),
		field.String("review_comment").
			Optional().
			Nillable().
			Comment("Comment left by reviewer during approval/rejection"),
		// We store manager email at submission time for audit immutability
		field.String("manager_email").
			NotEmpty(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("resolved_at").
			Optional().
			Nillable().
			Comment("When the request was approved or rejected"),
	}
}

func (AccessRequest) Edges() []ent.Edge {
	return []ent.Edge{
		// Many-to-one: many requests belong to one requester
		edge.From("requester", User.Type).
			Ref("submitted_requests").
			Field("requester_id").
			Unique().
			Required(),
		// Many-to-one: many requests have one manager
		edge.From("manager", User.Type).
			Ref("managed_requests").
			Unique().
			Field("manager_id"),
		// Optional reviewer (support/engineering/admin who acted on it)
		edge.From("reviewer", User.Type).
			Ref("reviewed_requests").
			Unique().
			Field("reviewer_id"),
		// One request can have many audit events
		edge.To("audit_logs", AuditLog.Type),
	}
}

func (AccessRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("requester_id"),
		index.Fields("manager_email"),
		// Composite index for the most common search: filter by requester + status
		index.Fields("requester_id", "status"),
		index.Fields("resource"),
	}
}
