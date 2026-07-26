// Package dto 定义请求和响应数据传输对象（DTO）。
// 遵循 OpenAPI 规范中定义的统一响应格式。
package dto

import (
	"time"

	"github.com/google/uuid"
)

// ── 统一响应格式 ──

// BaseResponse 统一响应外层包装。
// 所有 API 响应均使用此格式。
type BaseResponse struct {
	Code      int       `json:"code"`       // 业务状态码，0 表示成功
	Message   string    `json:"message"`    // 人类可读的消息
	Data      any       `json:"data"`       // 响应数据
	RequestID string    `json:"request_id"` // 请求唯一标识
	Timestamp time.Time `json:"timestamp"`  // 响应时间戳
}

// Success 构建成功响应。
func Success(data any) BaseResponse {
	return BaseResponse{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now(),
	}
}

// SuccessWithMeta 构建带元数据的成功响应（分页等）。
func SuccessWithMeta(data any, meta any) BaseResponse {
	return BaseResponse{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now(),
	}
}

// Error 构建错误响应。
func Error(code int, message string) BaseResponse {
	return BaseResponse{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// ErrorWithDetails 构建带详细信息的错误响应。
func ErrorWithDetails(code int, message string, details any) BaseResponse {
	return BaseResponse{
		Code:      code,
		Message:   message,
		Data:      details,
		Timestamp: time.Now(),
	}
}

// ── 业务错误码 ──

const (
	// CodeOK 成功。
	CodeOK = 0

	// CodeBadRequest 参数校验失败。
	CodeBadRequest = 40001

	// CodeUnauthorized 未认证。
	CodeUnauthorized = 40101

	// CodeTokenExpired Token 已过期。
	CodeTokenExpired = 40102

	// CodeForbidden 无权限。
	CodeForbidden = 40301

	// CodeProjectSuspended 项目已冻结。
	CodeProjectSuspended = 40302

	// CodeNotFound 资源不存在。
	CodeNotFound = 40401

	// CodeConflict 资源已存在。
	CodeConflict = 40901

	// CodeStateConflict 状态冲突（乐观锁）。
	CodeStateConflict = 40902

	// CodeRateLimited 请求频率超限。
	CodeRateLimited = 42901

	// CodeInternalError 内部错误。
	CodeInternalError = 50001

	// CodeCaddyUnreachable Caddy 不可达。
	CodeCaddyUnreachable = 50201
)

// ── 通用 ──

// HealthResponse 健康检查响应。
type HealthResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime,omitempty"`
}

// ── 认证 ──

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest 刷新令牌请求。
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthResponse 认证响应。
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email,omitempty"`
}

// ForgotPasswordRequest 忘记密码请求。
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest 重置密码请求。
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ── 用户 ──

// UserResponse 用户信息响应。
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateUserRequest 更新用户请求。
type UpdateUserRequest struct {
	Email string `json:"email,omitempty"`
}

// CreateTokenRequest 创建 API Token 请求。
type CreateTokenRequest struct {
	Name      string    `json:"name" binding:"required"`
	ProjectID uuid.UUID `json:"project_id" binding:"required"`
	Scopes    []string  `json:"scopes"`
}

// CreateTokenResponse 创建 API Token 响应。
type CreateTokenResponse struct {
	Token       string     `json:"token"`
	TokenID     uuid.UUID  `json:"token_id"`
	TokenPrefix string     `json:"token_prefix"`
	Name        string     `json:"name"`
	ProjectID   uuid.UUID  `json:"project_id"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ── 项目 ──

// CreateProjectRequest 创建项目请求。
type CreateProjectRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateProjectRequest 更新项目请求。
type UpdateProjectRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// ProjectResponse 项目响应。
type ProjectResponse struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	Plan        string    `json:"plan"`
	Environment string    `json:"environment"`
	MaxDomains  int       `json:"max_domains"`
	MaxMembers  int       `json:"max_members"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectListResponse 项目列表响应。
type ProjectListResponse struct {
	Items []ProjectResponse `json:"items"`
	Total int               `json:"total"`
}

// ── 域名 ──

// CreateDomainRequest 创建域名请求。
type CreateDomainRequest struct {
	DomainName string `json:"domain_name" binding:"required"`
	SSLEnabled bool   `json:"ssl_enabled"`
}

// UpdateDomainRequest 更新域名请求。
type UpdateDomainRequest struct {
	DomainName string `json:"domain_name,omitempty"`
	SSLEnabled *bool  `json:"ssl_enabled,omitempty"` // 使用指针区分"不传"和"传 false"
}

// DomainResponse 域名响应。
type DomainResponse struct {
	ID         uuid.UUID `json:"id"`
	DomainName string    `json:"domain_name"`
	Status     string    `json:"status"`
	SSLEnabled bool      `json:"ssl_enabled"`
	CaddyID    string    `json:"caddy_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ── 代理配置 ──

// ProxyConfigResponse 代理配置响应。
type ProxyConfigResponse struct {
	ID                  uuid.UUID `json:"id"`
	DomainID            uuid.UUID `json:"domain_id"`
	LbPolicy            string    `json:"lb_policy"`
	HealthCheckURI      string    `json:"health_check_uri,omitempty"`
	HealthCheckInterval string    `json:"health_check_interval"`
	Timeout             string    `json:"timeout"`
	Status              string    `json:"status"`
}

// UpdateProxyRequest 更新代理配置请求。
type UpdateProxyRequest struct {
	LbPolicy            string `json:"lb_policy,omitempty"`
	HealthCheckURI      string `json:"health_check_uri,omitempty"`
	HealthCheckInterval string `json:"health_check_interval,omitempty"`
	Timeout             string `json:"timeout,omitempty"`
}

// UpstreamStatusResponse 上游健康状态响应。
type UpstreamStatusResponse struct {
	UpstreamID  uuid.UUID `json:"upstream_id"`
	DialAddress string    `json:"dial_address"`
	Status      string    `json:"status"`
	Healthy     bool      `json:"healthy"`
	Fails       int       `json:"fails,omitempty"`
}

// ── 上游 ──

// AddUpstreamRequest 添加上游请求。
type AddUpstreamRequest struct {
	DialAddress string `json:"dial_address" binding:"required"`
	Weight      int    `json:"weight"`
	MaxRequests int    `json:"max_requests"`
}

// ── 成员 ──

// InviteMemberRequest 邀请成员请求。
type InviteMemberRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=admin editor viewer"`
}

// ChangeRoleRequest 变更角色请求。
type ChangeRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin editor viewer"`
}

// ── 邀请 ──

// InvitationResponse 邀请详情响应。
type InvitationResponse struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	Status          string    `json:"status"`
	InvitationToken string    `json:"invitation_token"`
	ExpiresAt       time.Time `json:"expires_at"`
	ProjectName     string    `json:"project_name,omitempty"`
	ProjectID       string    `json:"project_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ── 共享 ──

// CreateShareRequest 创建共享请求。
type CreateShareRequest struct {
	TargetProjectID uuid.UUID `json:"target_project_id" binding:"required"`
	Permission      string    `json:"permission" binding:"required,oneof=read_only edit"`
}

// ShareResponse 共享响应。
type ShareResponse struct {
	ID              uuid.UUID  `json:"id"`
	DomainID        uuid.UUID  `json:"domain_id"`
	SourceProjectID uuid.UUID  `json:"source_project_id"`
	TargetProjectID uuid.UUID  `json:"target_project_id"`
	Permission      string     `json:"permission"`
	Status          string     `json:"status"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ── API Token ──

// TokenResponse API Token 响应（不含明文 Token）。
type TokenResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"token_prefix"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Scopes      []string   `json:"scopes"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ── 审计 ──

// AuditLogQueryParams 审计日志查询参数。
type AuditLogQueryParams struct {
	Action       string `form:"action"`
	ResourceType string `form:"resource_type"`
	Limit        int    `form:"limit"`
	Offset       int    `form:"offset"`
}

// AuditLogResponse 审计日志响应。
type AuditLogResponse struct {
	ID           int64     `json:"id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	ActorType    string    `json:"actor_type"`
	ActorName    string    `json:"actor_name"`
	Result       string    `json:"result"`
	Via          string    `json:"via"`
	CreatedAt    time.Time `json:"created_at"`
}
