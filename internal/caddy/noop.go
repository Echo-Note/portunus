package caddy

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// NewNoopClient 创建空操作 Caddy 客户端，所有 Caddy API 调用返回空成功。
// 用于开发环境，避免因 Caddy 不可达导致服务启动失败。
func NewNoopClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 1 * time.Second},
		adminURL:   "http://localhost:2021",
		retryMax:   0,
	}
}

// NoopCaddyClient 实现 Caddy 客户端接口的空操作版本。
// 所有方法打印 Debug 日志后返回空成功。
type NoopCaddyClient struct{}

// NewNoopCaddyClient 创建空操作 Caddy 客户端包装器。
func NewNoopCaddyClient() *NoopCaddyClient {
	return &NoopCaddyClient{}
}

// GetID 返回空 JSON 和 noop-etag。
func (n *NoopCaddyClient) GetID(ctx context.Context, name string) ([]byte, string, error) {
	slog.DebugContext(ctx, "noop caddy: GetID", "name", name)
	return []byte("{}"), "noop-etag", nil
}

// PostID 返回空 JSON。
func (n *NoopCaddyClient) PostID(ctx context.Context, name string, body any, etag string) ([]byte, error) {
	slog.DebugContext(ctx, "noop caddy: PostID", "name", name)
	return []byte("{}"), nil
}

// PatchID 返回空 JSON。
func (n *NoopCaddyClient) PatchID(ctx context.Context, name string, body any, etag string) ([]byte, error) {
	slog.DebugContext(ctx, "noop caddy: PatchID", "name", name)
	return []byte("{}"), nil
}

// DeleteID 返回 nil 错误。
func (n *NoopCaddyClient) DeleteID(ctx context.Context, name string, etag string) error {
	slog.DebugContext(ctx, "noop caddy: DeleteID", "name", name)
	return nil
}

// GetConfig 返回空 JSON。
func (n *NoopCaddyClient) GetConfig(ctx context.Context) ([]byte, error) {
	slog.DebugContext(ctx, "noop caddy: GetConfig")
	return []byte("{}"), nil
}

// PostLoad 返回 nil 错误。
func (n *NoopCaddyClient) PostLoad(ctx context.Context, configJSON []byte) error {
	slog.DebugContext(ctx, "noop caddy: PostLoad")
	return nil
}

// GetUpstreams 返回空数组。
func (n *NoopCaddyClient) GetUpstreams(ctx context.Context) ([]byte, error) {
	slog.DebugContext(ctx, "noop caddy: GetUpstreams")
	return []byte("[]"), nil
}
