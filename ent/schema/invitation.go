package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Invitation holds the schema definition for the Invitation entity.
type Invitation struct {
	ent.Schema
}

// Fields of the Invitation.
func (Invitation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("project_id", uuid.UUID{}),
		field.String("email").
			NotEmpty().
			MaxLen(255),
		field.Enum("role").
			Values("admin", "editor", "viewer"),
		field.UUID("invited_by", uuid.UUID{}).
			Optional(),
		field.String("invitation_token").
			NotEmpty().
			MaxLen(500),
		field.Enum("status").
			Values("pending", "accepted", "rejected", "expired", "revoked").
			Default("pending"),
		field.Time("expires_at"),
		field.Time("accepted_at").
			Optional(),
		field.UUID("accepted_by", uuid.UUID{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Invitation.
func (Invitation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project", Project.Type).
			Unique().
			Required().
			Field("project_id"),
		edge.To("inviter", User.Type).
			Unique().
			Field("invited_by"),
		edge.To("acceptor", User.Type).
			Unique().
			Field("accepted_by"),
	}
}

// Indexes of the Invitation.
func (Invitation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("email"),
		index.Fields("invitation_token"),
		index.Fields("status"),
		index.Fields("expires_at"),
	}
}
