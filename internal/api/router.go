// Package api 提供 HTTP 路由注册。
// 中间件链：RequestID → Logging → CORS → RateLimit → Auth → ProjectContext → RBAC
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
	userSvc *service.UserService,
	memberSvc *service.MemberService,
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

		// 项目列表
		authed.GET("/projects", projectH.List)

		// 项目创建（需要 owner/admin 角色）
		authed.POST("/projects", projectH.Create)

		// 项目详情路由
		projects := authed.Group("/projects/:projectID")
		projects.Use(middleware.ProjectContext())
		{
			projects.GET("", projectH.Get)

			// 域名列表
			projects.GET("/domains", middleware.RequirePermission(memberSvc, "domain", "read"), domainH.List)

			// 域名创建
			projects.POST("/domains", middleware.RequirePermission(memberSvc, "domain", "create"), domainH.Create)

			// 成员列表
			projects.GET("/members", middleware.RequirePermission(memberSvc, "member", "read"), memberH.List)

			// 成员邀请
			projects.POST("/members", middleware.RequirePermission(memberSvc, "member", "create"), memberH.Invite)

			// 成员角色变更
			projects.PATCH("/members/:userID", middleware.RequirePermission(memberSvc, "member", "update"), memberH.ChangeRole)

			// 成员移除
			projects.DELETE("/members/:userID", middleware.RequirePermission(memberSvc, "member", "delete"), memberH.Remove)

			// 审计日志
			projects.GET("/audit-logs", middleware.RequirePermission(memberSvc, "audit", "read"), auditH.List)
		}

		// 单个域名操作
		domain := authed.Group("/projects/:projectID/domains/:domainID")
		domain.Use(middleware.ProjectContext())
		{
			domain.GET("", middleware.RequirePermission(memberSvc, "domain", "read"), domainH.Get)
			domain.DELETE("", middleware.RequirePermission(memberSvc, "domain", "delete"), domainH.Delete)

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