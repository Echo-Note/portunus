// Package dto 定义请求和响应数据传输对象（DTO）。
// Handler 层只负责参数绑定、调用 Service、将响应序列化为 DTO。
package dto

import (
	"time"

	"github.com/google/uuid"
)

// ── 通用 ──

// ErrorResponse 统一错误响应。
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

// PaginatedResponse 分页响应包装。
type PaginatedResponse struct {
	Data       any `json:"data"`
	Total      int `json:"total"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
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

// TokenResponse 令牌响应。
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// ── 项目 ──

// CreateProjectRequest 创建项目请求。
type CreateProjectRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
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

// ── 域名 ──

// CreateDomainRequest 创建域名请求。
type CreateDomainRequest struct {
	DomainName string `json:"domain_name" binding:"required"`
	SSLEnabled bool   `json:"ssl_enabled"`
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

// ── 共享 ──

// CreateShareRequest 创建共享请求。
type CreateShareRequest struct {
	TargetProjectID uuid.UUID `json:"target_project_id" binding:"required"`
	Permission      string    `json:"permission" binding:"required,oneof=read_only edit"`
}

// ── 审计 ──

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