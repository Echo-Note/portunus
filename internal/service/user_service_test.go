package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/internal/config"
)

// setupUserService 创建测试用 UserService 实例。
// 使用已有的测试数据库连接。
func setupUserService(t *testing.T) (*UserService, context.Context) {
	t.Helper()

	ctx := context.Background()

	// 连接测试数据库
	cfg := config.DatabaseConfig{
		URL:             "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
	}
	client, err := config.NewEntClient(ctx, cfg)
	require.NoError(t, err, "数据库连接失败")

	// 清理测试数据
	client.ProjectMember.Delete().Exec(ctx) //nolint:errcheck
	client.Project.Delete().Exec(ctx) //nolint:errcheck
	client.User.Delete().Exec(ctx) //nolint:errcheck

	// 生成 JWT 密钥对
	jwtCfg := config.JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	}

	// 从文件读取 JWT 密钥
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

	t.Cleanup(func() {
		client.Close()
	})

	return svc, ctx
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
}

// TestUserService_Register_Duplicate 测试重复注册。
func TestUserService_Register_Duplicate(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.Register(ctx, RegisterInput{
		Email:    "dup@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	// 重复注册应失败
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

	// 注册用户
	_, err := svc.Register(ctx, RegisterInput{
		Email:    "login@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	// 手动激活用户
	users, err := svc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	_, err = users[0].Update().SetStatus("active").Save(ctx)
	require.NoError(t, err)

	// 登录
	pair, err := svc.Login(ctx, LoginInput{
		Email:    "login@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	require.NotNil(t, pair)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
}

// TestUserService_Login_WrongPassword 测试错误密码。
func TestUserService_Login_WrongPassword(t *testing.T) {
	svc, ctx := setupUserService(t)

	_, err := svc.Register(ctx, RegisterInput{
		Email:    "wrongpass@test.com",
		Password: "password123",
	})
	require.NoError(t, err)

	// 激活用户
	users, _ := svc.client.User.Query().All(ctx)
	if len(users) > 0 {
		users[0].Update().SetStatus("active").Save(ctx) //nolint:errcheck
	}

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

	// 不激活，直接登录
	_, err = svc.Login(ctx, LoginInput{
		Email:    "pending@test.com",
		Password: "password123",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
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
	assert.Contains(t, hash, ":") // 格式应为 "salt:hash"

	// 验证密码
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