package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CaddyIDMapping holds the schema definition for the CaddyIDMapping entity.
// It maps Caddy's internal @id identifiers to Portunus project resources,
// so the Portunus side can resolve a Caddy config node back to the owning
// resource (domain, proxy_config, upstream, ...) without scanning the full
// Caddy JSON tree.
//
// The caddy_id column (VARCHAR 128) is the primary key. ent requires the PK
// field to be named "id", so it is declared as field.String("id") with a
// StorageKey of "caddy_id" to keep the DB column name aligned with the SQL
// migration.
type CaddyIDMapping struct {
	ent.Schema
}

// Fields of the CaddyIDMapping.
func (CaddyIDMapping) Fields() []ent.Field {
	return []ent.Field{
		// caddy_id (VARCHAR 128) is the primary key; provided by the caller on
		// Create (Caddy-generated identifier).
		field.String("id").
			StorageKey("caddy_id").
			MaxLen(128).
			Immutable(),
		// project_id is a REFERENCES column to projects(id); immutable after creation.
		field.UUID("project_id", uuid.UUID{}),
		field.String("resource_type").
			NotEmpty().
			MaxLen(30),
		field.UUID("resource_id", uuid.UUID{}),
		field.String("caddy_json_path").
			Optional().
			MaxLen(512),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the CaddyIDMapping.
func (CaddyIDMapping) Edges() []ent.Edge {
	return []ent.Edge{
		// Each mapping belongs to exactly one project (M2O, FK owner).
		edge.To("project", Project.Type).
			Unique().
			Required().
			Field("project_id"),
	}
}

// Indexes of the CaddyIDMapping.
func (CaddyIDMapping) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("resource_type"),
	}
}

// Annotations of the CaddyIDMapping.
// Pin the table name explicitly so the "ID" acronym in the type name does
// not produce an unexpected snake_case form.
func (CaddyIDMapping) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "caddy_id_mappings"},
	}
}
