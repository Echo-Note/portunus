package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/internal/api/dto"
)

// RequestID 为每个请求生成唯一 X-Request-ID。
// 如果请求头中已包含 X-Request-ID 则沿用，否则生成新的 UUID。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// GetRequestID 从 gin.Context 中提取请求 ID。
func GetRequestID(c *gin.Context) string {
	id, _ := c.Get("request_id")
	if id == nil {
		return ""
	}
	return id.(string)
}

// Logging 结构化日志中间件，记录每个请求的 method、path、status、latency。
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"client_ip", c.ClientIP(),
			"request_id", GetRequestID(c),
		}
		if query != "" {
			attrs = append(attrs, "query", query)
		}

		if status >= 500 {
			slog.ErrorContext(c.Request.Context(), "请求处理失败", attrs...)
		} else if status >= 400 {
			slog.WarnContext(c.Request.Context(), "请求处理异常", attrs...)
		} else {
			slog.InfoContext(c.Request.Context(), "请求处理完成", attrs...)
		}
	}
}

// CORS 跨域中间件。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimit 全局速率限制中间件。
func RateLimit(rpm, burst int) gin.HandlerFunc {
	// 简单的令牌桶实现，生产环境应替换为 Redis 分布式限流
	tokens := make(chan struct{}, burst)
	for i := 0; i < burst; i++ {
		tokens <- struct{}{}
	}

	go func() {
		interval := time.Minute / time.Duration(rpm)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
			}
		}
	}()

	return func(c *gin.Context) {
		select {
		case <-tokens:
			c.Next()
		default:
			c.JSON(http.StatusTooManyRequests, dto.Error(dto.CodeRateLimited, "请求频率超限，请稍后重试"))
			c.Abort()
		}
	}
}

// ProjectContext 从 URL 参数中提取 project_id 并注入上下文。
// 此中间件必须在 AuthMiddleware 之后使用。
func ProjectContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("projectID")
		if projectID == "" {
			projectID = c.Param("id")
		}
		if projectID != "" {
			pid, err := uuid.Parse(projectID)
			if err == nil {
				c.Set("project_id", pid)
			}
		}
		c.Next()
	}
}
