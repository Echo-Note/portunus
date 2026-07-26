package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Upstream holds the schema definition for the Upstream entity.
// An Upstream is a single dial target behind a ProxyConfig (load-balanced).
type Upstream struct {
	ent.Schema
}

// Fields of the Upstream.
func (Upstream) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		// proxy_config_id is a REFERENCES column to proxy_configs(id); immutable after creation.
		field.UUID("proxy_config_id", uuid.UUID{}),
		field.String("dial_address").
			NotEmpty().
			MaxLen(255),
		field.Int("max_requests").
			Optional(),
		field.Int("weight").
			Default(1),
		field.Enum("status").
			Values("active", "unhealthy", "disabled", "removed").
			Default("active"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Upstream.
func (Upstream) Edges() []ent.Edge {
	return []ent.Edge{
		// An upstream belongs to exactly one proxy config (M2O, FK owner).
		edge.To("proxy_config", ProxyConfig.Type).
			Unique().
			Required().
			Field("proxy_config_id"),
	}
}

// Indexes of the Upstream.
func (Upstream) Indexes() []ent.Index {
	return []ent.Index{
		// proxy_config_id is a FK; index for lookup performance.
		index.Fields("proxy_config_id"),
		// UNIQUE (proxy_config_id, dial_address) — a proxy config cannot list
		// the same dial address twice.
		index.Fields("proxy_config_id", "dial_address").Unique(),
	}
}
