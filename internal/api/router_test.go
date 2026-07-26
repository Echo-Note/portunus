package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/user"
	"github.com/Echo-Note/portunus/internal/api"
	"github.com/Echo-Note/portunus/internal/api/dto"
	"github.com/Echo-Note/portunus/internal/api/handler"
	"github.com/Echo-Note/portunus/internal/config"
	"github.com/Echo-Note/portunus/internal/service"
)

// APITestSuite 是 API 集成测试套件。
// 使用真实的数据库和完整的中间件链路。
type APITestSuite struct {
	suite.Suite
	router     *gin.Engine
	userSvc    *service.UserService
	memberSvc  *service.MemberService
	client     *generated.Client
	accessToken string
	projectID   string
	domainID    string
}

// SetupSuite 在所有测试开始前执行一次。
func (s *APITestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// 连接数据库
	ctx := s.ctx()
	cfg := config.DatabaseConfig{
		URL:             getEnvOrDefault("DATABASE_URL", "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable"),
		MaxOpenConns:    5,
		MaxIdleConns:    2,
	}
	client, err := config.NewEntClient(ctx, cfg)
	require.NoError(s.T(), err, "数据库连接失败，请确保 Docker 服务已启动")
	s.client = client

	// 清理测试数据（按外键依赖顺序删除）
	client.CaddyIDMapping.Delete().Exec(ctx) //nolint:errcheck
	client.Upstream.Delete().Exec(ctx) //nolint:errcheck
	client.ProxyConfig.Delete().Exec(ctx) //nolint:errcheck
	client.DomainShare.Delete().Exec(ctx) //nolint:errcheck
	client.Domain.Delete().Exec(ctx) //nolint:errcheck
	client.ProjectAuditLog.Delete().Exec(ctx) //nolint:errcheck
	client.Invitation.Delete().Exec(ctx) //nolint:errcheck
	client.ProjectMember.Delete().Exec(ctx) //nolint:errcheck
	client.Project.Delete().Exec(ctx) //nolint:errcheck
	client.ApiToken.Delete().Exec(ctx) //nolint:errcheck
	client.ConfigSnapshot.Delete().Exec(ctx) //nolint:errcheck
	client.User.Delete().Exec(ctx) //nolint:errcheck

	// 初始化 JWT 配置
	jwtCfg := config.JWTConfig{
		AccessTokenTTL:  config.JWTConfig{}.AccessTokenTTL,
		RefreshTokenTTL: config.JWTConfig{}.RefreshTokenTTL,
	}
	if jwtCfg.AccessTokenTTL == 0 {
		jwtCfg.AccessTokenTTL = 15 * 60 * 1000000000
	}
	if jwtCfg.RefreshTokenTTL == 0 {
		jwtCfg.RefreshTokenTTL = 168 * 60 * 60 * 1000000000
	}

	// 从文件读取 JWT 密钥
	privateKeyFile := os.Getenv("JWT_PRIVATE_KEY_FILE")
	if privateKeyFile == "" {
		privateKeyFile = "certs/jwt-private.pem"
	}
	publicKeyFile := os.Getenv("JWT_PUBLIC_KEY_FILE")
	if publicKeyFile == "" {
		publicKeyFile = "certs/jwt-public.pem"
	}

	if data, err := os.ReadFile(privateKeyFile); err == nil {
		jwtCfg.PrivateKey = string(data)
	}
	if data, err := os.ReadFile(publicKeyFile); err == nil {
		jwtCfg.PublicKey = string(data)
	}

	s.userSvc, err = service.NewUserService(client, jwtCfg)
	require.NoError(s.T(), err)

	// 初始化 Service 层
	stateMachine := service.NewStateMachine(client)
	caddyClient := service.NewNoopCaddyClient()
	projectSvc := service.NewProjectService(client, stateMachine)
	domainSvc := service.NewDomainService(client, stateMachine, caddyClient)
	proxySvc := service.NewProxyService(client, stateMachine, caddyClient)
	s.memberSvc = service.NewMemberService(client, stateMachine)
	shareSvc := service.NewShareService(client, stateMachine)
	auditSvc := service.NewAuditService(client)

	// 初始化 Handler
	authH := handler.NewAuthHandler(s.userSvc)
	projectH := handler.NewProjectHandler(projectSvc)
	domainH := handler.NewDomainHandler(domainSvc)
	proxyH := handler.NewProxyHandler(proxySvc)
	memberH := handler.NewMemberHandler(s.memberSvc)
	shareH := handler.NewShareHandler(shareSvc)
	auditH := handler.NewAuditHandler(auditSvc)

	// 创建路由
	s.router = gin.New()
	api.RegisterRoutes(
		s.router,
		authH, projectH, domainH, proxyH, memberH, shareH, auditH,
		s.userSvc, s.memberSvc,
		6000, 1000, // 高限流值用于测试
	)
}

// TearDownSuite 在所有测试结束后执行一次。
func (s *APITestSuite) TearDownSuite() {
	if s.client != nil {
		s.client.Close()
	}
}

// ctx 返回基础 context。
func (s *APITestSuite) ctx() context.Context {
	return context.Background()
}

// ── 测试用例 ──

// TestAuth_Register_Login 测试注册→激活→登录→获取信息→刷新令牌→退出登录的完整链路。
func (s *APITestSuite) TestAuth_Register_Login() {
	t := s.T()

	// 1. 注册
	regBody := `{"email":"api-test@example.com","password":"test123456"}`
	resp := s.doRequest("POST", "/api/v1/auth/register", regBody)
	assert.Equal(t, http.StatusCreated, resp.Code)
	var regResp struct {
		Code int `json:"code"`
		Data struct {
			UserID string `json:"user_id"`
			Email  string `json:"email"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &regResp)
	assert.Equal(t, 0, regResp.Code)
	assert.Equal(t, "api-test@example.com", regResp.Data.Email)

	// 2. 未激活用户登录应失败
	loginBody := `{"email":"api-test@example.com","password":"test123456"}`
	resp = s.doRequest("POST", "/api/v1/auth/login", loginBody)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	// 3. 手动激活用户
	ctx := s.ctx()
	users, err := s.client.User.Query().All(ctx)
	require.NoError(t, err)
	if len(users) > 0 {
		users[0].Update().SetStatus("active").Save(ctx) //nolint:errcheck
	}

	// 4. 登录
	resp = s.doRequest("POST", "/api/v1/auth/login", loginBody)
	assert.Equal(t, http.StatusOK, resp.Code)
	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &loginResp)
	assert.Equal(t, 0, loginResp.Code)
	assert.NotEmpty(t, loginResp.Data.AccessToken)
	assert.Equal(t, "Bearer", loginResp.Data.TokenType)

	s.accessToken = loginResp.Data.AccessToken

	// 5. 获取用户信息
	resp = s.doAuthedRequest("GET", "/api/v1/me", "")
	assert.Equal(t, http.StatusOK, resp.Code)
	var meResp struct {
		Code int `json:"code"`
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &meResp)
	assert.Equal(t, "api-test@example.com", meResp.Data.Email)

	// 6. 刷新令牌
	refreshBody := fmt.Sprintf(`{"refresh_token":"%s"}`, loginResp.Data.RefreshToken)
	resp = s.doRequest("POST", "/api/v1/auth/refresh", refreshBody)
	assert.Equal(t, http.StatusOK, resp.Code)

	// 7. 退出登录
	resp = s.doAuthedRequest("POST", "/api/v1/auth/logout", "")
	assert.Equal(t, http.StatusOK, resp.Code)
}

// TestProject_CRUD 测试项目创建、查询、列表。
func (s *APITestSuite) TestProject_CRUD() {
	t := s.T()

	// 先确保登录
	s.ensureLoggedIn(t)

	// 1. 创建项目
	projBody := `{"project_id":"test-proj-api","name":"API测试项目","description":"集成测试"}`
	resp := s.doAuthedRequest("POST", "/api/v1/projects", projBody)
	assert.Equal(t, http.StatusCreated, resp.Code)
	var projResp struct {
		Code int `json:"code"`
		Data struct {
			ID        string `json:"id"`
			ProjectID string `json:"project_id"`
			Name      string `json:"name"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &projResp)
	assert.Equal(t, 0, projResp.Code)
	assert.Equal(t, "test-proj-api", projResp.Data.ProjectID)
	assert.Equal(t, "active", projResp.Data.Status)
	s.projectID = projResp.Data.ID

	// 2. 获取项目详情
	resp = s.doAuthedRequest("GET", "/api/v1/projects/"+s.projectID, "")
	assert.Equal(t, http.StatusOK, resp.Code)

	// 3. 列出项目
	resp = s.doAuthedRequest("GET", "/api/v1/projects", "")
	assert.Equal(t, http.StatusOK, resp.Code)
	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &listResp)
	assert.GreaterOrEqual(t, listResp.Data.Total, 1)

	// 4. 重复创建项目 ID 应失败
	resp = s.doAuthedRequest("POST", "/api/v1/projects", projBody)
	assert.Equal(t, http.StatusConflict, resp.Code)
}

// TestDomain_CRUD 测试域名创建、列表、删除。
func (s *APITestSuite) TestDomain_CRUD() {
	t := s.T()
	s.ensureLoggedIn(t)
	s.ensureProject(t)

	// 1. 创建域名
	domainBody := `{"domain_name":"api-test.example.com","ssl_enabled":true}`
	resp := s.doAuthedRequest("POST", "/api/v1/projects/"+s.projectID+"/domains", domainBody)
	assert.Equal(t, http.StatusCreated, resp.Code)
	var domainResp struct {
		Code int `json:"code"`
		Data struct {
			ID      string `json:"id"`
			CaddyID string `json:"caddy_id"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &domainResp)
	assert.Equal(t, 0, domainResp.Code)
	assert.Contains(t, domainResp.Data.CaddyID, "tenant_")
	s.domainID = domainResp.Data.ID

	// 2. 列出域名
	resp = s.doAuthedRequest("GET", "/api/v1/projects/"+s.projectID+"/domains", "")
	assert.Equal(t, http.StatusOK, resp.Code)

	// 3. 获取域名详情
	resp = s.doAuthedRequest("GET", "/api/v1/projects/"+s.projectID+"/domains/"+s.domainID, "")
	assert.Equal(t, http.StatusOK, resp.Code)

	// 4. 删除域名
	resp = s.doAuthedRequest("DELETE", "/api/v1/projects/"+s.projectID+"/domains/"+s.domainID, "")
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

// TestMember_Invite 测试邀请成员和列表。
func (s *APITestSuite) TestMember_Invite() {
	t := s.T()
	s.ensureLoggedIn(t)
	s.ensureProject(t)

	// 1. 邀请成员
	inviteBody := `{"email":"invited@example.com","role":"editor"}`
	resp := s.doAuthedRequest("POST", "/api/v1/projects/"+s.projectID+"/members", inviteBody)
	assert.Equal(t, http.StatusCreated, resp.Code)

	// 2. 列出成员
	resp = s.doAuthedRequest("GET", "/api/v1/projects/"+s.projectID+"/members", "")
	assert.Equal(t, http.StatusOK, resp.Code)
	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Total int `json:"total"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &listResp)
	assert.GreaterOrEqual(t, listResp.Data.Total, 1) // owner + invited
}

// TestUnauthorized 测试未认证请求。
func (s *APITestSuite) TestUnauthorized() {
	t := s.T()

	resp := s.doRequest("GET", "/api/v1/me", "")
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	var errResp struct {
		Code int `json:"code"`
	}
	json.Unmarshal(resp.Body.Bytes(), &errResp)
	assert.Equal(t, dto.CodeUnauthorized, errResp.Code)
}

// TestHealthCheck 测试健康检查端点。
func (s *APITestSuite) TestHealthCheck() {
	t := s.T()

	resp := s.doRequest("GET", "/health", "")
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = s.doRequest("GET", "/ready", "")
	assert.Equal(t, http.StatusOK, resp.Code)
}

// TestDuplicateRegistration 测试重复注册。
func (s *APITestSuite) TestDuplicateRegistration() {
	t := s.T()

	body := `{"email":"dup-api-test@example.com","password":"test123456"}`

	// 首次注册
	resp := s.doRequest("POST", "/api/v1/auth/register", body)
	assert.Equal(t, http.StatusCreated, resp.Code)

	// 重复注册
	resp = s.doRequest("POST", "/api/v1/auth/register", body)
	assert.Equal(t, http.StatusConflict, resp.Code)
}

// ── 辅助方法 ──

// doRequest 发送 HTTP 请求。
func (s *APITestSuite) doRequest(method, path, body string) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

// doAuthedRequest 发送带认证的 HTTP 请求。
func (s *APITestSuite) doAuthedRequest(method, path, body string) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	if s.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.accessToken)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

// ensureLoggedIn 确保有有效的 access token。
func (s *APITestSuite) ensureLoggedIn(t *testing.T) {
	if s.accessToken != "" {
		return
	}

	email := "api-test@example.com"

	// 尝试注册（忽略已存在的用户）
	_ = s.doRequest("POST", "/api/v1/auth/register", `{"email":"`+email+`","password":"test123456"}`)

	// 激活特定用户（通过邮箱查询）
	ctx := s.ctx()
	_ = s.client.User.Update().Where(user.EmailEQ(email)).SetStatus("active").Exec(ctx) //nolint:errcheck

	// 登录
	resp := s.doRequest("POST", "/api/v1/auth/login", `{"email":"`+email+`","password":"test123456"}`)
	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &loginResp)
	s.accessToken = loginResp.Data.AccessToken
}

// ensureProject 确保有项目 ID。
func (s *APITestSuite) ensureProject(t *testing.T) {
	if s.projectID != "" {
		return
	}
	resp := s.doAuthedRequest("POST", "/api/v1/projects", `{"project_id":"api-test-proj","name":"API测试项目"}`)
	var projResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &projResp)
	s.projectID = projResp.Data.ID
}

// getEnvOrDefault 获取环境变量或默认值。
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// TestAPISuite 运行测试套件。
func TestAPISuite(t *testing.T) {
	// 仅在数据库可用时运行
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("跳过集成测试（SKIP_INTEGRATION 已设置）")
	}
	suite.Run(t, new(APITestSuite))
}