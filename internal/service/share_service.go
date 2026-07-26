package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/domainshare"
)

// ShareService 处理域名共享相关的业务逻辑。
type ShareService struct {
	client       *generated.Client
	stateMachine *StateMachine
}

// NewShareService 创建共享服务实例。
func NewShareService(client *generated.Client, sm *StateMachine) *ShareService {
	return &ShareService{
		client:       client,
		stateMachine: sm,
	}
}

// CreateShareInput 创建共享输入参数。
type CreateShareInput struct {
	DomainID        uuid.UUID  `json:"domain_id"`         // 必填，要共享的域名 ID
	SourceProjectID uuid.UUID  `json:"source_project_id"` // 必填，源项目 ID
	TargetProjectID uuid.UUID  `json:"target_project_id"` // 必填，目标项目 ID
	Permission      string     `json:"permission"`        // 必填，权限：read_only / edit
	ExpiresAt       *time.Time `json:"expires_at"`        // 可选，过期时间
	CreatedBy       uuid.UUID  `json:"created_by"`        // 必填，创建者 ID
}

// CreateShare 将域名从源项目共享给目标项目。
func (s *ShareService) CreateShare(ctx context.Context, input CreateShareInput) (*generated.DomainShare, error) {
	if input.DomainID == uuid.Nil || input.SourceProjectID == uuid.Nil || input.TargetProjectID == uuid.Nil {
		return nil, fmt.Errorf("%w: 域名 ID、源项目 ID、目标项目 ID 不能为空", ErrValidation)
	}

	// 不能共享给自己
	if input.SourceProjectID == input.TargetProjectID {
		return nil, fmt.Errorf("%w: 不能共享给自己", ErrValidation)
	}

	// 校验权限类型
	perm, err := toPermission(input.Permission)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	// 检查是否已有活跃共享
	existing, err := s.client.DomainShare.Query().
		Where(
			domainshare.DomainIDEQ(input.DomainID),
			domainshare.TargetProjectIDEQ(input.TargetProjectID),
			domainshare.StatusIn(domainshare.StatusPending, domainshare.StatusActive),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询共享: %w", err)
	}
	if existing {
		return nil, fmt.Errorf("%w: 该域名已共享给目标项目", ErrDuplicate)
	}

	// 创建共享
	share, err := s.client.DomainShare.Create().
		SetDomainID(input.DomainID).
		SetSourceProjectID(input.SourceProjectID).
		SetTargetProjectID(input.TargetProjectID).
		SetPermission(perm).
		SetStatus(domainshare.StatusActive). // 自动接受
		SetNillableExpiresAt(input.ExpiresAt).
		SetCreatedBy(input.CreatedBy).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建共享: %w", err)
	}

	slog.InfoContext(ctx, "域名共享已创建",
		"share_id", share.ID,
		"domain_id", input.DomainID,
		"source_project", input.SourceProjectID,
		"target_project", input.TargetProjectID,
		"permission", input.Permission,
	)

	return share, nil
}

// RevokeShare 撤销域名共享。
func (s *ShareService) RevokeShare(ctx context.Context, shareID uuid.UUID) error {
	share, err := s.client.DomainShare.Get(ctx, shareID)
	if err != nil {
		if generated.IsNotFound(err) {
			return fmt.Errorf("%w: 共享不存在", ErrNotFound)
		}
		return fmt.Errorf("查询共享: %w", err)
	}

	if share.Status == domainshare.StatusRevoked {
		return fmt.Errorf("%w: 共享已撤销", ErrNotFound)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "share",
		EntityID:   shareID.String(),
		FromState:  string(share.Status),
		ToState:    string(domainshare.StatusRevoked),
		Trigger:    "user_action",
		Reason:     "撤销共享",
	})
}

// ListSharesByDomain 列出域名的所有活跃共享。
func (s *ShareService) ListSharesByDomain(ctx context.Context, domainID uuid.UUID) ([]*generated.DomainShare, error) {
	shares, err := s.client.DomainShare.Query().
		Where(domainshare.DomainIDEQ(domainID), domainshare.StatusEQ(domainshare.StatusActive)).
		WithTargetProject().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询共享列表: %w", err)
	}
	return shares, nil
}

// ListSharesByTargetProject 列出目标项目收到的所有活跃共享。
func (s *ShareService) ListSharesByTargetProject(ctx context.Context, projectID uuid.UUID) ([]*generated.DomainShare, error) {
	shares, err := s.client.DomainShare.Query().
		Where(domainshare.TargetProjectIDEQ(projectID), domainshare.StatusEQ(domainshare.StatusActive)).
		WithDomain().
		WithSourceProject().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询共享列表: %w", err)
	}
	return shares, nil
}

// toPermission 将字符串转换为 domainshare.Permission 枚举。
func toPermission(perm string) (domainshare.Permission, error) {
	switch perm {
	case "read_only":
		return domainshare.PermissionReadOnly, nil
	case "edit":
		return domainshare.PermissionEdit, nil
	default:
		return "", fmt.Errorf("无效的权限类型 %s，可选值: read_only, edit", perm)
	}
}