package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("email").
			Unique().
			NotEmpty().
			MaxLen(255),
		field.String("password_hash").
			Optional().
			MaxLen(255),
		field.Enum("status").
			Values("pending", "active", "suspended", "deleted").
			Default("pending"),
		field.Time("email_verified_at").
			Optional(),
		field.Time("last_login_at").
			Optional(),
		field.String("oauth_provider").
			Optional().
			MaxLen(50),
		field.String("oauth_subject").
			Optional().
			MaxLen(255),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("owned_projects", Project.Type),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		// email is already unique via field definition; add a secondary index
		// on oauth provider/subject for lookup performance.
		index.Fields("oauth_provider", "oauth_subject"),
		index.Fields("status"),
	}
}
