package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated/user"
	"github.com/Echo-Note/portunus/internal/config"
	"github.com/Echo-Note/portunus/internal/testutil"
)

// setupUserService 创建测试用 UserService 实例。
func setupUserService(t *testing.T) (*UserService, context.Context) {
	t.Helper()

	ctx := context.Background()

	cfg := config.DatabaseConfig{
		URL:          "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}
	client, err := config.NewEntClient(ctx, cfg)
	require.NoError(t, err, "数据库连接失败")

	// 清理测试数据（按外键依赖顺序删除）
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

	svc, err := NewUserService(client, jwtCfg)
	require.NoError(t, err)

	t.Cleanup(func() { testutil.CloseClient(t, client) })

	return svc, ctx
}

// activateUser 通过邮箱激活用户。
func activateUser(ctx context.Context, t *testing.T, svc *UserService, email string) {
	t.Helper()
	_, err := svc.client.User.Update().
		Where(user.EmailEQ(email)).
		SetStatus(user.StatusActive).
		Save(ctx)
	require.NoError(t, err)
}

// TestUserService_Register_Success 测试成功注册。
func TestUserService_Register_Success(t *testing.T) {
	svc, ctx := setupUserService(t)

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "newuser@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEqual(t, "", result.UserID.String())
	assert.Equal(t, "newuser@test.com", result.Email)

	// 验证用户状态为 pending
	u, err := svc.GetUser(ctx, result.UserID)
	require.NoError(t, err)
	assert.Equal(t, user.StatusPending, u.Status)
}

// TestUserService_Register_Duplicate 测试重复注册。
func TestUserService_Register_Duplicate(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.Register(ctx, RegisterInput{
		Email:    "dup@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	_, err = svc.Register(ctx, RegisterInput{
		Email:    "dup@test.com",
		Password: "password123",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicate)
}

// TestUserService_Register_Validation 测试输入校验。
func TestUserService_Register_Validation(t *testing.T) {
	svc, ctx := setupUserService(t)

	tests := []struct {
		name    string
		input   RegisterInput
		wantErr bool
	}{
		{"空邮箱", RegisterInput{Email: "", Password: "password123"}, true},
		{"短密码", RegisterInput{Email: "test@test.com", Password: "123"}, true},
		{"有效输入", RegisterInput{Email: "valid@test.com", Password: "password123"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(ctx, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUserService_Login_Success 测试成功登录。
func TestUserService_Login_Success(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.Register(ctx, RegisterInput{
		Email:    "login@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	activateUser(ctx, t, svc, "login@test.com")

	pair, err := svc.Login(ctx, LoginInput{
		Email:    "login@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	require.NotNil(t, pair)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)

	// 验证 token 可被解析
	userID, err := svc.VerifyToken(ctx, pair.AccessToken)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, userID)
}

// TestUserService_Login_WrongPassword 测试错误密码。
func TestUserService_Login_WrongPassword(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.Register(ctx, RegisterInput{
		Email:    "wrongpass@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	activateUser(ctx, t, svc, "wrongpass@test.com")

	_, err = svc.Login(ctx, LoginInput{
		Email:    "wrongpass@test.com",
		Password: "wrongpassword",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

// TestUserService_Login_NotActive 测试未激活用户登录。
func TestUserService_Login_NotActive(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.Register(ctx, RegisterInput{
		Email:    "pending@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	_, err = svc.Login(ctx, LoginInput{
		Email:    "pending@test.com",
		Password: "password123",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

// TestUserService_RefreshToken 测试刷新令牌。
func TestUserService_RefreshToken(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.Register(ctx, RegisterInput{
		Email:    "refresh@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	activateUser(ctx, t, svc, "refresh@test.com")

	pair, err := svc.Login(ctx, LoginInput{
		Email:    "refresh@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	newPair, err := svc.RefreshToken(ctx, pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)
}

// TestUserService_GetUser 测试获取用户信息。
func TestUserService_GetUser(t *testing.T) {
	svc, ctx := setupUserService(t)

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "getuser@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	u, err := svc.GetUser(ctx, result.UserID)
	require.NoError(t, err)
	assert.Equal(t, result.Email, u.Email)
}

// TestUserService_GetUser_NotFound 测试获取不存在的用户。
func TestUserService_GetUser_NotFound(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.GetUser(ctx, uuid.Nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestHashPassword 测试密码哈希。
func TestHashPassword(t *testing.T) {
	hash, err := hashPassword("mypassword")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Contains(t, hash, ":")

	assert.True(t, verifyPassword("mypassword", hash))
	assert.False(t, verifyPassword("wrongpassword", hash))
}

// TestSplitN 测试字符串分割。
func TestSplitN(t *testing.T) {
	tests := []struct {
		s, sep string
		n      int
		want   []string
	}{
		{"a:b:c", ":", 2, []string{"a", "b:c"}},
		{"a:b:c", ":", 3, []string{"a", "b", "c"}},
		{"a", ":", 2, []string{"a"}},
		{"salt:hash", ":", 2, []string{"salt", "hash"}},
	}

	for _, tt := range tests {
		result := splitN(tt.s, tt.sep, tt.n)
		assert.Equal(t, tt.want, result)
	}
}

// TestUserService_UpdateUser_Success 测试成功更新用户信息。
func TestUserService_UpdateUser_Success(t *testing.T) {
	svc, ctx := setupUserService(t)

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "updateuser@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	u, err := svc.UpdateUser(ctx, result.UserID, UpdateUserInput{
		Email: "updated@test.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "updated@test.com", u.Email)
}

// TestUserService_UpdateUser_Duplicate 测试更新为已存在的邮箱。
func TestUserService_UpdateUser_Duplicate(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.Register(ctx, RegisterInput{
		Email:    "existing@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "another@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	_, err = svc.UpdateUser(ctx, result.UserID, UpdateUserInput{
		Email: "existing@test.com",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicate)
}

// TestUserService_UpdateUser_NotFound 测试更新不存在的用户。
func TestUserService_UpdateUser_NotFound(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.UpdateUser(ctx, uuid.New(), UpdateUserInput{Email: "test@test.com"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestUserService_UpdateUser_Noop 测试无变更的更新。
func TestUserService_UpdateUser_Noop(t *testing.T) {
	svc, ctx := setupUserService(t)

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "noop@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	// 不传任何变更
	u, err := svc.UpdateUser(ctx, result.UserID, UpdateUserInput{})
	require.NoError(t, err)
	assert.Equal(t, "noop@test.com", u.Email)
}
