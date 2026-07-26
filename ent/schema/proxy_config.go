package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProxyConfig holds the schema definition for the ProxyConfig entity.
// It models a Caddy reverse-proxy configuration for a domain.
type ProxyConfig struct {
	ent.Schema
}

// Fields of the ProxyConfig.
func (ProxyConfig) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		// domain_id is a REFERENCES column to domains(id); immutable after creation.
		field.UUID("domain_id", uuid.UUID{}),
		field.String("caddy_proxy_id").
			Unique().
			NotEmpty().
			MaxLen(128),
		field.Enum("lb_policy").
			Values("random", "round_robin", "least_conn", "ip_hash", "uri_hash").
			Default("random"),
		field.String("health_check_uri").
			Optional().
			MaxLen(255),
		field.String("health_check_interval").
			Default("30s").
			MaxLen(20),
		field.String("timeout").
			Default("0s").
			MaxLen(20),
		field.Enum("status").
			Values("active", "updating", "degraded", "unavailable").
			Default("active"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the ProxyConfig.
func (ProxyConfig) Edges() []ent.Edge {
	return []ent.Edge{
		// A proxy config belongs to exactly one domain (O2O, FK owner).
		edge.To("domain", Domain.Type).
			Unique().
			Required().
			Field("domain_id"),
		// A proxy config has many upstream targets.
		edge.To("upstreams", Upstream.Type),
	}
}

// Indexes of the ProxyConfig.
func (ProxyConfig) Indexes() []ent.Index {
	return []ent.Index{
		// domain_id is a FK; index for lookup performance.
		index.Fields("domain_id"),
	}
}
