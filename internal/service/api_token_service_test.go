package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated/apitoken"
	"github.com/Echo-Note/portunus/internal/config"
)

// setupApiTokenService 创建测试用 ApiTokenService 实例。
func setupApiTokenService(t *testing.T) (*ApiTokenService, *ProjectService, *UserService, context.Context) {
	t.Helper()
	ctx := context.Background()

	cfg := config.DatabaseConfig{
		URL:             "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
	}
	client, err := config.NewEntClient(ctx, cfg)
	require.NoError(t, err)

	// 清理
	client.CaddyIDMapping.Delete().Exec(ctx) //nolint:errcheck
	client.Upstream.Delete().Exec(ctx)       //nolint:errcheck
	client.ProxyConfig.Delete().Exec(ctx)    //nolint:errcheck
	client.DomainShare.Delete().Exec(ctx)    //nolint:errcheck
	client.Domain.Delete().Exec(ctx)         //nolint:errcheck
	client.ProjectAuditLog.Delete().Exec(ctx) //nolint:errcheck
	client.Invitation.Delete().Exec(ctx)     //nolint:errcheck
	client.ProjectMember.Delete().Exec(ctx)  //nolint:errcheck
	client.Project.Delete().Exec(ctx)        //nolint:errcheck
	client.ApiToken.Delete().Exec(ctx)       //nolint:errcheck
	client.ConfigSnapshot.Delete().Exec(ctx) //nolint:errcheck
	client.User.Delete().Exec(ctx)           //nolint:errcheck

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
	apiTokenSvc := NewApiTokenService(client)

	t.Cleanup(func() { client.Close() })
	return apiTokenSvc, projectSvc, userSvc, ctx
}

// TestApiTokenService_Create_Success 测试成功创建 API Token。
func TestApiTokenService_Create_Success(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupApiTokenService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "token-test", "token-owner@test.com")

	// 获取 owner 用户 ID
	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	result, err := svc.CreateApiToken(ctx, CreateApiTokenInput{
		UserID:    userID,
		ProjectID: projectID,
		Name:      "测试 Token",
		Scopes:    []string{"read", "write"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	assert.NotEqual(t, uuid.Nil, result.TokenID)
	assert.Equal(t, 8, len(result.TokenPrefix))
	assert.Equal(t, "测试 Token", result.Name)
	assert.Equal(t, projectID, result.ProjectID)

	// 验证数据库中存储的是哈希值，不是明文
	at, err := svc.client.ApiToken.Get(ctx, result.TokenID)
	require.NoError(t, err)
	assert.NotEqual(t, result.Token, at.TokenHash)
	assert.Len(t, at.TokenHash, 64) // SHA-256 十六进制
	assert.Equal(t, apitoken.StatusActive, at.Status)
}

// TestApiTokenService_Create_Validation 测试创建 API Token 的输入校验。
func TestApiTokenService_Create_Validation(t *testing.T) {
	svc, _, _, ctx := setupApiTokenService(t)

	tests := []struct {
		name  string
		input CreateApiTokenInput
	}{
		{"空名称", CreateApiTokenInput{Name: "", ProjectID: uuid.New(), UserID: uuid.New()}},
		{"空项目ID", CreateApiTokenInput{Name: "test", ProjectID: uuid.Nil, UserID: uuid.New()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateApiToken(ctx, tt.input)
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}

// TestApiTokenService_List_Success 测试列出 API Token。
func TestApiTokenService_List_Success(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupApiTokenService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "token-list", "token-list@test.com")

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	// 创建两个 Token
	_, err = svc.CreateApiToken(ctx, CreateApiTokenInput{
		UserID: userID, ProjectID: projectID, Name: "Token A",
	})
	require.NoError(t, err)
	_, err = svc.CreateApiToken(ctx, CreateApiTokenInput{
		UserID: userID, ProjectID: projectID, Name: "Token B",
	})
	require.NoError(t, err)

	// 列出 Token
	tokens, err := svc.ListApiTokens(ctx, ListApiTokensInput{UserID: userID})
	require.NoError(t, err)
	assert.Len(t, tokens, 2)

	// 按项目筛选
	tokens, err = svc.ListApiTokens(ctx, ListApiTokensInput{UserID: userID, ProjectID: projectID})
	require.NoError(t, err)
	assert.Len(t, tokens, 2)
}

// TestApiTokenService_List_Empty 测试无 Token 的用户。
func TestApiTokenService_List_Empty(t *testing.T) {
	svc, _, _, ctx := setupApiTokenService(t)

	tokens, err := svc.ListApiTokens(ctx, ListApiTokensInput{UserID: uuid.New()})
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

// TestApiTokenService_Revoke_Success 测试撤销 API Token。
func TestApiTokenService_Revoke_Success(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupApiTokenService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "token-revoke", "token-revoke@test.com")

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	result, err := svc.CreateApiToken(ctx, CreateApiTokenInput{
		UserID: userID, ProjectID: projectID, Name: "待撤销",
	})
	require.NoError(t, err)

	err = svc.RevokeApiToken(ctx, result.TokenID, userID)
	require.NoError(t, err)

	// 验证 Token 已撤销
	at, err := svc.client.ApiToken.Get(ctx, result.TokenID)
	require.NoError(t, err)
	assert.Equal(t, apitoken.StatusRevoked, at.Status)

	// 撤销后不再出现在列表中
	tokens, err := svc.ListApiTokens(ctx, ListApiTokensInput{UserID: userID})
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

// TestApiTokenService_Revoke_NotFound 测试撤销不存在的 Token。
func TestApiTokenService_Revoke_NotFound(t *testing.T) {
	svc, _, _, ctx := setupApiTokenService(t)

	err := svc.RevokeApiToken(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestApiTokenService_Revoke_AlreadyRevoked 测试重复撤销。
func TestApiTokenService_Revoke_AlreadyRevoked(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupApiTokenService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "token-double-revoke", "token-dr@test.com")

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	result, err := svc.CreateApiToken(ctx, CreateApiTokenInput{
		UserID: userID, ProjectID: projectID, Name: "两次撤销",
	})
	require.NoError(t, err)

	err = svc.RevokeApiToken(ctx, result.TokenID, userID)
	require.NoError(t, err)

	err = svc.RevokeApiToken(ctx, result.TokenID, userID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestApiTokenService_TokenUnique 测试每次生成的 Token 明文都不同。
func TestApiTokenService_TokenUnique(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupApiTokenService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "token-unique", "token-unique@test.com")

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	tokens := make(map[string]bool)
	for i := 0; i < 5; i++ {
		result, err := svc.CreateApiToken(ctx, CreateApiTokenInput{
			UserID: userID, ProjectID: projectID, Name: "Token " + string(rune('A'+i)),
		})
		require.NoError(t, err)
		assert.False(t, tokens[result.Token], "Token 明文不应重复")
		tokens[result.Token] = true
	}
}