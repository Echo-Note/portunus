package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/apitoken"
)

// ApiTokenService 处理 API Token 相关的业务逻辑。
type ApiTokenService struct {
	client *generated.Client
}

// NewApiTokenService 创建 API Token 服务实例。
func NewApiTokenService(client *generated.Client) *ApiTokenService {
	return &ApiTokenService{client: client}
}

// CreateApiTokenInput 创建 API Token 输入参数。
type CreateApiTokenInput struct {
	UserID    uuid.UUID `json:"user_id"`    // 必填，所属用户 ID
	ProjectID uuid.UUID `json:"project_id"` // 必填，所属项目 ID
	Name      string    `json:"name"`       // 必填，Token 名称
	Scopes    []string  `json:"scopes"`     // 可选，权限范围
}

// CreateApiTokenOutput 创建 API Token 返回结果。
type CreateApiTokenOutput struct {
	Token       string    `json:"token"`        // 明文 Token（仅创建时返回一次）
	TokenID     uuid.UUID `json:"token_id"`     // Token ID
	TokenPrefix string    `json:"token_prefix"` // Token 前缀（用于识别）
	Name        string    `json:"name"`         // Token 名称
	ProjectID   uuid.UUID `json:"project_id"`   // 所属项目 ID
}

// CreateApiToken 创建 API Token。
// 生成随机 Token，SHA-256 哈希后存储，明文仅返回一次。
func (s *ApiTokenService) CreateApiToken(ctx context.Context, input CreateApiTokenInput) (*CreateApiTokenOutput, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: Token 名称不能为空", ErrValidation)
	}
	if input.ProjectID == uuid.Nil {
		return nil, fmt.Errorf("%w: 项目 ID 不能为空", ErrValidation)
	}

	// 生成随机 Token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("生成 Token: %w", err)
	}
	plainToken := hex.EncodeToString(tokenBytes)

	// SHA-256 哈希
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Token 前缀（前 8 位，用于界面识别）
	tokenPrefix := plainToken[:8]

	// 设置默认 scopes
	scopes := input.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read", "write"}
	}

	// 创建数据库记录
	at, err := s.client.ApiToken.Create().
		SetUserID(input.UserID).
		SetProjectID(input.ProjectID).
		SetName(input.Name).
		SetTokenHash(tokenHash).
		SetTokenPrefix(tokenPrefix).
		SetScopes(scopes).
		SetStatus(apitoken.StatusActive).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建 API Token: %w", err)
	}

	slog.InfoContext(ctx, "API Token 已创建",
		"token_id", at.ID,
		"user_id", input.UserID,
		"project_id", input.ProjectID,
		"name", input.Name,
	)

	return &CreateApiTokenOutput{
		Token:       plainToken,
		TokenID:     at.ID,
		TokenPrefix: tokenPrefix,
		Name:        at.Name,
		ProjectID:   at.ProjectID,
	}, nil
}

// ListApiTokensInput 列出 API Token 输入参数。
type ListApiTokensInput struct {
	UserID    uuid.UUID `json:"user_id"`    // 必填，用户 ID
	ProjectID uuid.UUID `json:"project_id"` // 可选，筛选项目 ID
}

// ListApiTokens 列出用户的 API Token。
func (s *ApiTokenService) ListApiTokens(ctx context.Context, input ListApiTokensInput) ([]*generated.ApiToken, error) {
	query := s.client.ApiToken.Query().
		Where(apitoken.UserIDEQ(input.UserID), apitoken.StatusEQ(apitoken.StatusActive))

	if input.ProjectID != uuid.Nil {
		query = query.Where(apitoken.ProjectIDEQ(input.ProjectID))
	}

	tokens, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询 API Token 列表: %w", err)
	}
	return tokens, nil
}

// RevokeApiToken 撤销 API Token。
func (s *ApiTokenService) RevokeApiToken(ctx context.Context, tokenID uuid.UUID, userID uuid.UUID) error {
	at, err := s.client.ApiToken.Query().
		Where(apitoken.IDEQ(tokenID), apitoken.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return fmt.Errorf("%w: API Token 不存在", ErrNotFound)
		}
		return fmt.Errorf("查询 API Token: %w", err)
	}

	if at.Status != apitoken.StatusActive {
		return fmt.Errorf("%w: API Token 已失效", ErrNotFound)
	}

	_, err = at.Update().
		SetStatus(apitoken.StatusRevoked).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("撤销 API Token: %w", err)
	}

	slog.InfoContext(ctx, "API Token 已撤销",
		"token_id", tokenID,
		"user_id", userID,
	)

	return nil
}

// Ensure time package is used.
var _ = time.Now