// Package service 提供业务逻辑层通用错误定义。
package service

import "errors"

// 预定义业务错误，供 Handler 层进行错误类型判断和 HTTP 状态码映射。
var (
	// ErrConcurrentModification 表示乐观锁冲突，实体状态已被其他操作修改。
	ErrConcurrentModification = errors.New("并发修改冲突，请刷新后重试")

	// ErrInvalidTransition 表示状态转换不合法。
	ErrInvalidTransition = errors.New("非法的状态转换")

	// ErrNotFound 表示请求的资源不存在。
	ErrNotFound = errors.New("资源不存在")

	// ErrForbidden 表示无权访问该资源。
	ErrForbidden = errors.New("无权访问该资源")

	// ErrUnauthorized 表示未认证或认证已过期。
	ErrUnauthorized = errors.New("未认证或认证已过期")

	// ErrDuplicate 表示资源已存在（如邮箱已注册、域名已存在）。
	ErrDuplicate = errors.New("资源已存在")

	// ErrQuotaExceeded 表示项目配额已超限。
	ErrQuotaExceeded = errors.New("配额已超限")

	// ErrProjectSuspended 表示项目已冻结，拒绝写操作。
	ErrProjectSuspended = errors.New("项目已冻结")

	// ErrValidation 表示输入参数校验失败。
	ErrValidation = errors.New("输入参数校验失败")
)
