package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").
			Unique().
			NotEmpty().
			Immutable(), // Email never changes after creation
		field.String("name").
			NotEmpty(),
		field.String("password_hash").
			Sensitive(), // Ent won't log this field
		field.Enum("role").
			Values("EMPLOYEE", "SUPPORT", "ENGINEERING", "ADMIN").
			Default("EMPLOYEE"),
		field.Bool("is_active").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		// A user can create many access requests
		edge.To("submitted_requests", AccessRequest.Type),

		// A user can manage/approve many requests
		edge.To("managed_requests", AccessRequest.Type),

		// A user can review many requests
		edge.To("reviewed_requests", AccessRequest.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email"),
		index.Fields("role"),
	}
}
