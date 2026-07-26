package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectAuditLog holds the schema definition for the ProjectAuditLog entity.
// 审计日志主键为 BIGSERIAL 自增序列（非 UUID），记录项目内所有变更操作。
type ProjectAuditLog struct {
	ent.Schema
}

// Fields of the ProjectAuditLog.
func (ProjectAuditLog) Fields() []ent.Field {
	return []ent.Field{
		// BIGSERIAL 自增主键。
		field.Int64("id").
			StorageKey("id").
			SchemaType(map[string]string{"postgres": "BIGSERIAL"}).
			Immutable(),
		// project_id REFERENCES projects(id) ON DELETE CASCADE
		field.UUID("project_id", uuid.UUID{}),
		field.Enum("actor_type").
			Values("user", "ai_agent", "system"),
		field.UUID("actor_id", uuid.UUID{}).
			Optional(),
		field.String("actor_name").
			Optional().
			MaxLen(255),
		// actor_ip INET，Ent 不原生支持 INET，使用 String 存储。
		field.String("actor_ip").
			Optional(),
		field.String("action").
			NotEmpty().
			MaxLen(50),
		field.String("resource_type").
			NotEmpty().
			MaxLen(30),
		field.String("resource_id").
			Optional().
			MaxLen(128),
		// changes_before / changes_after JSONB
		field.JSON("changes_before", map[string]any{}).
			Optional(),
		field.JSON("changes_after", map[string]any{}).
			Optional(),
		field.Enum("result").
			Values("success", "failed").
			Default("success"),
		field.Text("error_message").
			Optional(),
		field.Enum("via").
			Values("web_ui", "mcp_tool", "api_token", "system").
			Default("web_ui"),
		field.String("mcp_tool_name").
			Optional().
			MaxLen(100),
		field.String("user_agent").
			Optional().
			MaxLen(255),
		// request_body JSONB
		field.JSON("request_body", map[string]any{}).
			Optional(),
		field.Int("response_status").
			Optional(),
		field.String("request_id").
			Optional().
			MaxLen(64),
		field.String("correlation_id").
			Optional().
			MaxLen(64),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the ProjectAuditLog.
func (ProjectAuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project", Project.Type).
			Unique().
			Required().
			Field("project_id"),
	}
}

// Indexes of the ProjectAuditLog.
func (ProjectAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// idx_audit_logs_proj_time: (project_id, created_at DESC)
		index.Fields("project_id", "created_at"),
		// idx_audit_logs_action
		index.Fields("action"),
		// idx_audit_logs_actor: (actor_id, created_at DESC)
		index.Fields("actor_id", "created_at"),
		// idx_audit_logs_resource: (resource_type, resource_id)
		index.Fields("resource_type", "resource_id"),
	}
}
