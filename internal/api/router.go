// Package api 提供 HTTP 路由注册。
// 中间件链：RequestID → Logging → CORS → RateLimit → Auth → ProjectContext → RBAC
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/internal/api/dto"
	"github.com/Echo-Note/portunus/internal/api/handler"
	"github.com/Echo-Note/portunus/internal/api/middleware"
	"github.com/Echo-Note/portunus/internal/service"
)

// RegisterRoutes 注册所有 API 路由。
func RegisterRoutes(
	r *gin.Engine,
	authH *handler.AuthHandler,
	projectH *handler.ProjectHandler,
	domainH *handler.DomainHandler,
	proxyH *handler.ProxyHandler,
	memberH *handler.MemberHandler,
	shareH *handler.ShareHandler,
	auditH *handler.AuditHandler,
	snapshotH *handler.SnapshotHandler,
	userSvc *service.UserService,
	memberSvc *service.MemberService,
	apiTokenSvc *service.ApiTokenService,
	rateLimitRPM, rateLimitBurst int,
) {
	// ── 全局中间件 ──
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logging())
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimit(rateLimitRPM, rateLimitBurst))

	// ── 健康检查（无需认证）──
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, dto.BaseResponse{
			Code:      0,
			Message:   "success",
			Data:      dto.HealthResponse{Status: "ok"},
			RequestID: middleware.GetRequestID(c),
		})
	})
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, dto.BaseResponse{
			Code:      0,
			Message:   "success",
			Data:      gin.H{"status": "ready"},
			RequestID: middleware.GetRequestID(c),
		})
	})

	// ── API v1 ──
	v1 := r.Group("/api/v1")

	// ── 认证路由（公开）──
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.RefreshToken)

		// 邮箱验证与密码重置（阶段 2）
		auth.POST("/verify-email", authH.VerifyEmail)
		auth.POST("/forgot-password", authH.ForgotPassword)
		auth.POST("/reset-password", authH.ResetPassword)

		// OAuth（阶段 2）
		auth.GET("/oauth/:provider", authH.OAuthRedirect)
		auth.GET("/oauth/:provider/callback", authH.OAuthCallback)
	}

	// ── 需要认证的路由 ──
	authed := v1.Group("")
	authed.Use(middleware.AuthMiddleware(userSvc))
	{
		// 退出登录
		authed.POST("/auth/logout", authH.Logout)

		// 用户管理
		authed.GET("/me", func(c *gin.Context) {
			userID, _ := middleware.GetUserID(c)
			u, err := userSvc.GetUser(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusNotFound, dto.Error(dto.CodeNotFound, err.Error()))
				return
			}
			c.JSON(http.StatusOK, dto.Success(dto.UserResponse{
				ID:        u.ID,
				Email:     u.Email,
				Status:    string(u.Status),
				CreatedAt: u.CreatedAt,
				UpdatedAt: u.UpdatedAt,
			}))
		})

		authed.PATCH("/me", func(c *gin.Context) {
			userID, _ := middleware.GetUserID(c)
			var req dto.UpdateUserRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, dto.Error(dto.CodeBadRequest, "参数校验失败"))
				return
			}
			u, err := userSvc.UpdateUser(c.Request.Context(), userID, service.UpdateUserInput{
				Email: req.Email,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, dto.Error(dto.CodeInternalError, err.Error()))
				return
			}
			c.JSON(http.StatusOK, dto.Success(dto.UserResponse{
				ID:        u.ID,
				Email:     u.Email,
				Status:    string(u.Status),
				CreatedAt: u.CreatedAt,
				UpdatedAt: u.UpdatedAt,
			}))
		})

		// API Token 管理
		authed.GET("/me/tokens", func(c *gin.Context) {
			userID, _ := middleware.GetUserID(c)
			tokens, err := apiTokenSvc.ListApiTokens(c.Request.Context(), service.ListApiTokensInput{
				UserID: userID,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, dto.Error(dto.CodeInternalError, err.Error()))
				return
			}
			var items []dto.TokenResponse
			for _, t := range tokens {
				var lastUsedAt *time.Time
				if !t.LastUsedAt.IsZero() {
					lastUsedAt = &t.LastUsedAt
				}
				var expiresAt *time.Time
				if !t.ExpiresAt.IsZero() {
					expiresAt = &t.ExpiresAt
				}
				items = append(items, dto.TokenResponse{
					ID:          t.ID,
					Name:        t.Name,
					TokenPrefix: t.TokenPrefix,
					ProjectID:   t.ProjectID,
					Scopes:      t.Scopes,
					LastUsedAt:  lastUsedAt,
					ExpiresAt:   expiresAt,
					Status:      string(t.Status),
					CreatedAt:   t.CreatedAt,
				})
			}
			c.JSON(http.StatusOK, dto.Success(gin.H{"items": items, "total": len(items)}))
		})

		authed.POST("/me/tokens", func(c *gin.Context) {
			userID, _ := middleware.GetUserID(c)
			var req dto.CreateTokenRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, dto.Error(dto.CodeBadRequest, "参数校验失败"))
				return
			}
			result, err := apiTokenSvc.CreateApiToken(c.Request.Context(), service.CreateApiTokenInput{
				UserID:    userID,
				ProjectID: req.ProjectID,
				Name:      req.Name,
				Scopes:    req.Scopes,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, dto.Error(dto.CodeInternalError, err.Error()))
				return
			}
			c.JSON(http.StatusCreated, dto.Success(dto.CreateTokenResponse{
				Token:       result.Token,
				TokenID:     result.TokenID,
				TokenPrefix: result.TokenPrefix,
				Name:        result.Name,
				ProjectID:   result.ProjectID,
			}))
		})

		authed.DELETE("/me/tokens/:tid", func(c *gin.Context) {
			userID, _ := middleware.GetUserID(c)
			tokenID, err := uuid.Parse(c.Param("tid"))
			if err != nil {
				c.JSON(http.StatusBadRequest, dto.Error(dto.CodeBadRequest, "无效的 Token ID"))
				return
			}
			if err := apiTokenSvc.RevokeApiToken(c.Request.Context(), tokenID, userID); err != nil {
				c.JSON(http.StatusNotFound, dto.Error(dto.CodeNotFound, err.Error()))
				return
			}
			c.JSON(http.StatusOK, dto.Success(gin.H{"message": "API Token 已撤销"}))
		})

		// 项目列表
		authed.GET("/projects", projectH.List)

		// 项目创建（需要 owner/admin 角色）
		authed.POST("/projects", projectH.Create)

		// 邀请相关路由（公开访问，但需要认证）
		authed.GET("/invitations/:token", memberH.GetInvitation)
		authed.POST("/invitations/:token/accept", memberH.AcceptInvitation)
		authed.POST("/invitations/:token/reject", memberH.RejectInvitation)

		// 收到的共享列表
		authed.GET("/shares", shareH.ListReceived)
		authed.POST("/shares/:shareID/accept", shareH.AcceptShare)

		// 项目详情路由
		projects := authed.Group("/projects/:projectID")
		projects.Use(middleware.ProjectContext())
		{
			projects.GET("", projectH.Get)

			// 项目更新（需要 owner/admin 角色）
			projects.PATCH("", middleware.RequirePermission(memberSvc, "project", "update"), projectH.Update)

			// 项目删除（需要 owner 角色）
			projects.DELETE("", middleware.RequirePermission(memberSvc, "project", "delete"), projectH.Delete)

			// 项目冻结/解冻（需要 owner/admin 角色）
			projects.POST("/suspend", middleware.RequirePermission(memberSvc, "project", "update"), projectH.Suspend)
			projects.POST("/unsuspend", middleware.RequirePermission(memberSvc, "project", "update"), projectH.Unsuspend)

			// 域名列表
			projects.GET("/domains", middleware.RequirePermission(memberSvc, "domain", "read"), domainH.List)

			// 域名创建
			projects.POST("/domains", middleware.RequirePermission(memberSvc, "domain", "create"), domainH.Create)

			// 成员列表
			projects.GET("/members", middleware.RequirePermission(memberSvc, "member", "read"), memberH.List)

			// 成员邀请
			projects.POST("/members", middleware.RequirePermission(memberSvc, "member", "create"), memberH.Invite)

			// 成员退出
			projects.POST("/members/me/leave", memberH.Leave)

			// 成员角色变更
			projects.PATCH("/members/:userID", middleware.RequirePermission(memberSvc, "member", "update"), memberH.ChangeRole)

			// 成员移除
			projects.DELETE("/members/:userID", middleware.RequirePermission(memberSvc, "member", "delete"), memberH.Remove)

			// 审计日志
			projects.GET("/audit-logs", middleware.RequirePermission(memberSvc, "audit", "read"), auditH.List)

			// 快照管理（阶段 2）
			projects.GET("/snapshots", middleware.RequirePermission(memberSvc, "project", "read"), snapshotH.List)
			projects.POST("/snapshots/:version/rollback", middleware.RequirePermission(memberSvc, "project", "update"), snapshotH.Rollback)
		}

		// 单个域名操作
		domain := authed.Group("/projects/:projectID/domains/:domainID")
		domain.Use(middleware.ProjectContext())
		{
			domain.GET("", middleware.RequirePermission(memberSvc, "domain", "read"), domainH.Get)
			domain.PATCH("", middleware.RequirePermission(memberSvc, "domain", "update"), domainH.Update)
			domain.DELETE("", middleware.RequirePermission(memberSvc, "domain", "delete"), domainH.Delete)

			// 代理配置
			domain.GET("/proxy", middleware.RequirePermission(memberSvc, "proxy", "read"), proxyH.GetProxyConfig)
			domain.PATCH("/proxy", middleware.RequirePermission(memberSvc, "proxy", "update"), proxyH.UpdateProxyConfig)

			// 上游健康状态
			domain.GET("/status", middleware.RequirePermission(memberSvc, "upstream", "read"), proxyH.GetUpstreamStatus)

			// 域名共享
			domain.POST("/shares", middleware.RequirePermission(memberSvc, "share", "create"), shareH.Create)
			domain.GET("/shares", middleware.RequirePermission(memberSvc, "share", "read"), shareH.List)
		}

		// 上游管理
		upstreams := authed.Group("/projects/:projectID/domains/:domainID/proxy/upstreams")
		upstreams.Use(middleware.ProjectContext())
		{
			upstreams.GET("", middleware.RequirePermission(memberSvc, "upstream", "read"), proxyH.ListUpstreams)
			upstreams.POST("", middleware.RequirePermission(memberSvc, "upstream", "create"), proxyH.AddUpstream)
			upstreams.DELETE("/:upstreamID", middleware.RequirePermission(memberSvc, "upstream", "delete"), proxyH.RemoveUpstream)
		}

		// 共享撤销
		authed.DELETE("/shares/:shareID", shareH.Revoke)
	}
}
