package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/internal/service"
)

// RBAC 统一权限矩阵，定义各角色对各资源的操作权限。
var rolePermissions = map[string]map[string][]string{
	"owner": {
		"domain":   {"create", "read", "update", "delete"},
		"proxy":    {"create", "read", "update", "delete"},
		"upstream": {"create", "read", "update", "delete"},
		"member":   {"create", "read", "update", "delete"},
		"project":  {"create", "read", "update", "delete"},
		"share":    {"create", "read", "update", "delete"},
		"audit":    {"read"},
	},
	"admin": {
		"domain":   {"create", "read", "update", "delete"},
		"proxy":    {"create", "read", "update", "delete"},
		"upstream": {"create", "read", "update", "delete"},
		"member":   {"create", "read", "update", "delete"},
		"project":  {"read"},
		"share":    {"create", "read", "update", "delete"},
		"audit":    {"read"},
	},
	"editor": {
		"domain":   {"create", "read", "update"},
		"proxy":    {"read", "update"},
		"upstream": {"create", "read"},
		"member":   {},
		"project":  {"read"},
		"share":    {},
		"audit":    {},
	},
	"viewer": {
		"domain":   {"read"},
		"proxy":    {"read"},
		"upstream": {"read"},
		"member":   {},
		"project":  {"read"},
		"share":    {},
		"audit":    {},
	},
}

// RequirePermission 返回一个中间件，检查当前用户是否具有指定资源的操作权限。
// resourceType: domain / proxy / upstream / member / project / share / audit
// action: create / read / update / delete
func RequirePermission(memberSvc *service.MemberService, resourceType, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			return
		}

		projectID, err := GetProjectID(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
			return
		}

		// 获取用户角色
		role, err := memberSvc.GetMemberRole(c.Request.Context(), projectID, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权访问该项目"})
			return
		}

		// 检查权限
		perms, ok := rolePermissions[role]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "未知角色"})
			return
		}

		allowedActions, ok := perms[resourceType]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权操作该资源"})
			return
		}

		allowed := false
		for _, a := range allowedActions {
			if a == action {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			return
		}

		// 将角色和项目 ID 存入上下文供后续使用
		c.Set("role", role)
		c.Set("project_id", projectID)
		c.Next()
	}
}

// GetRole 从 gin.Context 中提取用户角色。
func GetRole(c *gin.Context) string {
	role, _ := c.Get("role")
	if role == nil {
		return ""
	}
	return role.(string)
}

// GetProjectIDFromCtx 从 gin.Context 中提取项目 ID。
func GetProjectIDFromCtx(c *gin.Context) uuid.UUID {
	pid, _ := c.Get("project_id")
	if pid == nil {
		return uuid.Nil
	}
	return pid.(uuid.UUID)
}