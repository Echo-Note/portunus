package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated/caddyidmapping"
	"github.com/Echo-Note/portunus/ent/generated/domain"
	"github.com/Echo-Note/portunus/ent/generated/project"
	"github.com/Echo-Note/portunus/internal/config"
	"github.com/Echo-Note/portunus/internal/testutil"
)

// setupDomainService 创建测试用 DomainService 实例。
func setupDomainService(t *testing.T) (*DomainService, *ProjectService, *UserService, context.Context) {
	t.Helper()
	ctx := context.Background()

	cfg := config.DatabaseConfig{
		URL:          "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}
	client, err := config.NewEntClient(ctx, cfg)
	require.NoError(t, err)

	// 清理
	testutil.CleanDB(t, client)

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

	t.Cleanup(func() { testutil.CloseClient(t, client) })
	return domainSvc, projectSvc, userSvc, ctx
}

// createTestProject 创建测试用项目和 owner 用户。
func createTestProject(t *testing.T, projectSvc *ProjectService, userSvc *UserService, ctx context.Context, projectID, email string) uuid.UUID {
	t.Helper()
	ownerID := createTestUser(t, userSvc, ctx, email)
	p, err := projectSvc.CreateProject(ctx, CreateProjectInput{
		ProjectID: projectID, Name: projectID, OwnerID: ownerID,
	})
	require.NoError(t, err)
	return p.ID
}

// TestDomainService_Create_Success 测试成功创建域名。
func TestDomainService_Create_Success(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-test", "dom-owner@test.com")

	d, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "example.com",
		SslEnabled: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "example.com", d.DomainName)
	assert.Equal(t, domain.StatusActive, d.Status)
	assert.Contains(t, d.CaddyID, "tenant_")

	// 验证 Caddy ID 映射已创建
	exists, err := svc.client.CaddyIDMapping.Query().Where(caddyidmapping.IDEQ(d.CaddyID)).Exist(ctx)
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestDomainService_Create_DuplicateDomain 测试重复域名。
func TestDomainService_Create_DuplicateDomain(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-dup", "dom-dup@test.com")

	_, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "dup.example.com",
	})
	require.NoError(t, err)

	_, err = svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "dup.example.com",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicate)
}

// TestDomainService_Create_ProjectSuspended 测试冻结项目创建域名。
func TestDomainService_Create_ProjectSuspended(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-sus", "dom-sus@test.com")

	// 冻结项目
	_, err := svc.client.Project.UpdateOneID(projectID).SetStatus(project.StatusSuspended).Save(ctx)
	require.NoError(t, err)

	_, err = svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "suspended.example.com",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProjectSuspended)
}

// TestDomainService_GetDomain 测试获取域名。
func TestDomainService_GetDomain(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-get", "dom-get@test.com")

	d, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "get.example.com",
	})
	require.NoError(t, err)

	got, err := svc.GetDomain(ctx, d.ID)
	require.NoError(t, err)
	assert.Equal(t, d.DomainName, got.DomainName)
}

// TestDomainService_GetDomain_NotFound 测试获取不存在的域名。
func TestDomainService_GetDomain_NotFound(t *testing.T) {
	svc, _, _, ctx := setupDomainService(t)

	_, err := svc.GetDomain(ctx, uuid.Nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestDomainService_ListDomainsByProject 测试列出项目域名。
func TestDomainService_ListDomainsByProject(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-list", "dom-list@test.com")

	_, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID: projectID, DomainName: "a.example.com",
	})
	require.NoError(t, err)
	_, err = svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID: projectID, DomainName: "b.example.com",
	})
	require.NoError(t, err)

	domains, err := svc.ListDomainsByProject(ctx, projectID)
	require.NoError(t, err)
	assert.Len(t, domains, 2)
}

// TestDomainService_DeleteDomain 测试删除域名。
func TestDomainService_DeleteDomain(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-del", "dom-del@test.com")

	d, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID: projectID, DomainName: "delete.example.com",
	})
	require.NoError(t, err)

	err = svc.DeleteDomain(ctx, d.ID)
	require.NoError(t, err)

	// 验证域名已标记为删除
	got, err := svc.GetDomain(ctx, d.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDeleted, got.Status)
}

// TestDomainService_DeleteDomain_AlreadyDeleted 测试重复删除。
func TestDomainService_DeleteDomain_AlreadyDeleted(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-del2", "dom-del2@test.com")

	d, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID: projectID, DomainName: "del2.example.com",
	})
	require.NoError(t, err)

	err = svc.DeleteDomain(ctx, d.ID)
	require.NoError(t, err)

	err = svc.DeleteDomain(ctx, d.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestDomainService_Create_Validation 测试输入校验。
func TestDomainService_Create_Validation(t *testing.T) {
	svc, _, _, ctx := setupDomainService(t)

	tests := []struct {
		name  string
		input CreateDomainInput
	}{
		{"空域名", CreateDomainInput{ProjectID: uuid.New(), DomainName: ""}},
		{"空项目ID", CreateDomainInput{ProjectID: uuid.Nil, DomainName: "test.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateDomain(ctx, tt.input)
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}

// TestDomainService_Update_Success 测试成功更新域名。
func TestDomainService_Update_Success(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-update", "dom-update@test.com")

	d, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "update.example.com",
		SslEnabled: true,
	})
	require.NoError(t, err)

	// 更新域名名称
	sslDisabled := false
	updated, err := svc.UpdateDomain(ctx, d.ID, UpdateDomainInput{
		DomainName: "updated.example.com",
		SSLEnabled: &sslDisabled,
	})
	require.NoError(t, err)
	assert.Equal(t, "updated.example.com", updated.DomainName)
	assert.False(t, updated.SslEnabled)
}

// TestDomainService_Update_Duplicate 测试更新为已存在的域名。
func TestDomainService_Update_Duplicate(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-dup-update", "dom-dup-up@test.com")

	_, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "existing.example.com",
	})
	require.NoError(t, err)

	d, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "target.example.com",
	})
	require.NoError(t, err)

	// 尝试更新为已存在的域名
	_, err = svc.UpdateDomain(ctx, d.ID, UpdateDomainInput{
		DomainName: "existing.example.com",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicate)
}

// TestDomainService_Update_Deleting 测试更新正在删除的域名。
func TestDomainService_Update_Deleting(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupDomainService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "dom-del-update", "dom-del-up@test.com")

	d, err := svc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "deleting-update.example.com",
	})
	require.NoError(t, err)

	// 开始删除域名
	err = svc.DeleteDomain(ctx, d.ID)
	require.NoError(t, err)

	// 尝试更新已删除的域名
	_, err = svc.UpdateDomain(ctx, d.ID, UpdateDomainInput{DomainName: "new.example.com"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}
