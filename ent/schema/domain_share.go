package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DomainShare holds the schema definition for the DomainShare entity.
// 域名共享表：将某域名从源项目共享给目标项目，支持只读/编辑权限与生命周期状态。
type DomainShare struct {
	ent.Schema
}

// Fields of the DomainShare.
func (DomainShare) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		// domain_id REFERENCES domains(id) ON DELETE CASCADE
		field.UUID("domain_id", uuid.UUID{}),
		// source_project_id REFERENCES projects(id) ON DELETE CASCADE
		field.UUID("source_project_id", uuid.UUID{}),
		// target_project_id REFERENCES projects(id) ON DELETE CASCADE
		field.UUID("target_project_id", uuid.UUID{}),
		field.Enum("permission").
			Values("read_only", "edit").
			Default("read_only"),
		field.Enum("status").
			Values("pending", "active", "revoked", "expired", "rejected").
			Default("pending"),
		field.Time("expires_at").
			Optional(),
		// created_by REFERENCES users(id)（可空）
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

// Edges of the DomainShare.
func (DomainShare) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("domain", Domain.Type).
			Unique().
			Required().
			Field("domain_id"),
		edge.To("source_project", Project.Type).
			Unique().
			Required().
			Field("source_project_id"),
		edge.To("target_project", Project.Type).
			Unique().
			Required().
			Field("target_project_id"),
		edge.To("creator", User.Type).
			Unique().
			Field("created_by"),
	}
}

// Indexes of the DomainShare.
func (DomainShare) Indexes() []ent.Index {
	return []ent.Index{
		// UNIQUE(domain_id, target_project_id)
		index.Fields("domain_id", "target_project_id").Unique(),
		// idx_domain_shares_target: (target_project_id, status)
		index.Fields("target_project_id", "status"),
		// idx_domain_shares_domain: (domain_id)
		index.Fields("domain_id"),
	}
}
