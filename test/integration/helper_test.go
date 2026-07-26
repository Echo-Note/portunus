package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated/user"
	"github.com/Echo-Note/portunus/internal/service"
)

// ── HTTP 请求辅助函数 ──

// doRequest 发送 HTTP 请求并返回 ResponseRecorder。
func doRequest(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = bytes.NewBufferString(body)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, r)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// doAuthedRequest 发送带 Bearer token 的 HTTP 请求。
func doAuthedRequest(t *testing.T, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = bytes.NewBufferString(body)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, r)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// doRequestWithHeaders 发送带自定义 Header 的 HTTP 请求（可选 Bearer token）。
func doRequestWithHeaders(t *testing.T, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = bytes.NewBufferString(body)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, r)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// ── 响应解析辅助函数 ──

// parseEnvelope 解析统一响应外层 {code, message, data, request_id, timestamp}。
func parseEnvelope(t *testing.T, resp *httptest.ResponseRecorder) (code int, data map[string]any) {
	t.Helper()
	var envelope struct {
		Code      int            `json:"code"`
		Message   string         `json:"message"`
		Data      map[string]any `json:"data"`
		RequestID string         `json:"request_id"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &envelope)
	require.NoError(t, err, "failed to parse response envelope")
	return envelope.Code, envelope.Data
}

// assertEnvelopeRequestID 断言响应中包含 request_id 和 timestamp。
func assertEnvelopeRequestID(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()
	var envelope struct {
		RequestID string `json:"request_id"`
		Timestamp string `json:"timestamp"`
		Message   string `json:"message"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &envelope)
	require.NoError(t, err, "failed to parse response envelope")
	assert.NotEmpty(t, envelope.RequestID)
	assert.NotEmpty(t, envelope.Timestamp)
}

// ── 用户相关辅助函数 ──

// registerAndActivateUser 注册用户并通过 DB 直接激活。
// 返回 (email, password)。如果用户已存在则忽略注册错误。
func registerAndActivateUser(t *testing.T, email, password string) {
	t.Helper()
	ctx := context.Background()

	// 注册（忽略重复注册错误）
	regBody := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	_ = doRequest(t, "POST", "/api/v1/auth/register", regBody)

	// 通过 DB 直接激活用户（状态从 pending 改为 active）
	err := testClient.User.Update().
		Where(user.EmailEQ(email)).
		SetStatus("active").
		Exec(ctx)
	require.NoError(t, err, "failed to activate user: %s", email)
}

// loginUser 登录并返回 (accessToken, refreshToken)。
func loginUser(t *testing.T, email, password string) (accessToken, refreshToken string) {
	t.Helper()
	loginBody := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	resp := doRequest(t, "POST", "/api/v1/auth/login", loginBody)
	require.Equal(t, http.StatusOK, resp.Code, "login failed for %s", email)

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &loginResp))
	require.Equal(t, 0, loginResp.Code)
	require.NotEmpty(t, loginResp.Data.AccessToken)
	require.Equal(t, "Bearer", loginResp.Data.TokenType)

	return loginResp.Data.AccessToken, loginResp.Data.RefreshToken
}

// createUserAndLogin 注册 + 激活 + 登录，返回 (email, password, accessToken)。
func createUserAndLogin(t *testing.T, email, password string) (accessToken string) {
	t.Helper()
	registerAndActivateUser(t, email, password)
	accessToken, _ = loginUser(t, email, password)
	return accessToken
}

// ── 项目相关辅助函数 ──

// createProject 创建项目并返回项目 ID（UUID 字符串）。
func createProject(t *testing.T, token, projectID, name string) string {
	t.Helper()
	desc := fmt.Sprintf("Integration test project: %s", projectID)
	projBody := fmt.Sprintf(`{"project_id":"%s","name":"%s","description":"%s"}`, projectID, name, desc)
	resp := doAuthedRequest(t, "POST", "/api/v1/projects", projBody, token)
	require.Equal(t, http.StatusCreated, resp.Code, "failed to create project: %s", projectID)

	var projResp struct {
		Code int `json:"code"`
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &projResp))
	require.Equal(t, 0, projResp.Code)
	require.NotEmpty(t, projResp.Data.ID)

	return projResp.Data.ID
}

// ── 域名相关辅助函数 ──

// createDomain 创建域名并返回域名 ID（UUID 字符串）。
func createDomain(t *testing.T, token, projectID, domainName string) string {
	t.Helper()
	domainBody := fmt.Sprintf(`{"domain_name":"%s","ssl_enabled":true}`, domainName)
	path := fmt.Sprintf("/api/v1/projects/%s/domains", projectID)
	resp := doAuthedRequest(t, "POST", path, domainBody, token)
	require.Equal(t, http.StatusCreated, resp.Code, "failed to create domain: %s", domainName)

	var domainResp struct {
		Code int `json:"code"`
		Data struct {
			ID      string `json:"id"`
			CaddyID string `json:"caddy_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &domainResp))
	require.Equal(t, 0, domainResp.Code)
	require.NotEmpty(t, domainResp.Data.ID)

	return domainResp.Data.ID
}

// ensureProxyConfig 确保域名有代理配置（通过服务层直接创建，当前无 HTTP 端点）。
// 返回代理配置 ID。
func ensureProxyConfig(t *testing.T, domainID string) string {
	t.Helper()
	uid, err := uuid.Parse(domainID)
	require.NoError(t, err, "invalid domain ID: %s", domainID)

	ctx := context.Background()

	// 检查是否已存在
	existing, err := testProxySvc.GetProxyConfigByDomainID(ctx, uid)
	if err == nil {
		return existing.ID.String()
	}

	// 创建代理配置
	pc, err := testProxySvc.CreateProxyConfig(ctx, service.CreateProxyConfigInput{
		DomainID:            uid,
		LbPolicy:            "round_robin",
		HealthCheckInterval: "30s",
		Timeout:             "0s",
	})
	require.NoError(t, err, "failed to create proxy config for domain %s", domainID)

	return pc.ID.String()
}

// ── 成员相关辅助函数 ──

// inviteMember 邀请成员并返回邀请令牌。
func inviteMember(t *testing.T, token, projectID, email, role string) string {
	t.Helper()
	inviteBody := fmt.Sprintf(`{"email":"%s","role":"%s"}`, email, role)
	path := fmt.Sprintf("/api/v1/projects/%s/members", projectID)
	resp := doAuthedRequest(t, "POST", path, inviteBody, token)
	require.Equal(t, http.StatusCreated, resp.Code, "failed to invite member: %s as %s", email, role)

	var invResp struct {
		Code int `json:"code"`
		Data struct {
			InvitationToken string `json:"invitation_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &invResp))
	require.Equal(t, 0, invResp.Code)
	require.NotEmpty(t, invResp.Data.InvitationToken)

	return invResp.Data.InvitationToken
}

// acceptInvitation 接受邀请。
func acceptInvitation(t *testing.T, token, invitationToken string) {
	t.Helper()
	path := fmt.Sprintf("/api/v1/invitations/%s/accept", invitationToken)
	resp := doAuthedRequest(t, "POST", path, "", token)
	require.Equal(t, http.StatusOK, resp.Code, "failed to accept invitation: %s", invitationToken)
}

// ── 唯一标识生成 ──

// uniqueCounter 全局递增计数器，用于生成唯一标识。
var uniqueCounter int64

// uniqueSuffix 返回基于计数器和时间戳的唯一短后缀，用于生成唯一的邮箱和项目标识。
func uniqueSuffix(t *testing.T) string {
	t.Helper()
	seq := atomic.AddInt64(&uniqueCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, seq)
}