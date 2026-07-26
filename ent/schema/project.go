package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Project holds the schema definition for the Project entity.
type Project struct {
	ent.Schema
}

// Fields of the Project.
func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("project_id").
			Unique().
			NotEmpty().
			MaxLen(64),
		field.String("name").
			NotEmpty().
			MaxLen(255),
		field.Text("description").
			Optional(),
		field.Text("repository_url").
			Optional(),
		field.String("repository_branch").
			Default("main").
			MaxLen(100),
		field.Ints("ports_exposed").
			Default([]int{}),
		field.Ints("ports_internal").
			Default([]int{}),
		field.Enum("environment").
			Values("development", "staging", "production").
			Default("development"),
		field.Strings("tags").
			Default([]string{}),
		field.String("caddy_server_id").
			Optional().
			MaxLen(64),
		field.Enum("default_ssl_mode").
			Values("auto", "manual", "disabled").
			Default("auto"),
		field.Enum("plan").
			Values("free", "pro", "enterprise").
			Default("free"),
		field.Int("max_domains").
			Default(10),
		field.Int("max_upstreams_per_proxy").
			Default(5),
		field.Int("max_members").
			Default(5),
		field.Int("max_shared_domains").
			Default(3),
		field.Int("max_config_snapshots").
			Default(50),
		field.Int("rate_limit_rpm").
			Default(30),
		field.Int("rate_limit_wpm").
			Default(10),
		field.Enum("status").
			Values("active", "suspended", "deleting", "error", "deleted").
			Default("active"),
		field.UUID("created_by", uuid.UUID{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Project.
func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("domains", Domain.Type),
		edge.To("members", ProjectMember.Type),
		edge.To("audit_logs", ProjectAuditLog.Type),
		edge.To("invitations", Invitation.Type),
		edge.To("shares", DomainShare.Type),
		edge.To("config_snapshots", ConfigSnapshot.Type),
		edge.To("api_tokens", ApiToken.Type),
		edge.From("owner", User.Type).
			Ref("owned_projects").
			Unique().
			Field("created_by"),
	}
}

// Indexes of the Project.
func (Project) Indexes() []ent.Index {
	return []ent.Index{
		// project_id is unique via field definition; add indexes for lookup.
		index.Fields("environment"),
		index.Fields("status"),
		index.Fields("created_by"),
		index.Fields("caddy_server_id"),
	}
}
