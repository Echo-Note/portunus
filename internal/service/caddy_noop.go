package service

import (
	"context"

	"github.com/Echo-Note/portunus/internal/caddy"
)

// CaddyClient 定义 Caddy Admin API 客户端接口。
// 用于 Service 层依赖注入，支持生产环境的真实客户端和开发环境的空操作客户端。
type CaddyClient interface {
	GetID(ctx context.Context, name string) ([]byte, string, error)
	PostID(ctx context.Context, name string, body any, etag string) ([]byte, error)
	PatchID(ctx context.Context, name string, body any, etag string) ([]byte, error)
	DeleteID(ctx context.Context, name string, etag string) error
	GetConfig(ctx context.Context) ([]byte, error)
	PostLoad(ctx context.Context, configJSON []byte) error
	GetUpstreams(ctx context.Context) ([]byte, error)
}

// NewNoopCaddyClient 创建一个不执行实际 Caddy 操作的客户端（用于开发环境）。
func NewNoopCaddyClient() CaddyClient {
	return caddy.NewNoopCaddyClient()
}
