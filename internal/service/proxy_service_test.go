package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated/proxyconfig"
	"github.com/Echo-Note/portunus/ent/generated/upstream"
	"github.com/Echo-Note/portunus/internal/config"
)

// setupProxyService 创建测试用 ProxyService 实例。
func setupProxyService(t *testing.T) (*ProxyService, *DomainService, *ProjectService, *UserService, context.Context) {
	t.Helper()
	ctx := context.Background()

	cfg := config.DatabaseConfig{
		URL:             "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
	}
	client, err := config.NewEntClient(ctx, cfg)
	require.NoError(t, err)

	// 清理所有表
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

	jwtCfg := config.JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	}
	privateKeyFile := os.Getenv("JWT_PRIVATE_KEY_FILE")
	if privateKeyFile == "" {
		privateKeyFile = "../../certs/jwt-private.pem"
	}
	publicKeyFile := os.Getenv("JWT_PUBLIC_KEY_FILE")
	if publicKeyFile == "" {
		publicKeyFile = "../../certs/jwt-public.pem"
	}
	if data, err := os.ReadFile(privateKeyFile); err == nil {
		jwtCfg.PrivateKey = string(data)
	}
	if data, err := os.ReadFile(publicKeyFile); err == nil {
		jwtCfg.PublicKey = string(data)
	}

	userSvc, err := NewUserService(client, jwtCfg)
	require.NoError(t, err)

	sm := NewStateMachine(client)
	projectSvc := NewProjectService(client, sm)
	caddyClient := NewNoopCaddyClient()
	domainSvc := NewDomainService(client, sm, caddyClient)
	proxySvc := NewProxyService(client, sm, caddyClient)

	t.Cleanup(func() { client.Close() })
	return proxySvc, domainSvc, projectSvc, userSvc, ctx
}

// TestProxyService_CreateProxyConfig 测试创建代理配置。
func TestProxyService_CreateProxyConfig(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "proxy-test", "proxy-owner@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "proxy.example.com",
	})
	require.NoError(t, err)

	pc, err := svc.CreateProxyConfig(ctx, CreateProxyConfigInput{
		DomainID: d.ID,
		LbPolicy: "round_robin",
	})
	require.NoError(t, err)
	assert.NotNil(t, pc)
	assert.Contains(t, pc.CaddyProxyID, "tenant_")
}

// TestProxyService_CreateProxyConfig_Duplicate 测试重复创建代理配置。
func TestProxyService_CreateProxyConfig_Duplicate(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "proxy-dup", "proxy-dup@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "proxy-dup.example.com",
	})
	require.NoError(t, err)

	_, err = svc.CreateProxyConfig(ctx, CreateProxyConfigInput{DomainID: d.ID})
	require.NoError(t, err)

	_, err = svc.CreateProxyConfig(ctx, CreateProxyConfigInput{DomainID: d.ID})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicate)
}

// TestProxyService_AddUpstream 测试添加上游。
func TestProxyService_AddUpstream(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "upstream-test", "upstream-owner@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "upstream.example.com",
	})
	require.NoError(t, err)

	pc, err := svc.CreateProxyConfig(ctx, CreateProxyConfigInput{DomainID: d.ID})
	require.NoError(t, err)

	u, err := svc.AddUpstream(ctx, AddUpstreamInput{
		ProxyConfigID: pc.ID,
		DialAddress:   "10.0.0.1:8080",
		Weight:        2,
	})
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.1:8080", u.DialAddress)
	assert.Equal(t, 2, u.Weight)
	assert.Equal(t, upstream.StatusActive, u.Status)
}

// TestProxyService_AddUpstream_Duplicate 测试重复添加上游。
func TestProxyService_AddUpstream_Duplicate(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "upstream-dup", "upstream-dup@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "upstream-dup.example.com",
	})
	require.NoError(t, err)

	pc, err := svc.CreateProxyConfig(ctx, CreateProxyConfigInput{DomainID: d.ID})
	require.NoError(t, err)

	_, err = svc.AddUpstream(ctx, AddUpstreamInput{
		ProxyConfigID: pc.ID,
		DialAddress:   "10.0.0.1:8080",
	})
	require.NoError(t, err)

	_, err = svc.AddUpstream(ctx, AddUpstreamInput{
		ProxyConfigID: pc.ID,
		DialAddress:   "10.0.0.1:8080",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicate)
}

// TestProxyService_RemoveUpstream 测试移除上游。
func TestProxyService_RemoveUpstream(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "upstream-rm", "upstream-rm@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "upstream-rm.example.com",
	})
	require.NoError(t, err)

	pc, err := svc.CreateProxyConfig(ctx, CreateProxyConfigInput{DomainID: d.ID})
	require.NoError(t, err)

	u, err := svc.AddUpstream(ctx, AddUpstreamInput{
		ProxyConfigID: pc.ID,
		DialAddress:   "10.0.0.1:8080",
	})
	require.NoError(t, err)

	err = svc.RemoveUpstream(ctx, u.ID)
	require.NoError(t, err)

	// 验证上游已标记为 removed
	removed, err := svc.client.Upstream.Get(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, upstream.StatusRemoved, removed.Status)
}

// TestProxyService_ListUpstreams 测试列出上游。
func TestProxyService_ListUpstreams(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "upstream-list", "upstream-list@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "upstream-list.example.com",
	})
	require.NoError(t, err)

	pc, err := svc.CreateProxyConfig(ctx, CreateProxyConfigInput{DomainID: d.ID})
	require.NoError(t, err)

	_, err = svc.AddUpstream(ctx, AddUpstreamInput{
		ProxyConfigID: pc.ID, DialAddress: "10.0.0.1:8080",
	})
	require.NoError(t, err)
	_, err = svc.AddUpstream(ctx, AddUpstreamInput{
		ProxyConfigID: pc.ID, DialAddress: "10.0.0.2:8080",
	})
	require.NoError(t, err)

	upstreams, err := svc.ListUpstreams(ctx, pc.ID)
	require.NoError(t, err)
	assert.Len(t, upstreams, 2)
}

// TestProxyService_AddUpstream_Validation 测试输入校验。
func TestProxyService_AddUpstream_Validation(t *testing.T) {
	svc, _, _, _, ctx := setupProxyService(t)

	tests := []struct {
		name  string
		input AddUpstreamInput
	}{
		{"空代理ID", AddUpstreamInput{ProxyConfigID: uuid.Nil, DialAddress: "10.0.0.1:8080"}},
		{"空地址", AddUpstreamInput{ProxyConfigID: uuid.New(), DialAddress: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.AddUpstream(ctx, tt.input)
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}

// TestLbPolicyConversion 测试负载均衡策略转换。
func TestLbPolicyConversion(t *testing.T) {
	assert.Equal(t, proxyconfig.LbPolicyRandom, toLbPolicy(""))
	assert.Equal(t, proxyconfig.LbPolicyRoundRobin, toLbPolicy("round_robin"))
	assert.Equal(t, proxyconfig.LbPolicyLeastConn, toLbPolicy("least_conn"))
	assert.Equal(t, proxyconfig.LbPolicyIPHash, toLbPolicy("ip_hash"))
	assert.Equal(t, proxyconfig.LbPolicyURIHash, toLbPolicy("uri_hash"))
	assert.Equal(t, proxyconfig.LbPolicyRandom, toLbPolicy("unknown"))
}

// TestNilInt 测试 nilInt 辅助函数。
func TestNilInt(t *testing.T) {
	assert.Nil(t, nilInt(0))
	assert.NotNil(t, nilInt(5))
	assert.Equal(t, 5, *nilInt(5))
}

// TestProxyService_GetProxyConfigByDomainID 测试根据域名 ID 获取代理配置。
func TestProxyService_GetProxyConfigByDomainID(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "proxy-get", "proxy-get@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "proxy-get.example.com",
	})
	require.NoError(t, err)

	pc, err := svc.CreateProxyConfig(ctx, CreateProxyConfigInput{
		DomainID:            d.ID,
		LbPolicy:            "round_robin",
		HealthCheckURI:      "/health",
		HealthCheckInterval: "30s",
		Timeout:             "5s",
	})
	require.NoError(t, err)

	got, err := svc.GetProxyConfigByDomainID(ctx, d.ID)
	require.NoError(t, err)
	assert.Equal(t, pc.ID, got.ID)
	assert.Equal(t, proxyconfig.LbPolicyRoundRobin, got.LbPolicy)
	assert.Equal(t, "/health", got.HealthCheckURI)
}

// TestProxyService_GetProxyConfigByDomainID_NotFound 测试获取不存在的代理配置。
func TestProxyService_GetProxyConfigByDomainID_NotFound(t *testing.T) {
	svc, _, _, _, ctx := setupProxyService(t)

	_, err := svc.GetProxyConfigByDomainID(ctx, uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestProxyService_UpdateProxyConfig 测试更新代理配置。
func TestProxyService_UpdateProxyConfig(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "proxy-update", "proxy-update@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "proxy-update.example.com",
	})
	require.NoError(t, err)

	pc, err := svc.CreateProxyConfig(ctx, CreateProxyConfigInput{DomainID: d.ID})
	require.NoError(t, err)

	updated, err := svc.UpdateProxyConfig(ctx, pc.ID, UpdateProxyConfigInput{
		LbPolicy:            "least_conn",
		HealthCheckInterval: "60s",
		Timeout:             "10s",
	})
	require.NoError(t, err)
	assert.Equal(t, proxyconfig.LbPolicyLeastConn, updated.LbPolicy)
	assert.Equal(t, "60s", updated.HealthCheckInterval)
	assert.Equal(t, "10s", updated.Timeout)
}

// TestProxyService_GetUpstreamStatus 测试获取上游健康状态。
func TestProxyService_GetUpstreamStatus(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "upstream-status", "upstream-status@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "upstream-status.example.com",
	})
	require.NoError(t, err)

	pc, err := svc.CreateProxyConfig(ctx, CreateProxyConfigInput{DomainID: d.ID})
	require.NoError(t, err)

	_, err = svc.AddUpstream(ctx, AddUpstreamInput{
		ProxyConfigID: pc.ID, DialAddress: "10.0.0.1:8080",
	})
	require.NoError(t, err)
	_, err = svc.AddUpstream(ctx, AddUpstreamInput{
		ProxyConfigID: pc.ID, DialAddress: "10.0.0.2:8080",
	})
	require.NoError(t, err)

	statuses, err := svc.GetUpstreamStatus(ctx, d.ID)
	require.NoError(t, err)
	assert.Len(t, statuses, 2)
	assert.Equal(t, "10.0.0.1:8080", statuses[0].DialAddress)
	assert.True(t, statuses[0].Healthy)
}

// TestProxyService_GetUpstreamStatus_NoProxyConfig 测试无代理配置的域名。
func TestProxyService_GetUpstreamStatus_NoProxyConfig(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupProxyService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "upstream-nopc", "upstream-nopc@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "upstream-nopc.example.com",
	})
	require.NoError(t, err)

	_, err = svc.GetUpstreamStatus(ctx, d.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}