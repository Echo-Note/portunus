// Package handler 提供 HTTP Handler 实现。
// 严格遵守"薄 Handler"原则：只负责参数绑定、上下文提取、错误映射和响应序列化。
// 所有响应使用 OpenAPI 定义的统一格式 {code, message, data, request_id, timestamp}。
package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/internal/api/dto"
	"github.com/Echo-Note/portunus/internal/api/middleware"
	"github.com/Echo-Note/portunus/internal/service"
)

// respond 发送统一格式的成功响应。
func respond(c *gin.Context, status int, data any) {
	c.JSON(status, dto.BaseResponse{
		Code:      0,
		Message:   "success",
		Data:      data,
		RequestID: middleware.GetRequestID(c),
		Timestamp: time.Now(),
	})
}

// respondError 发送统一格式的错误响应。
func respondError(c *gin.Context, status int, code int, message string) {
	c.JSON(status, dto.BaseResponse{
		Code:      code,
		Message:   message,
		RequestID: middleware.GetRequestID(c),
		Timestamp: time.Now(),
	})
}

// ── AuthHandler ──

// AuthHandler 认证相关的 HTTP Handler。
type AuthHandler struct {
	userSvc *service.UserService
}

// NewAuthHandler 创建认证 Handler 实例。
func NewAuthHandler(userSvc *service.UserService) *AuthHandler {
	return &AuthHandler{userSvc: userSvc}
}

// Register 用户注册。POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	result, err := h.userSvc.Register(c.Request.Context(), service.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusCreated, gin.H{
		"user_id": result.UserID,
		"email":   result.Email,
	})
}

// Login 用户登录。POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	pair, err := h.userSvc.Login(c.Request.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, dto.AuthResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
		TokenType:    pair.TokenType,
	})
}

// RefreshToken 刷新令牌。POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	pair, err := h.userSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, dto.AuthResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
		TokenType:    pair.TokenType,
	})
}

// Logout 退出登录。POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Token 撤销由客户端负责（删除本地 token），服务端仅记录日志
	userID, _ := middleware.GetUserID(c)
	slog.InfoContext(c.Request.Context(), "用户登出", "user_id", userID)
	respond(c, http.StatusOK, gin.H{"message": "已退出登录"})
}

// ── ProjectHandler ──

// ProjectHandler 项目相关的 HTTP Handler。
type ProjectHandler struct {
	projectSvc *service.ProjectService
}

// NewProjectHandler 创建项目 Handler 实例。
func NewProjectHandler(projectSvc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectSvc: projectSvc}
}

// Create 创建项目。POST /api/v1/projects
func (h *ProjectHandler) Create(c *gin.Context) {
	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	userID, _ := middleware.GetUserID(c)

	p, err := h.projectSvc.CreateProject(c.Request.Context(), service.CreateProjectInput{
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusCreated, dto.ProjectResponse{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		Plan:        string(p.Plan),
		Environment: string(p.Environment),
		MaxDomains:  p.MaxDomains,
		MaxMembers:  p.MaxMembers,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	})
}

// Get 获取项目详情。GET /api/v1/projects/:projectID
func (h *ProjectHandler) Get(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	p, err := h.projectSvc.GetProject(c.Request.Context(), projectID)
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, dto.ProjectResponse{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		Plan:        string(p.Plan),
		Environment: string(p.Environment),
		MaxDomains:  p.MaxDomains,
		MaxMembers:  p.MaxMembers,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	})
}

// List 列出用户的项目。GET /api/v1/projects
func (h *ProjectHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	projects, err := h.projectSvc.ListUserProjects(c.Request.Context(), userID)
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	var items []dto.ProjectResponse
	for _, p := range projects {
		items = append(items, dto.ProjectResponse{
			ID:          p.ID,
			ProjectID:   p.ProjectID,
			Name:        p.Name,
			Description: p.Description,
			Status:      string(p.Status),
			Plan:        string(p.Plan),
			Environment: string(p.Environment),
			MaxDomains:  p.MaxDomains,
			MaxMembers:  p.MaxMembers,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	respond(c, http.StatusOK, dto.ProjectListResponse{Items: items, Total: len(items)})
}

// Update 更新项目元数据。PATCH /api/v1/projects/:projectID
func (h *ProjectHandler) Update(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	p, err := h.projectSvc.UpdateProject(c.Request.Context(), projectID, service.UpdateProjectInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, dto.ProjectResponse{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		Plan:        string(p.Plan),
		Environment: string(p.Environment),
		MaxDomains:  p.MaxDomains,
		MaxMembers:  p.MaxMembers,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	})
}

// Delete 删除项目。DELETE /api/v1/projects/:projectID
func (h *ProjectHandler) Delete(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	actorID, _ := middleware.GetUserID(c)

	if err := h.projectSvc.DeleteProject(c.Request.Context(), projectID, actorID); err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusAccepted, gin.H{"message": "项目删除已接受，异步清理中"})
}

// Suspend 冻结项目。POST /api/v1/projects/:projectID/suspend
func (h *ProjectHandler) Suspend(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	actorID, _ := middleware.GetUserID(c)

	if err := h.projectSvc.SuspendProject(c.Request.Context(), projectID, actorID, "管理员冻结项目"); err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{"message": "项目已冻结"})
}

// Unsuspend 解冻项目。POST /api/v1/projects/:projectID/unsuspend
func (h *ProjectHandler) Unsuspend(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	actorID, _ := middleware.GetUserID(c)

	if err := h.projectSvc.ReactivateProject(c.Request.Context(), projectID, actorID); err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{"message": "项目已解冻"})
}

// ── DomainHandler ──

// DomainHandler 域名相关的 HTTP Handler。
type DomainHandler struct {
	domainSvc *service.DomainService
}

// NewDomainHandler 创建域名 Handler 实例。
func NewDomainHandler(domainSvc *service.DomainService) *DomainHandler {
	return &DomainHandler{domainSvc: domainSvc}
}

// Create 创建域名。POST /api/v1/projects/:projectID/domains
func (h *DomainHandler) Create(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	var req dto.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	d, err := h.domainSvc.CreateDomain(c.Request.Context(), service.CreateDomainInput{
		ProjectID:  projectID,
		DomainName: req.DomainName,
		SslEnabled: req.SSLEnabled,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusCreated, dto.DomainResponse{
		ID:         d.ID,
		DomainName: d.DomainName,
		Status:     string(d.Status),
		SSLEnabled: d.SslEnabled,
		CaddyID:    d.CaddyID,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	})
}

// List 列出项目下的域名。GET /api/v1/projects/:projectID/domains
func (h *DomainHandler) List(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	domains, err := h.domainSvc.ListDomainsByProject(c.Request.Context(), projectID)
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	var result []dto.DomainResponse
	for _, d := range domains {
		result = append(result, dto.DomainResponse{
			ID:         d.ID,
			DomainName: d.DomainName,
			Status:     string(d.Status),
			SSLEnabled: d.SslEnabled,
			CaddyID:    d.CaddyID,
			CreatedAt:  d.CreatedAt,
			UpdatedAt:  d.UpdatedAt,
		})
	}

	respond(c, http.StatusOK, gin.H{"items": result, "total": len(result)})
}

// Get 获取域名详情。GET /api/v1/projects/:projectID/domains/:domainID
func (h *DomainHandler) Get(c *gin.Context) {
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的域名 ID")
		return
	}

	d, err := h.domainSvc.GetDomain(c.Request.Context(), domainID)
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, dto.DomainResponse{
		ID:         d.ID,
		DomainName: d.DomainName,
		Status:     string(d.Status),
		SSLEnabled: d.SslEnabled,
		CaddyID:    d.CaddyID,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	})
}

// Delete 删除域名。DELETE /api/v1/projects/:projectID/domains/:domainID
func (h *DomainHandler) Delete(c *gin.Context) {
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的域名 ID")
		return
	}

	if err := h.domainSvc.DeleteDomain(c.Request.Context(), domainID); err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusNoContent, nil)
}

// ── ProxyHandler ──

// ProxyHandler 代理配置和上游相关的 HTTP Handler。
type ProxyHandler struct {
	proxySvc *service.ProxyService
}

// NewProxyHandler 创建代理 Handler 实例。
func NewProxyHandler(proxySvc *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{proxySvc: proxySvc}
}

// AddUpstream 添加上游。POST /api/v1/projects/:projectID/domains/:domainID/proxy/upstreams
func (h *ProxyHandler) AddUpstream(c *gin.Context) {
	proxyConfigID, err := uuid.Parse(c.Param("proxyConfigID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的代理配置 ID")
		return
	}

	var req dto.AddUpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	u, err := h.proxySvc.AddUpstream(c.Request.Context(), service.AddUpstreamInput{
		ProxyConfigID: proxyConfigID,
		DialAddress:   req.DialAddress,
		Weight:        req.Weight,
		MaxRequests:   req.MaxRequests,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusCreated, gin.H{
		"id":              u.ID,
		"dial_address":    u.DialAddress,
		"weight":          u.Weight,
		"status":          string(u.Status),
		"proxy_config_id": u.ProxyConfigID,
	})
}

// ListUpstreams 列出上游。GET /api/v1/projects/:projectID/domains/:domainID/proxy/upstreams
func (h *ProxyHandler) ListUpstreams(c *gin.Context) {
	proxyConfigID, err := uuid.Parse(c.Param("proxyConfigID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的代理配置 ID")
		return
	}

	upstreams, err := h.proxySvc.ListUpstreams(c.Request.Context(), proxyConfigID)
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{"items": upstreams, "total": len(upstreams)})
}

// RemoveUpstream 移除上游。DELETE /api/v1/projects/:projectID/domains/:domainID/proxy/upstreams/:upstreamID
func (h *ProxyHandler) RemoveUpstream(c *gin.Context) {
	upstreamID, err := uuid.Parse(c.Param("upstreamID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的上游 ID")
		return
	}

	if err := h.proxySvc.RemoveUpstream(c.Request.Context(), upstreamID); err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusNoContent, nil)
}

// ── MemberHandler ──

// MemberHandler 成员管理相关的 HTTP Handler。
type MemberHandler struct {
	memberSvc *service.MemberService
}

// NewMemberHandler 创建成员 Handler 实例。
func NewMemberHandler(memberSvc *service.MemberService) *MemberHandler {
	return &MemberHandler{memberSvc: memberSvc}
}

// Invite 邀请成员。POST /api/v1/projects/:projectID/members
func (h *MemberHandler) Invite(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	userID, _ := middleware.GetUserID(c)

	var req dto.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	inv, err := h.memberSvc.InviteMember(c.Request.Context(), service.InviteMemberInput{
		ProjectID: projectID,
		Email:     req.Email,
		Role:      req.Role,
		InvitedBy: userID,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusCreated, gin.H{
		"id":               inv.ID,
		"email":            inv.Email,
		"role":             string(inv.Role),
		"status":           string(inv.Status),
		"invitation_token": inv.InvitationToken,
		"expires_at":       inv.ExpiresAt,
	})
}

// List 列出成员。GET /api/v1/projects/:projectID/members
func (h *MemberHandler) List(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	members, err := h.memberSvc.ListMembers(c.Request.Context(), projectID)
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	// 脱敏处理：移除敏感字段（如 password_hash）
	var sanitized []gin.H
	for _, m := range members {
		item := gin.H{
			"user_id":    m.UserID,
			"project_id": m.ProjectID,
			"role":       m.Role,
			"status":     m.Status,
			"joined_at":  m.JoinedAt,
		}
		if m.Edges.User != nil {
			item["user"] = gin.H{
				"id":    m.Edges.User.ID,
				"email": m.Edges.User.Email,
				"status": m.Edges.User.Status,
			}
		}
		sanitized = append(sanitized, item)
	}

	respond(c, http.StatusOK, gin.H{"items": sanitized, "total": len(sanitized)})
}

// Remove 移除成员。DELETE /api/v1/projects/:projectID/members/:userID
func (h *MemberHandler) Remove(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	memberUserID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的用户 ID")
		return
	}

	actorID, _ := middleware.GetUserID(c)

	if err := h.memberSvc.RemoveMember(c.Request.Context(), projectID, memberUserID, actorID); err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusNoContent, nil)
}

// ChangeRole 变更成员角色。PATCH /api/v1/projects/:projectID/members/:userID
func (h *MemberHandler) ChangeRole(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	memberUserID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的用户 ID")
		return
	}

	var req dto.ChangeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	if err := h.memberSvc.ChangeMemberRole(c.Request.Context(), projectID, memberUserID, req.Role); err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{"message": "角色已变更"})
}

// ── ShareHandler ──

// ShareHandler 域名共享相关的 HTTP Handler。
type ShareHandler struct {
	shareSvc *service.ShareService
}

// NewShareHandler 创建共享 Handler 实例。
func NewShareHandler(shareSvc *service.ShareService) *ShareHandler {
	return &ShareHandler{shareSvc: shareSvc}
}

// Create 创建共享。POST /api/v1/projects/:projectID/domains/:domainID/shares
func (h *ShareHandler) Create(c *gin.Context) {
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的域名 ID")
		return
	}

	var req dto.CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "参数校验失败")
		return
	}

	userID, _ := middleware.GetUserID(c)
	projectID := middleware.GetProjectIDFromCtx(c)

	share, err := h.shareSvc.CreateShare(c.Request.Context(), service.CreateShareInput{
		DomainID:        domainID,
		SourceProjectID: projectID,
		TargetProjectID: req.TargetProjectID,
		Permission:      req.Permission,
		CreatedBy:       userID,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusCreated, gin.H{
		"id":                share.ID,
		"domain_id":         share.DomainID,
		"target_project_id": share.TargetProjectID,
		"permission":        string(share.Permission),
		"status":            string(share.Status),
	})
}

// List 列出域名的共享。GET /api/v1/projects/:projectID/domains/:domainID/shares
func (h *ShareHandler) List(c *gin.Context) {
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的域名 ID")
		return
	}

	shares, err := h.shareSvc.ListSharesByDomain(c.Request.Context(), domainID)
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusOK, gin.H{"items": shares, "total": len(shares)})
}

// Revoke 撤销共享。DELETE /api/v1/shares/:shareID
func (h *ShareHandler) Revoke(c *gin.Context) {
	shareID, err := uuid.Parse(c.Param("shareID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的共享 ID")
		return
	}

	if err := h.shareSvc.RevokeShare(c.Request.Context(), shareID); err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	respond(c, http.StatusNoContent, nil)
}

// ── AuditHandler ──

// AuditHandler 审计日志相关的 HTTP Handler。
type AuditHandler struct {
	auditSvc *service.AuditService
}

// NewAuditHandler 创建审计 Handler 实例。
func NewAuditHandler(auditSvc *service.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

// List 查询审计日志。GET /api/v1/projects/:projectID/audit-logs
func (h *AuditHandler) List(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, dto.CodeBadRequest, "无效的项目 ID")
		return
	}

	var params dto.AuditLogQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		params.Limit = 50
	}

	logs, err := h.auditSvc.Query(c.Request.Context(), service.QueryInput{
		ProjectID:    projectID,
		Action:       params.Action,
		ResourceType: params.ResourceType,
		Limit:        params.Limit,
		Offset:       params.Offset,
	})
	if err != nil {
		code, status := mapError(err)
		respondError(c, status, code, err.Error())
		return
	}

	var result []dto.AuditLogResponse
	for _, l := range logs {
		result = append(result, dto.AuditLogResponse{
			ID:           l.ID,
			Action:       l.Action,
			ResourceType: l.ResourceType,
			ResourceID:   l.ResourceID,
			ActorType:    string(l.ActorType),
			ActorName:    l.ActorName,
			Result:       string(l.Result),
			Via:          string(l.Via),
			CreatedAt:    l.CreatedAt,
		})
	}

	respond(c, http.StatusOK, gin.H{"items": result, "total": len(result)})
}

// ── 错误映射 ──

// mapError 将业务错误映射为 (业务错误码, HTTP 状态码)。
func mapError(err error) (int, int) {
	switch {
	case isWrapped(err, service.ErrNotFound):
		return dto.CodeNotFound, http.StatusNotFound
	case isWrapped(err, service.ErrUnauthorized):
		return dto.CodeUnauthorized, http.StatusUnauthorized
	case isWrapped(err, service.ErrForbidden):
		return dto.CodeForbidden, http.StatusForbidden
	case isWrapped(err, service.ErrConcurrentModification):
		return dto.CodeStateConflict, http.StatusConflict
	case isWrapped(err, service.ErrDuplicate):
		return dto.CodeConflict, http.StatusConflict
	case isWrapped(err, service.ErrQuotaExceeded):
		return dto.CodeForbidden, http.StatusForbidden
	case isWrapped(err, service.ErrValidation):
		return dto.CodeBadRequest, http.StatusBadRequest
	case isWrapped(err, service.ErrInvalidTransition):
		return dto.CodeStateConflict, http.StatusConflict
	case isWrapped(err, service.ErrProjectSuspended):
		return dto.CodeProjectSuspended, http.StatusForbidden
	default:
		return dto.CodeInternalError, http.StatusInternalServerError
	}
}

// isWrapped 检查错误链中是否包含目标错误。
func isWrapped(err, target error) bool {
	if err == nil {
		return false
	}
	for {
		if err == target {
			return true
		}
		type unwrapper interface {
			Unwrap() error
		}
		if ue, ok := err.(unwrapper); ok {
			err = ue.Unwrap()
			if err == nil {
				return false
			}
		} else {
			return false
		}
	}
}