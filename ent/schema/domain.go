package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Domain holds the schema definition for the Domain entity.
type Domain struct {
	ent.Schema
}

// Fields of the Domain.
func (Domain) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		// project_id is a REFERENCES column to projects(id); immutable after creation.
		field.UUID("project_id", uuid.UUID{}),
		field.String("domain_name").
			Unique().
			NotEmpty().
			MaxLen(253),
		field.String("caddy_id").
			Unique().
			NotEmpty().
			MaxLen(128),
		field.String("route_id").
			Optional().
			MaxLen(128),
		field.Bool("ssl_enabled").
			Default(true),
		field.Enum("status").
			Values("creating", "active", "updating", "error", "disabled", "deleting", "deleted").
			Default("creating"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Domain.
func (Domain) Edges() []ent.Edge {
	return []ent.Edge{
		// Each domain belongs to exactly one project (M2O, FK owner).
		edge.To("project", Project.Type).
			Unique().
			Required().
			Field("project_id"),
		// A domain has one proxy configuration (O2O); the FK lives on proxy_configs.
		edge.To("proxy_config", ProxyConfig.Type).
			Unique(),
		// A domain may be shared with multiple projects.
		edge.To("shares", DomainShare.Type),
	}
}

// Indexes of the Domain.
func (Domain) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		// caddy_id already has a unique index via the field definition; this
		// secondary index mirrors the SQL schema (idx_domains_caddy_id).
		index.Fields("caddy_id"),
	}
}
