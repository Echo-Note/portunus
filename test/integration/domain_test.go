package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/internal/api/dto"
)

// ── 域名 CRUD 测试 ──

// TestDomain_CreateListGetDelete 测试域名完整生命周期。
func TestDomain_CreateListGetDelete(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("domain-crud-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-domain-crud-%s", suffix)
	domainName := fmt.Sprintf("api-crud-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Domain CRUD Test")

	// 1. 创建域名
	domainID := createDomain(t, token, projectID, domainName)

	// 2. 列出域名
	path := fmt.Sprintf("/api/v1/projects/%s/domains", projectID)
	resp := doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	items, ok := data["items"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(items), 1)

	// 3. 获取域名详情
	path = fmt.Sprintf("/api/v1/projects/%s/domains/%s", projectID, domainID)
	resp = doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data = parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, domainName, data["domain_name"])
	assert.Contains(t, data["caddy_id"], "tenant_")

	// 4. 删除域名
	path = fmt.Sprintf("/api/v1/projects/%s/domains/%s", projectID, domainID)
	resp = doAuthedRequest(t, "DELETE", path, "", token)
	assert.Equal(t, http.StatusNoContent, resp.Code)

	// 5. 删除后域名仍可通过 ID 查询（软删除，状态变为 deleted）
	// 但列表不应包含已删除的域名
	path = fmt.Sprintf("/api/v1/projects/%s/domains", projectID)
	resp = doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data = parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	items, _ = data["items"].([]any)
	// 已删除的域名不应出现在列表中
	for _, item := range items {
		domain := item.(map[string]any)
		assert.NotEqual(t, domainID, domain["id"], "deleted domain should not appear in list")
	}
}

// TestDomain_GlobalUniqueness 测试域名全局唯一性（不同项目不能配相同域名）。
func TestDomain_GlobalUniqueness(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("domain-unique-%s@example.com", suffix)
	password := "test123456"
	projectSlugA := fmt.Sprintf("test-domain-uniq-a-%s", suffix)
	projectSlugB := fmt.Sprintf("test-domain-uniq-b-%s", suffix)
	domainName := fmt.Sprintf("global-unique-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)

	projectIDA := createProject(t, token, projectSlugA, "Unique Test A")
	projectIDB := createProject(t, token, projectSlugB, "Unique Test B")

	// 在项目 A 中创建域名
	createDomain(t, token, projectIDA, domainName)

	// 在项目 B 中尝试创建相同域名
	domainBody := fmt.Sprintf(`{"domain_name":"%s","ssl_enabled":true}`, domainName)
	path := fmt.Sprintf("/api/v1/projects/%s/domains", projectIDB)
	resp := doAuthedRequest(t, "POST", path, domainBody, token)
	assert.Equal(t, http.StatusConflict, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeConflict, code)
}

// TestDomain_Update 测试更新域名名称。
func TestDomain_Update(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("domain-update-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-domain-update-%s", suffix)
	domainName := fmt.Sprintf("domain-update-%s.example.com", suffix)
	newDomainName := fmt.Sprintf("domain-update-new-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Domain Update Test")
	domainID := createDomain(t, token, projectID, domainName)

	// 更新域名
	updateBody := fmt.Sprintf(`{"domain_name":"%s"}`, newDomainName)
	path := fmt.Sprintf("/api/v1/projects/%s/domains/%s", projectID, domainID)
	resp := doAuthedRequest(t, "PATCH", path, updateBody, token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, newDomainName, data["domain_name"])
}

// TestDomain_InvalidDomainID 测试无效的域名 ID 返回 400。
func TestDomain_InvalidDomainID(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("domain-invalid-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-domain-invalid-%s", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Invalid Domain ID Test")

	// 使用非 UUID 的 domainID
	path := fmt.Sprintf("/api/v1/projects/%s/domains/not-a-uuid", projectID)
	resp := doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

// ── 上游管理测试 ──

// TestDomain_AddUpstream 测试添加和列出上游。
func TestDomain_AddUpstream(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("upstream-add-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-upstream-%s", suffix)
	domainName := fmt.Sprintf("upstream-add-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Upstream Add Test")
	domainID := createDomain(t, token, projectID, domainName)

	// 确保代理配置存在（当前无 HTTP 创建端点，通过服务层创建）
	ensureProxyConfig(t, domainID)

	// 添加上游
	upstreamBody := `{"dial_address":"127.0.0.1:8080","weight":1}`
	path := fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy/upstreams", projectID, domainID)
	resp := doAuthedRequest(t, "POST", path, upstreamBody, token)
	assert.Equal(t, http.StatusCreated, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, "127.0.0.1:8080", data["dial_address"])

	// 列出上游
	path = fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy/upstreams", projectID, domainID)
	resp = doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data = parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	items, ok := data["items"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(items), 1)

	// 验证上游内容
	upstream := items[0].(map[string]any)
	assert.Equal(t, "127.0.0.1:8080", upstream["dial_address"])
}

// TestDomain_RemoveUpstream 测试移除上游。
func TestDomain_RemoveUpstream(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("upstream-remove-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-upstream-rm-%s", suffix)
	domainName := fmt.Sprintf("upstream-remove-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Upstream Remove Test")
	domainID := createDomain(t, token, projectID, domainName)

	ensureProxyConfig(t, domainID)

	// 添加上游
	upstreamBody := `{"dial_address":"127.0.0.1:9090","weight":1}`
	path := fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy/upstreams", projectID, domainID)
	resp := doAuthedRequest(t, "POST", path, upstreamBody, token)
	assert.Equal(t, http.StatusCreated, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	upstreamID := data["id"].(string)

	// 移除上游
	path = fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy/upstreams/%s", projectID, domainID, upstreamID)
	resp = doAuthedRequest(t, "DELETE", path, "", token)
	assert.Equal(t, http.StatusNoContent, resp.Code)

	// 验证上游已移除
	path = fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy/upstreams", projectID, domainID)
	resp = doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data = parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	items := data["items"].([]any)
	assert.Equal(t, 0, len(items))
}

// TestDomain_DuplicateUpstream 测试重复添加上游返回 409。
func TestDomain_DuplicateUpstream(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("upstream-dup-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-upstream-dup-%s", suffix)
	domainName := fmt.Sprintf("upstream-dup-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Upstream Dup Test")
	domainID := createDomain(t, token, projectID, domainName)

	ensureProxyConfig(t, domainID)

	// 添加上游
	upstreamBody := `{"dial_address":"127.0.0.1:8081","weight":1}`
	path := fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy/upstreams", projectID, domainID)

	resp := doAuthedRequest(t, "POST", path, upstreamBody, token)
	assert.Equal(t, http.StatusCreated, resp.Code)

	// 重复添加相同 dial_address
	resp = doAuthedRequest(t, "POST", path, upstreamBody, token)
	assert.Equal(t, http.StatusConflict, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeConflict, code)
}

// TestUpstream_Status 测试获取上游健康状态。
func TestUpstream_Status(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("upstream-status-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-upstream-status-%s", suffix)
	domainName := fmt.Sprintf("upstream-status-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Upstream Status Test")
	domainID := createDomain(t, token, projectID, domainName)

	ensureProxyConfig(t, domainID)

	// 添加一个上游
	upstreamBody := `{"dial_address":"127.0.0.1:8082","weight":1}`
	path := fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy/upstreams", projectID, domainID)
	resp := doAuthedRequest(t, "POST", path, upstreamBody, token)
	assert.Equal(t, http.StatusCreated, resp.Code)

	// 获取上游状态
	path = fmt.Sprintf("/api/v1/projects/%s/domains/%s/status", projectID, domainID)
	resp = doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	items, ok := data["items"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(items), 1)
}

// ── 代理配置测试 ──

// TestProxy_GetConfig 测试获取代理配置。
func TestProxy_GetConfig(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("proxy-get-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-proxy-get-%s", suffix)
	domainName := fmt.Sprintf("proxy-get-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Proxy Get Test")
	domainID := createDomain(t, token, projectID, domainName)

	ensureProxyConfig(t, domainID)

	// 获取代理配置
	path := fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy", projectID, domainID)
	resp := doAuthedRequest(t, "GET", path, "", token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "round_robin", data["lb_policy"])
}

// TestProxy_UpdateConfig 测试更新代理配置。
func TestProxy_UpdateConfig(t *testing.T) {
	suffix := uniqueSuffix(t)
	email := fmt.Sprintf("proxy-update-%s@example.com", suffix)
	password := "test123456"
	projectSlug := fmt.Sprintf("test-proxy-update-%s", suffix)
	domainName := fmt.Sprintf("proxy-update-%s.example.com", suffix)

	token := createUserAndLogin(t, email, password)
	projectID := createProject(t, token, projectSlug, "Proxy Update Test")
	domainID := createDomain(t, token, projectID, domainName)

	ensureProxyConfig(t, domainID)

	// 更新代理配置
	updateBody := `{"lb_policy":"least_conn","health_check_uri":"/health"}`
	path := fmt.Sprintf("/api/v1/projects/%s/domains/%s/proxy", projectID, domainID)
	resp := doAuthedRequest(t, "PATCH", path, updateBody, token)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, "least_conn", data["lb_policy"])
	assert.Equal(t, "/health", data["health_check_uri"])
}