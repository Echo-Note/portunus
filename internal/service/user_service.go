// Package service 提供用户相关的业务逻辑。
// UserService 处理注册、登录、JWT 签发与验证、邮箱验证等操作。
package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/user"
	"github.com/Echo-Note/portunus/internal/config"
)

// UserService 处理用户相关的业务逻辑。
// 所有方法接收 context.Context 和普通结构体，不依赖任何 Web 框架。
type UserService struct {
	client    *generated.Client
	jwtCfg    config.JWTConfig
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewUserService 创建用户服务实例。
func NewUserService(client *generated.Client, jwtCfg config.JWTConfig) (*UserService, error) {
	svc := &UserService{
		client: client,
		jwtCfg: jwtCfg,
	}

	// 解析 JWT 密钥对
	if jwtCfg.PrivateKey != "" {
		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(jwtCfg.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("解析 JWT 私钥失败: %w", err)
		}
		svc.privateKey = privateKey
	}

	if jwtCfg.PublicKey != "" {
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(jwtCfg.PublicKey))
		if err != nil {
			return nil, fmt.Errorf("解析 JWT 公钥失败: %w", err)
		}
		svc.publicKey = publicKey
	}

	return svc, nil
}

// RegisterInput 用户注册输入参数。
type RegisterInput struct {
	Email    string `json:"email"`    // 必填，邮箱地址
	Password string `json:"password"` // 必填，明文密码（至少 8 位）
}

// RegisterOutput 用户注册返回结果。
type RegisterOutput struct {
	UserID uuid.UUID `json:"user_id"` // 新创建的用户 ID
	Email  string    `json:"email"`   // 注册邮箱
}

// Register 用户注册。
// 校验邮箱唯一性，使用 Argon2id 哈希密码，创建用户记录。
func (s *UserService) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	// 校验邮箱格式
	if input.Email == "" {
		return nil, fmt.Errorf("%w: 邮箱不能为空", ErrValidation)
	}
	if len(input.Password) < 8 {
		return nil, fmt.Errorf("%w: 密码至少需要 8 位", ErrValidation)
	}

	// 检查邮箱是否已注册
	exists, err := s.client.User.Query().
		Where(user.EmailEQ(input.Email)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询用户: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: 邮箱 %s 已注册", ErrDuplicate, input.Email)
	}

	// 哈希密码
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 创建用户
	u, err := s.client.User.Create().
		SetEmail(input.Email).
		SetPasswordHash(passwordHash).
		SetStatus(user.StatusPending).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建用户: %w", err)
	}

	slog.InfoContext(ctx, "用户注册成功",
		"user_id", u.ID,
		"email", input.Email,
	)

	return &RegisterOutput{
		UserID: u.ID,
		Email:  u.Email,
	}, nil
}

// LoginInput 用户登录输入参数。
type LoginInput struct {
	Email    string `json:"email"`    // 必填
	Password string `json:"password"` // 必填
}

// TokenPair JWT 令牌对。
type TokenPair struct {
	AccessToken  string    `json:"access_token"`  // 访问令牌（15 分钟有效期）
	RefreshToken string    `json:"refresh_token"` // 刷新令牌（7 天有效期）
	ExpiresAt    time.Time `json:"expires_at"`    // access_token 过期时间
	TokenType    string    `json:"token_type"`    // 固定为 "Bearer"
}

// Login 用户登录。
// 验证邮箱和密码，签发 JWT 令牌对。
func (s *UserService) Login(ctx context.Context, input LoginInput) (*TokenPair, error) {
	if input.Email == "" || input.Password == "" {
		return nil, fmt.Errorf("%w: 邮箱和密码不能为空", ErrValidation)
	}

	// 查询用户
	u, err := s.client.User.Query().
		Where(user.EmailEQ(input.Email)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 邮箱或密码错误", ErrUnauthorized)
		}
		return nil, fmt.Errorf("查询用户: %w", err)
	}

	// 检查用户状态
	if u.Status != user.StatusActive {
		return nil, fmt.Errorf("%w: 用户状态为 %s，无法登录", ErrUnauthorized, u.Status)
	}

	// 验证密码
	if !verifyPassword(input.Password, u.PasswordHash) {
		return nil, fmt.Errorf("%w: 邮箱或密码错误", ErrUnauthorized)
	}

	// 签发令牌
	pair, err := s.issueTokenPair(ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("签发令牌: %w", err)
	}

	// 更新最后登录时间
	_, err = s.client.User.UpdateOneID(u.ID).
		SetLastLoginAt(time.Now()).
		Save(ctx)
	if err != nil {
		slog.WarnContext(ctx, "更新最后登录时间失败", "err", err)
	}

	slog.InfoContext(ctx, "用户登录成功",
		"user_id", u.ID,
		"email", input.Email,
	)

	return pair, nil
}

// RefreshToken 使用 refresh_token 刷新 access_token。
func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// 解析 refresh_token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: refresh_token 无效或已过期", ErrUnauthorized)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("%w: refresh_token 声明无效", ErrUnauthorized)
	}

	// 验证 token 类型
	if tokenType, _ := claims["type"].(string); tokenType != "refresh" {
		return nil, fmt.Errorf("%w: 非 refresh_token", ErrUnauthorized)
	}

	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return nil, fmt.Errorf("%w: 用户 ID 无效", ErrUnauthorized)
	}

	// 验证用户状态
	u, err := s.client.User.Get(ctx, userID)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrUnauthorized)
		}
		return nil, fmt.Errorf("查询用户: %w", err)
	}
	if u.Status != user.StatusActive {
		return nil, fmt.Errorf("%w: 用户状态为 %s", ErrUnauthorized, u.Status)
	}

	return s.issueTokenPair(ctx, userID)
}

// VerifyToken 验证 access_token 并返回用户 ID。
func (s *UserService) VerifyToken(ctx context.Context, accessToken string) (uuid.UUID, error) {
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("非预期的签名方法: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("%w: access_token 无效或已过期", ErrUnauthorized)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, fmt.Errorf("%w: access_token 声明无效", ErrUnauthorized)
	}

	if tokenType, _ := claims["type"].(string); tokenType != "access" {
		return uuid.Nil, fmt.Errorf("%w: 非 access_token", ErrUnauthorized)
	}

	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: 用户 ID 无效", ErrUnauthorized)
	}

	return userID, nil
}

// GetUser 根据 ID 查询用户信息。
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*generated.User, error) {
	u, err := s.client.User.Get(ctx, id)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询用户: %w", err)
	}
	return u, nil
}

// issueTokenPair 签发 JWT 令牌对。
func (s *UserService) issueTokenPair(ctx context.Context, userID uuid.UUID) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.jwtCfg.AccessTokenTTL)
	refreshExpiry := now.Add(s.jwtCfg.RefreshTokenTTL)

	// 签发 access_token
	accessClaims := jwt.MapClaims{
		"sub":   userID.String(),
		"type":  "access",
		"iat":   now.Unix(),
		"exp":   accessExpiry.Unix(),
		"jti":   uuid.New().String(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("签发 access_token: %w", err)
	}

	// 签发 refresh_token
	refreshClaims := jwt.MapClaims{
		"sub":   userID.String(),
		"type":  "refresh",
		"iat":   now.Unix(),
		"exp":   refreshExpiry.Unix(),
		"jti":   uuid.New().String(),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims).SignedString(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("签发 refresh_token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiry,
		TokenType:    "Bearer",
	}, nil
}

// hashPassword 使用 Argon2id 哈希密码。
// 返回 base64 编码的哈希值，格式为 "salt:hash"。
func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成盐值失败: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return base64.RawStdEncoding.EncodeToString(salt) + ":" + base64.RawStdEncoding.EncodeToString(hash), nil
}

// verifyPassword 验证密码是否与哈希值匹配。
func verifyPassword(password, encoded string) bool {
	parts := splitN(encoded, ":", 2)
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	actualHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return sha256.Sum256(actualHash) == sha256.Sum256(expectedHash)
}

// splitN 分割字符串，最多分割成 n 部分。
func splitN(s, sep string, n int) []string {
	result := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep)
		if idx < 0 {
			result = append(result, s)
			return result
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	result = append(result, s)
	return result
}

// indexOf 返回子串在字符串中的位置，未找到返回 -1。
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}