// Package handler 提供 HTTP Handler 实现。
// 严格遵守"薄 Handler"原则：只负责参数绑定、上下文提取、错误映射和响应序列化。
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nrgao/portunus/internal/api/dto"
	"github.com/nrgao/portunus/internal/api/middleware"
	"github.com/nrgao/portunus/internal/service"
)

// AuthHandler 认证相关的 HTTP Handler。
type AuthHandler struct {
	userSvc *service.UserService
}

// NewAuthHandler 创建认证 Handler 实例。
func NewAuthHandler(userSvc *service.UserService) *AuthHandler {
	return &AuthHandler{userSvc: userSvc}
}

// Register 用户注册。
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "参数校验失败", Details: err.Error()})
		return
	}

	result, err := h.userSvc.Register(c.Request.Context(), service.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": result.UserID,
		"email":   result.Email,
	})
}

// Login 用户登录。
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "参数校验失败", Details: err.Error()})
		return
	}

	pair, err := h.userSvc.Login(c.Request.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
		TokenType:    pair.TokenType,
	})
}

// RefreshToken 刷新令牌。
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "参数校验失败", Details: err.Error()})
		return
	}

	pair, err := h.userSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
		TokenType:    pair.TokenType,
	})
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

// Create 创建项目。
// POST /api/v1/projects
func (h *ProjectHandler) Create(c *gin.Context) {
	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "参数校验失败", Details: err.Error()})
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
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.ProjectResponse{
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

// Get 获取项目详情。
// GET /api/v1/projects/:projectID
func (h *ProjectHandler) Get(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的项目 ID"})
		return
	}

	p, err := h.projectSvc.GetProject(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ProjectResponse{
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

// List 列出用户的项目。
// GET /api/v1/projects
func (h *ProjectHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	projects, err := h.projectSvc.ListUserProjects(c.Request.Context(), userID)
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	var result []dto.ProjectResponse
	for _, p := range projects {
		result = append(result, dto.ProjectResponse{
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

	c.JSON(http.StatusOK, gin.H{"data": result, "total": len(result)})
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

// Create 创建域名。
// POST /api/v1/projects/:projectID/domains
func (h *DomainHandler) Create(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的项目 ID"})
		return
	}

	var req dto.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "参数校验失败", Details: err.Error()})
		return
	}

	d, err := h.domainSvc.CreateDomain(c.Request.Context(), service.CreateDomainInput{
		ProjectID:  projectID,
		DomainName: req.DomainName,
		SslEnabled: req.SSLEnabled,
	})
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.DomainResponse{
		ID:         d.ID,
		DomainName: d.DomainName,
		Status:     string(d.Status),
		SSLEnabled: d.SslEnabled,
		CaddyID:    d.CaddyID,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	})
}

// List 列出项目下的域名。
// GET /api/v1/projects/:projectID/domains
func (h *DomainHandler) List(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的项目 ID"})
		return
	}

	domains, err := h.domainSvc.ListDomainsByProject(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
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

	c.JSON(http.StatusOK, gin.H{"data": result, "total": len(result)})
}

// Get 获取域名详情。
// GET /api/v1/projects/:projectID/domains/:domainID
func (h *DomainHandler) Get(c *gin.Context) {
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的域名 ID"})
		return
	}

	d, err := h.domainSvc.GetDomain(c.Request.Context(), domainID)
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.DomainResponse{
		ID:         d.ID,
		DomainName: d.DomainName,
		Status:     string(d.Status),
		SSLEnabled: d.SslEnabled,
		CaddyID:    d.CaddyID,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	})
}

// Delete 删除域名。
// DELETE /api/v1/projects/:projectID/domains/:domainID
func (h *DomainHandler) Delete(c *gin.Context) {
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的域名 ID"})
		return
	}

	if err := h.domainSvc.DeleteDomain(c.Request.Context(), domainID); err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
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

// AddUpstream 添加上游。
// POST /api/v1/projects/:projectID/domains/:domainID/upstreams
func (h *ProxyHandler) AddUpstream(c *gin.Context) {
	proxyConfigIDStr := c.Param("proxyConfigID")
	proxyConfigID, err := uuid.Parse(proxyConfigIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的代理配置 ID"})
		return
	}

	var req dto.AddUpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "参数校验失败", Details: err.Error()})
		return
	}

	u, err := h.proxySvc.AddUpstream(c.Request.Context(), service.AddUpstreamInput{
		ProxyConfigID: proxyConfigID,
		DialAddress:   req.DialAddress,
		Weight:        req.Weight,
		MaxRequests:   req.MaxRequests,
	})
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             u.ID,
		"dial_address":   u.DialAddress,
		"weight":         u.Weight,
		"status":         string(u.Status),
		"proxy_config_id": u.ProxyConfigID,
	})
}

// ListUpstreams 列出上游。
// GET /api/v1/projects/:projectID/domains/:domainID/upstreams
func (h *ProxyHandler) ListUpstreams(c *gin.Context) {
	proxyConfigIDStr := c.Param("proxyConfigID")
	proxyConfigID, err := uuid.Parse(proxyConfigIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的代理配置 ID"})
		return
	}

	upstreams, err := h.proxySvc.ListUpstreams(c.Request.Context(), proxyConfigID)
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	// 直接返回上游列表
	c.JSON(http.StatusOK, gin.H{"data": upstreams, "total": len(upstreams)})
}

// RemoveUpstream 移除上游。
// DELETE /api/v1/projects/:projectID/domains/:domainID/upstreams/:upstreamID
func (h *ProxyHandler) RemoveUpstream(c *gin.Context) {
	upstreamID, err := uuid.Parse(c.Param("upstreamID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的上游 ID"})
		return
	}

	if err := h.proxySvc.RemoveUpstream(c.Request.Context(), upstreamID); err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
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

// Invite 邀请成员。
// POST /api/v1/projects/:projectID/members
func (h *MemberHandler) Invite(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的项目 ID"})
		return
	}

	userID, _ := middleware.GetUserID(c)

	var req dto.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "参数校验失败", Details: err.Error()})
		return
	}

	inv, err := h.memberSvc.InviteMember(c.Request.Context(), service.InviteMemberInput{
		ProjectID: projectID,
		Email:     req.Email,
		Role:      req.Role,
		InvitedBy: userID,
	})
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            inv.ID,
		"email":         inv.Email,
		"role":          string(inv.Role),
		"status":        string(inv.Status),
		"invitation_token": inv.InvitationToken,
	})
}

// List 列出成员。
// GET /api/v1/projects/:projectID/members
func (h *MemberHandler) List(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的项目 ID"})
		return
	}

	members, err := h.memberSvc.ListMembers(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": members, "total": len(members)})
}

// Remove 移除成员。
// DELETE /api/v1/projects/:projectID/members/:userID
func (h *MemberHandler) Remove(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的项目 ID"})
		return
	}

	memberUserID, err := uuid.Parse(c.Param("userID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的用户 ID"})
		return
	}

	actorID, _ := middleware.GetUserID(c)

	if err := h.memberSvc.RemoveMember(c.Request.Context(), projectID, memberUserID, actorID); err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
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

// Create 创建共享。
// POST /api/v1/projects/:projectID/domains/:domainID/share
func (h *ShareHandler) Create(c *gin.Context) {
	domainID, err := uuid.Parse(c.Param("domainID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的域名 ID"})
		return
	}

	var req dto.CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "参数校验失败", Details: err.Error()})
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
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                share.ID,
		"domain_id":         share.DomainID,
		"target_project_id": share.TargetProjectID,
		"permission":        string(share.Permission),
		"status":            string(share.Status),
	})
}

// Revoke 撤销共享。
// DELETE /api/v1/projects/:projectID/domains/:domainID/share/:shareID
func (h *ShareHandler) Revoke(c *gin.Context) {
	shareID, err := uuid.Parse(c.Param("shareID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的共享 ID"})
		return
	}

	if err := h.shareSvc.RevokeShare(c.Request.Context(), shareID); err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
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

// List 查询审计日志。
// GET /api/v1/projects/:projectID/audit-logs
func (h *AuditHandler) List(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("projectID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "无效的项目 ID"})
		return
	}

	logs, err := h.auditSvc.Query(c.Request.Context(), service.QueryInput{
		ProjectID: projectID,
		Limit:     50,
	})
	if err != nil {
		c.JSON(mapErrorStatus(err), dto.ErrorResponse{Error: err.Error()})
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

	c.JSON(http.StatusOK, gin.H{"data": result, "total": len(result)})
}

// ── 错误映射 ──

// mapErrorStatus 将业务错误映射为 HTTP 状态码。
func mapErrorStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case isError(err, service.ErrNotFound):
		return http.StatusNotFound
	case isError(err, service.ErrUnauthorized):
		return http.StatusUnauthorized
	case isError(err, service.ErrForbidden):
		return http.StatusForbidden
	case isError(err, service.ErrConcurrentModification):
		return http.StatusConflict
	case isError(err, service.ErrDuplicate):
		return http.StatusConflict
	case isError(err, service.ErrQuotaExceeded):
		return http.StatusForbidden
	case isError(err, service.ErrValidation):
		return http.StatusBadRequest
	case isError(err, service.ErrInvalidTransition):
		return http.StatusConflict
	case isError(err, service.ErrProjectSuspended):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// isError 检查错误链中是否包含目标错误。
func isError(err, target error) bool {
	if err == nil {
		return false
	}
	return errorsIs(err, target)
}

// errorsIs 简单的错误链检查（避免依赖 errors.Is）。
func errorsIs(err, target error) bool {
	for {
		if err == target {
			return true
		}
		// 检查是否实现了 Unwrap
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