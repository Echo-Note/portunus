package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConfigSnapshot holds the schema definition for the ConfigSnapshot entity.
// 配置快照表：保存某项目在某 version 下的 Caddy JSON 全量配置，支持回滚与审计。
type ConfigSnapshot struct {
	ent.Schema
}

// Fields of the ConfigSnapshot.
func (ConfigSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		// project_id REFERENCES projects(id) ON DELETE CASCADE
		field.UUID("project_id", uuid.UUID{}),
		field.Text("caddy_json"),
		field.Int("version"),
		// checksum VARCHAR(64) NOT NULL
		field.String("checksum").
			NotEmpty().
			MaxLen(64),
		// created_by REFERENCES users(id)（可空）
		field.UUID("created_by", uuid.UUID{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the ConfigSnapshot.
func (ConfigSnapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project", Project.Type).
			Unique().
			Required().
			Field("project_id"),
		edge.To("creator", User.Type).
			Unique().
			Field("created_by"),
	}
}

// Indexes of the ConfigSnapshot.
func (ConfigSnapshot) Indexes() []ent.Index {
	return []ent.Index{
		// UNIQUE(project_id, version)
		index.Fields("project_id", "version").Unique(),
	}
}
