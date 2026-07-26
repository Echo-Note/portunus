package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectMember holds the schema definition for the ProjectMember entity.
// The composite primary key is (user_id, project_id).
type ProjectMember struct {
	ent.Schema
}

// Fields of the ProjectMember.
func (ProjectMember) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}),
		field.UUID("project_id", uuid.UUID{}),
		field.Enum("role").
			Values("owner", "admin", "editor", "viewer"),
		field.Enum("status").
			Values("pending", "active", "removed", "left").
			Default("pending"),
		field.UUID("invited_by", uuid.UUID{}).
			Optional(),
		field.Time("joined_at").
			Optional(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.JSON("scope", []string{}).
			Default([]string{}),
	}
}

// Edges of the ProjectMember.
func (ProjectMember) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Unique().
			Required().
			Field("user_id"),
		edge.To("project", Project.Type).
			Unique().
			Required().
			Field("project_id"),
		edge.To("inviter", User.Type).
			Unique().
			Field("invited_by"),
	}
}

// Indexes of the ProjectMember.
func (ProjectMember) Indexes() []ent.Index {
	return []ent.Index{
		// Composite primary key (user_id, project_id).
		index.Fields("user_id", "project_id").Unique(),
		index.Fields("project_id"),
		index.Fields("status"),
	}
}
