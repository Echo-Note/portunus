package schema

import (
	"time"

	"github.com/google/uuid"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ApiToken holds the schema definition for the ApiToken entity.
// API Token 表：用户为某项目颁发的访问令牌，按 scope 与 IP 白名单受限。
type ApiToken struct {
	ent.Schema
}

// Fields of the ApiToken.
func (ApiToken) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		// user_id REFERENCES users(id) ON DELETE CASCADE
		field.UUID("user_id", uuid.UUID{}),
		// project_id REFERENCES projects(id) ON DELETE CASCADE
		field.UUID("project_id", uuid.UUID{}),
		field.String("name").
			NotEmpty().
			MaxLen(255),
		// token_hash UNIQUE NOT NULL
		field.String("token_hash").
			NotEmpty().
			MaxLen(64).
			Unique(),
		// token_prefix NOT NULL
		field.String("token_prefix").
			NotEmpty().
			MaxLen(20),
		// scopes TEXT[] DEFAULT '{}'
		field.Strings("scopes").
			Default([]string{}),
		field.Time("expires_at").
			Optional(),
		field.Time("last_used_at").
			Optional(),
		// ip_whitelist INET[]，Ent 不原生支持 INET[]，使用 Strings 存储。
		field.Strings("ip_whitelist").
			Optional(),
		field.Enum("status").
			Values("active", "revoked", "expired").
			Default("active"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the ApiToken.
func (ApiToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Unique().
			Required().
			Field("user_id"),
		edge.To("project", Project.Type).
			Unique().
			Required().
			Field("project_id"),
	}
}

// Indexes of the ApiToken.
func (ApiToken) Indexes() []ent.Index {
	return []ent.Index{
		// idx_api_tokens_user: (user_id, status)
		index.Fields("user_id", "status"),
		// idx_api_tokens_hash: (token_hash) —— token_hash 已通过字段 Unique 建唯一约束，
		// 此处仍显式声明索引以便查询命中与命名对齐。
		index.Fields("token_hash"),
	}
}
