// Package api 提供 HTTP 路由注册。
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/nrgao/portunus/internal/api/handler"
	"github.com/nrgao/portunus/internal/api/middleware"
	"github.com/nrgao/portunus/internal/service"
)

// RegisterRoutes 注册所有 API 路由。
// 中间件链：Auth → RBAC+Ownership → RateLimit → Audit
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
) {
	// 健康检查（无需认证）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")

	// ── 认证路由（公开）──
	auth := api.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.RefreshToken)
	}

	// ── 需要认证的路由 ──
	authenticated := api.Group("")
	authenticated.Use(middleware.AuthMiddleware(userSvc))
	{
		// 项目列表
		authenticated.GET("/projects", projectH.List)

		// 项目 CRUD
		projects := authenticated.Group("/projects/:projectID")
		{
			projects.GET("", projectH.Get)
		}

		// 项目域名（需要 RBAC —— 域名 read 权限）
		domains := projects.Group("/domains")
		domains.Use(middleware.RequirePermission(memberSvc, "domain", "read"))
		{
			domains.GET("", domainH.List)
		}
		// 域名创建（需要 RBAC —— 域名 create 权限）
		domains.POST("", middleware.RequirePermission(memberSvc, "domain", "create"), domainH.Create)

		// 单个域名操作
		domain := authenticated.Group("/projects/:projectID/domains/:domainID")
		domain.Use(middleware.AuthMiddleware(userSvc))
		domain.Use(middleware.RequirePermission(memberSvc, "domain", "read"))
		{
			domain.GET("", domainH.Get)
			domain.DELETE("", middleware.RequirePermission(memberSvc, "domain", "delete"), domainH.Delete)
		}

		// 上游管理（需要 RBAC）
		upstreams := authenticated.Group("/projects/:projectID/proxy-configs/:proxyConfigID/upstreams")
		upstreams.Use(middleware.AuthMiddleware(userSvc))
		{
			upstreams.GET("", middleware.RequirePermission(memberSvc, "upstream", "read"), proxyH.ListUpstreams)
			upstreams.POST("", middleware.RequirePermission(memberSvc, "upstream", "create"), proxyH.AddUpstream)
			upstreams.DELETE("/:upstreamID", middleware.RequirePermission(memberSvc, "upstream", "delete"), proxyH.RemoveUpstream)
		}

		// 成员管理
		members := authenticated.Group("/projects/:projectID/members")
		members.Use(middleware.AuthMiddleware(userSvc))
		{
			members.GET("", middleware.RequirePermission(memberSvc, "member", "read"), memberH.List)
			members.POST("", middleware.RequirePermission(memberSvc, "member", "create"), memberH.Invite)
			members.DELETE("/:userID", middleware.RequirePermission(memberSvc, "member", "delete"), memberH.Remove)
		}

		// 域名共享
		shares := authenticated.Group("/projects/:projectID/domains/:domainID/share")
		shares.Use(middleware.AuthMiddleware(userSvc))
		{
			shares.POST("", middleware.RequirePermission(memberSvc, "share", "create"), shareH.Create)
			shares.DELETE("/:shareID", middleware.RequirePermission(memberSvc, "share", "delete"), shareH.Revoke)
		}

		// 审计日志
		auditLogs := authenticated.Group("/projects/:projectID/audit-logs")
		auditLogs.Use(middleware.AuthMiddleware(userSvc))
		{
			auditLogs.GET("", middleware.RequirePermission(memberSvc, "audit", "read"), auditH.List)
		}
	}
}