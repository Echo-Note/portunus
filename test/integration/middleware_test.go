package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Echo-Note/portunus/internal/api/dto"
	"github.com/Echo-Note/portunus/internal/api/middleware"
)

// ── 认证中间件集成测试 ──

// TestMiddleware_NoAuthHeader 测试无 Authorization 头返回 401。
func TestMiddleware_NoAuthHeader(t *testing.T) {
	resp := doRequest(t, "GET", "/api/v1/me", "")
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
}

// TestMiddleware_MalformedAuthHeader 测试格式错误的 Authorization 头返回 401。
func TestMiddleware_MalformedAuthHeader(t *testing.T) {
	// 缺少 "Bearer" 前缀
	headers := map[string]string{"Authorization": "not-bearer some-token"}
	resp := doRequestWithHeaders(t, "GET", "/api/v1/me", "", headers)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
}

// TestMiddleware_MalformedAuthHeader_BasicFormat 测试 Basic 格式的 Authorization 头。
func TestMiddleware_MalformedAuthHeader_BasicFormat(t *testing.T) {
	headers := map[string]string{"Authorization": "Basic dXNlcjpwYXNz"}
	resp := doRequestWithHeaders(t, "GET", "/api/v1/me", "", headers)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

// TestMiddleware_InvalidToken 测试无效 token 返回 401。
func TestMiddleware_InvalidToken(t *testing.T) {
	resp := doAuthedRequest(t, "GET", "/api/v1/me", "", "this-is-not-a-valid-jwt-token")
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
}

// TestMiddleware_NoBearerPrefix 测试 token 无 Bearer 前缀。
func TestMiddleware_NoBearerPrefix(t *testing.T) {
	headers := map[string]string{"Authorization": "sometokenwithoutbearer"}
	resp := doRequestWithHeaders(t, "GET", "/api/v1/me", "", headers)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

// ── CORS 中间件测试 ──

// TestMiddleware_CORS_Preflight 测试 OPTIONS 预检请求返回 204 并包含 CORS 头。
func TestMiddleware_CORS_Preflight(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), "OPTIONS", "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// 验证 CORS 响应头
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

// TestMiddleware_CORS_Headers 测试正常请求包含 CORS 头。
func TestMiddleware_CORS_Headers(t *testing.T) {
	resp := doRequest(t, "GET", "/health", "")
	assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
}

// ── RequestID 中间件测试 ──

// TestMiddleware_RequestID_Generated 测试自动生成 RequestID。
func TestMiddleware_RequestID_Generated(t *testing.T) {
	resp := doRequest(t, "GET", "/health", "")
	assert.NotEmpty(t, resp.Header().Get("X-Request-ID"))
}

// TestMiddleware_RequestID_Propagated 测试传递已有的 RequestID。
func TestMiddleware_RequestID_Propagated(t *testing.T) {
	headers := map[string]string{"X-Request-ID": "my-custom-request-id"}
	resp := doRequestWithHeaders(t, "GET", "/health", "", headers)
	assert.Equal(t, "my-custom-request-id", resp.Header().Get("X-Request-ID"))

	// 验证响应体外层 envelope 中包含 request_id
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	// request_id 在外层 envelope 中，不在 data 中 — 验证 header 即可
}

// ── 限流中间件测试 ──

// TestMiddleware_RateLimit 测试限流中间件在超出 burst 时返回 429。
// 注意：全局 router 的限流配置为 (6000, 1000)，不会在日常测试中触发。
// 因此创建一个独立的低限流 router 来验证中间件行为。
func TestMiddleware_RateLimit(t *testing.T) {
	// 创建独立的低限流 router
	router := gin.New()
	router.Use(middleware.RateLimit(1, 1)) // 1 RPM, burst 1
	router.GET("/test-rate-limit", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 第一个请求应通过
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test-rate-limit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 第二个请求应立即触发限流（burst 已耗尽）
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// ── RBAC 中间件集成测试 ──

// TestMiddleware_RBAC_NonMemberForbidden 测试非成员访问项目资源返回 403。
func TestMiddleware_RBAC_NonMemberForbidden(t *testing.T) {
	suffix := uniqueSuffix(t)
	emailA := fmt.Sprintf("rbac-nonmember-a-%s@example.com", suffix)
	emailB := fmt.Sprintf("rbac-nonmember-b-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-rbac-nonmem-%s", suffix)

	tokenA := createUserAndLogin(t, emailA, password)
	tokenB := createUserAndLogin(t, emailB, password)

	projectID := createProject(t, tokenA, projectSlug, "RBAC Non-Member Test")

	// 用户 B 尝试访问项目 A 的域名
	path := fmt.Sprintf("/api/v1/projects/%s/domains", projectID)
	resp := doAuthedRequest(t, "GET", path, "", tokenB)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

// TestMiddleware_RBAC_InvalidProjectID 测试无效项目 ID 在 RBAC 中间件中返回 400。
func TestMiddleware_RBAC_InvalidProjectID(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("rbac-invalidid-%s@example.com", suffix)
	password := "test123456"

	token := createUserAndLogin(t, email, password)

	// 使用非 UUID 的 projectID 访问需要 RBAC 的路由
	path := "/api/v1/projects/not-a-uuid/domains"
	resp := doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

// TestMiddleware_AuthWorks 测试有效的 token 能通过认证中间件。
func TestMiddleware_AuthWorks(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("rbac-authworks-%s@example.com", suffix)
	password := "test123456"

	token := createUserAndLogin(t, email, password)

	// 使用有效 token 访问 /me
	resp := doAuthedRequest(t, "GET", "/api/v1/me", "", token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, email, data["email"])
}

// ── 响应格式测试 ──

// TestResponse_Envelope 测试所有响应都使用统一信封格式。
func TestResponse_Envelope(t *testing.T) {
	// 成功响应
	resp := doRequest(t, "GET", "/health", "")
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, "ok", data["status"])
	assertEnvelopeRequestID(t, resp)

	// 错误响应
	resp = doRequest(t, "GET", "/api/v1/me", "")
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ = parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
	assertEnvelopeRequestID(t, resp)
}