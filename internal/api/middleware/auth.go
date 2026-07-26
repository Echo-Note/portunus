// Package middleware 提供 HTTP 中间件链。
// 中间件顺序：Auth → RBAC → Ownership → RateLimit → Audit
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nrgao/portunus/internal/service"
)

// AuthMiddleware 从 Authorization 头提取并验证 JWT access_token。
// 验证通过后将 userID 注入 gin.Context。
func AuthMiddleware(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 Authorization 头"})
			return
		}

		// 提取 Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization 格式错误，应为 Bearer <token>"})
			return
		}

		token := parts[1]

		// 验证 token
		userID, err := userSvc.VerifyToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "access_token 无效或已过期"})
			return
		}

		// 将 userID 存入上下文
		c.Set("user_id", userID)
		c.Next()
	}
}

// OptionalAuthMiddleware 可选的认证中间件。
// 如果提供了有效的 token 则注入 userID，否则继续处理（用于公开接口）。
func OptionalAuthMiddleware(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		token := parts[1]
		userID, err := userSvc.VerifyToken(c.Request.Context(), token)
		if err == nil {
			c.Set("user_id", userID)
		}
		c.Next()
	}
}

// GetUserID 从 gin.Context 中提取已验证的用户 ID。
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := val.(uuid.UUID)
	return userID, ok
}

// GetProjectID 从 URL 参数中提取项目 ID。
func GetProjectID(c *gin.Context) (uuid.UUID, error) {
	return uuid.Parse(c.Param("projectID"))
}