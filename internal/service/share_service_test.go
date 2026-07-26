package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated/domainshare"
	"github.com/Echo-Note/portunus/internal/config"
	"github.com/Echo-Note/portunus/internal/testutil"
)

// setupShareService 创建测试用 ShareService 实例。
func setupShareService(t *testing.T) (*ShareService, *DomainService, *ProjectService, *UserService, context.Context) {
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
	shareSvc := NewShareService(client, sm)

	t.Cleanup(func() { testutil.CloseClient(t, client) })
	return shareSvc, domainSvc, projectSvc, userSvc, ctx
}

// TestShareService_Create_Success 测试成功创建共享。
func TestShareService_Create_Success(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	sourceProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-src", "share-src@test.com")
	targetProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-tgt", "share-tgt@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  sourceProjectID,
		DomainName: "share-test.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	share, err := svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: sourceProjectID,
		TargetProjectID: targetProjectID,
		Permission:      "read_only",
		CreatedBy:       userID,
	})
	require.NoError(t, err)
	assert.Equal(t, d.ID, share.DomainID)
	assert.Equal(t, domainshare.PermissionReadOnly, share.Permission)
	assert.Equal(t, domainshare.StatusActive, share.Status)
}

// TestShareService_Create_Duplicate 测试重复共享。
func TestShareService_Create_Duplicate(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	sourceProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-dup-src", "share-dup-src@test.com")
	targetProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-dup-tgt", "share-dup-tgt@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  sourceProjectID,
		DomainName: "share-dup.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	_, err = svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: sourceProjectID,
		TargetProjectID: targetProjectID,
		Permission:      "read_only",
		CreatedBy:       userID,
	})
	require.NoError(t, err)

	_, err = svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: sourceProjectID,
		TargetProjectID: targetProjectID,
		Permission:      "read_only",
		CreatedBy:       userID,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicate)
}

// TestShareService_Create_SelfShare 测试共享给自己。
func TestShareService_Create_SelfShare(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "share-self", "share-self@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  projectID,
		DomainName: "share-self.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	_, err = svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: projectID,
		TargetProjectID: projectID,
		Permission:      "read_only",
		CreatedBy:       userID,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidation)
}

// TestShareService_ListByDomain 测试列出域名的共享。
func TestShareService_ListByDomain(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	sourceProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-list-src", "share-list-src@test.com")
	targetProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-list-tgt", "share-list-tgt@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  sourceProjectID,
		DomainName: "share-list.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	_, err = svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: sourceProjectID,
		TargetProjectID: targetProjectID,
		Permission:      "read_only",
		CreatedBy:       userID,
	})
	require.NoError(t, err)

	shares, err := svc.ListSharesByDomain(ctx, d.ID)
	require.NoError(t, err)
	assert.Len(t, shares, 1)
}

// TestShareService_ListByTargetProject 测试列出目标项目收到的共享。
func TestShareService_ListByTargetProject(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	sourceProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-tgt-src", "share-tgt-src@test.com")
	targetProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-tgt-tgt", "share-tgt-tgt@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  sourceProjectID,
		DomainName: "share-by-target.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	_, err = svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: sourceProjectID,
		TargetProjectID: targetProjectID,
		Permission:      "read_only",
		CreatedBy:       userID,
	})
	require.NoError(t, err)

	shares, err := svc.ListSharesByTargetProject(ctx, targetProjectID)
	require.NoError(t, err)
	assert.Len(t, shares, 1)
}

// TestShareService_ListSharesByUser 测试列出用户收到的共享。
func TestShareService_ListSharesByUser(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	sourceProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-user-src", "share-user-src@test.com")
	targetProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-user-tgt", "share-user-tgt@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  sourceProjectID,
		DomainName: "share-by-user.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(users), 2)

	// 源项目 owner 创建共享
	_, err = svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: sourceProjectID,
		TargetProjectID: targetProjectID,
		Permission:      "read_only",
		CreatedBy:       users[0].ID,
	})
	require.NoError(t, err)

	// 目标项目 owner 应能看到收到的共享
	shares, err := svc.ListSharesByUser(ctx, users[1].ID)
	require.NoError(t, err)
	assert.Len(t, shares, 1)
}

// TestShareService_Revoke 测试撤销共享。
func TestShareService_Revoke(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	sourceProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-revoke-src", "share-revoke-src@test.com")
	targetProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-revoke-tgt", "share-revoke-tgt@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  sourceProjectID,
		DomainName: "share-revoke.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	share, err := svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: sourceProjectID,
		TargetProjectID: targetProjectID,
		Permission:      "read_only",
		CreatedBy:       userID,
	})
	require.NoError(t, err)

	err = svc.RevokeShare(ctx, share.ID)
	require.NoError(t, err)

	got, err := svc.client.DomainShare.Get(ctx, share.ID)
	require.NoError(t, err)
	assert.Equal(t, domainshare.StatusRevoked, got.Status)
}

// TestShareService_AcceptShare 测试接受共享。
func TestShareService_AcceptShare(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	sourceProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-accept-src", "share-accept-src@test.com")
	targetProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-accept-tgt", "share-accept-tgt@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  sourceProjectID,
		DomainName: "share-accept.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	// 创建 pending 状态的共享
	share, err := svc.client.DomainShare.Create().
		SetDomainID(d.ID).
		SetSourceProjectID(sourceProjectID).
		SetTargetProjectID(targetProjectID).
		SetPermission(domainshare.PermissionReadOnly).
		SetStatus(domainshare.StatusPending).
		SetCreatedBy(userID).
		Save(ctx)
	require.NoError(t, err)

	err = svc.AcceptShare(ctx, share.ID)
	require.NoError(t, err)

	got, err := svc.client.DomainShare.Get(ctx, share.ID)
	require.NoError(t, err)
	assert.Equal(t, domainshare.StatusActive, got.Status)
}

// TestShareService_AcceptShare_NotPending 测试接受非 pending 状态的共享。
func TestShareService_AcceptShare_NotPending(t *testing.T) {
	svc, domainSvc, projectSvc, userSvc, ctx := setupShareService(t)
	sourceProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-np-src", "share-np-src@test.com")
	targetProjectID := createTestProject(t, projectSvc, userSvc, ctx, "share-np-tgt", "share-np-tgt@test.com")

	d, err := domainSvc.CreateDomain(ctx, CreateDomainInput{
		ProjectID:  sourceProjectID,
		DomainName: "share-np.example.com",
	})
	require.NoError(t, err)

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	// 创建并直接自动接受
	share, err := svc.CreateShare(ctx, CreateShareInput{
		DomainID:        d.ID,
		SourceProjectID: sourceProjectID,
		TargetProjectID: targetProjectID,
		Permission:      "read_only",
		CreatedBy:       userID,
	})
	require.NoError(t, err)

	// 试图接受已 active 的共享
	err = svc.AcceptShare(ctx, share.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

// TestShareService_Validation 测试创建共享的输入校验。
func TestShareService_Validation(t *testing.T) {
	svc, _, _, _, ctx := setupShareService(t)

	tests := []struct {
		name  string
		input CreateShareInput
	}{
		{"空域名ID", CreateShareInput{DomainID: uuid.Nil, SourceProjectID: uuid.New(), TargetProjectID: uuid.New(), Permission: "read_only"}},
		{"空源项目ID", CreateShareInput{DomainID: uuid.New(), SourceProjectID: uuid.Nil, TargetProjectID: uuid.New(), Permission: "read_only"}},
		{"空目标项目ID", CreateShareInput{DomainID: uuid.New(), SourceProjectID: uuid.New(), TargetProjectID: uuid.Nil, Permission: "read_only"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateShare(ctx, tt.input)
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}
