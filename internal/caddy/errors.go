// Package caddy 提供 Caddy Admin API 客户端。
package caddy

import "errors"

// 预定义错误，用于调用方进行错误类型判断。
var (
	// ErrPreconditionFailed 表示乐观并发控制失败（412），etag 不匹配。
	ErrPreconditionFailed = errors.New("caddy precondition failed")

	// ErrCaddyUnreachable 表示 Caddy Admin API 不可达。
	ErrCaddyUnreachable = errors.New("caddy admin api unreachable")
)