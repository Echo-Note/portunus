package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/invitation"
	"github.com/Echo-Note/portunus/ent/generated/project"
	"github.com/Echo-Note/portunus/ent/generated/projectmember"
	"github.com/Echo-Note/portunus/ent/generated/user"
)

// MemberService 处理项目成员和邀请相关的业务逻辑。
type MemberService struct {
	client       *generated.Client
	stateMachine *StateMachine
}

// NewMemberService 创建成员服务实例。
func NewMemberService(client *generated.Client, sm *StateMachine) *MemberService {
	return &MemberService{
		client:       client,
		stateMachine: sm,
	}
}

// InviteMemberInput 邀请成员输入参数。
type InviteMemberInput struct {
	ProjectID uuid.UUID `json:"project_id"` // 必填，项目 ID
	Email     string    `json:"email"`      // 必填，被邀请者邮箱
	Role      string    `json:"role"`       // 必填，角色
	InvitedBy uuid.UUID `json:"invited_by"` // 必填，邀请者 ID
}

// InviteMember 邀请成员加入项目。
func (s *MemberService) InviteMember(ctx context.Context, input InviteMemberInput) (*generated.Invitation, error) {
	if input.Email == "" {
		return nil, fmt.Errorf("%w: 邮箱不能为空", ErrValidation)
	}
	if input.ProjectID == uuid.Nil {
		return nil, fmt.Errorf("%w: 项目 ID 不能为空", ErrValidation)
	}

	// 校验角色
	invRole, err := toInvitationRole(input.Role)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	// 检查是否已有活跃邀请
	existing, err := s.client.Invitation.Query().
		Where(invitation.ProjectIDEQ(input.ProjectID), invitation.EmailEQ(input.Email), invitation.StatusEQ(invitation.StatusPending)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询邀请: %w", err)
	}
	if existing {
		return nil, fmt.Errorf("%w: 已向 %s 发送过邀请", ErrDuplicate, input.Email)
	}

	// 检查是否已是成员
	u, err := s.client.User.Query().Where(user.EmailEQ(input.Email)).Only(ctx)
	if err == nil {
		exists, err := s.client.ProjectMember.Query().
			Where(projectmember.UserIDEQ(u.ID), projectmember.ProjectIDEQ(input.ProjectID), projectmember.StatusEQ(projectmember.StatusActive)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("查询成员: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("%w: 该用户已是项目成员", ErrDuplicate)
		}
	}

	// 生成邀请令牌
	inviteToken := uuid.New().String()

	// 创建邀请
	inv, err := s.client.Invitation.Create().
		SetProjectID(input.ProjectID).
		SetEmail(input.Email).
		SetRole(invRole).
		SetInvitedBy(input.InvitedBy).
		SetInvitationToken(inviteToken).
		SetStatus(invitation.StatusPending).
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建邀请: %w", err)
	}

	slog.InfoContext(ctx, "成员邀请已创建",
		"invitation_id", inv.ID,
		"project_id", input.ProjectID,
		"email", input.Email,
		"role", input.Role,
	)

	return inv, nil
}

// AcceptInvitation 接受邀请加入项目。
func (s *MemberService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	inv, err := s.client.Invitation.Query().
		Where(invitation.InvitationTokenEQ(token), invitation.StatusEQ(invitation.StatusPending)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return fmt.Errorf("%w: 邀请不存在或已失效", ErrNotFound)
		}
		return fmt.Errorf("查询邀请: %w", err)
	}

	// 检查是否过期
	if time.Now().After(inv.ExpiresAt) {
		inv.Update().SetStatus(invitation.StatusExpired).Exec(ctx) //nolint:errcheck
		return fmt.Errorf("%w: 邀请已过期", ErrValidation)
	}

	// 将 invitation.Role 转换为 projectmember.Role
	memberRole := toProjectMemberRole(inv.Role)

	// 创建成员记录
	_, err = s.client.ProjectMember.Create().
		SetUserID(userID).
		SetProjectID(inv.ProjectID).
		SetRole(memberRole).
		SetStatus(projectmember.StatusActive).
		SetInvitedBy(inv.InvitedBy).
		SetJoinedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("创建成员: %w", err)
	}

	// 更新邀请状态
	_, err = inv.Update().
		SetStatus(invitation.StatusAccepted).
		SetAcceptedAt(time.Now()).
		SetAcceptedBy(userID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新邀请状态: %w", err)
	}

	slog.InfoContext(ctx, "成员邀请已接受",
		"invitation_id", inv.ID,
		"user_id", userID,
		"project_id", inv.ProjectID,
	)

	return nil
}

// RejectInvitation 拒绝邀请。
func (s *MemberService) RejectInvitation(ctx context.Context, token string) error {
	inv, err := s.client.Invitation.Query().
		Where(invitation.InvitationTokenEQ(token), invitation.StatusEQ(invitation.StatusPending)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return fmt.Errorf("%w: 邀请不存在或已失效", ErrNotFound)
		}
		return fmt.Errorf("查询邀请: %w", err)
	}

	_, err = inv.Update().SetStatus(invitation.StatusRejected).Save(ctx)
	if err != nil {
		return fmt.Errorf("拒绝邀请: %w", err)
	}

	return nil
}

// RemoveMember 移除项目成员。
func (s *MemberService) RemoveMember(ctx context.Context, projectID, userID, actorID uuid.UUID) error {
	member, err := s.client.ProjectMember.Query().
		Where(projectmember.ProjectIDEQ(projectID), projectmember.UserIDEQ(userID), projectmember.StatusEQ(projectmember.StatusActive)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return fmt.Errorf("%w: 成员不存在", ErrNotFound)
		}
		return fmt.Errorf("查询成员: %w", err)
	}

	// 不能移除 owner
	if member.Role == projectmember.RoleOwner {
		return fmt.Errorf("%w: 不能移除项目所有者", ErrForbidden)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "member",
		EntityID:   fmt.Sprintf("%s-%s", userID, projectID),
		FromState:  string(member.Status),
		ToState:    string(projectmember.StatusRemoved),
		Trigger:    "user_action",
		ActorID:    actorID.String(),
		Reason:     "管理员移除成员",
	})
}

// ChangeMemberRole 变更成员角色。
func (s *MemberService) ChangeMemberRole(ctx context.Context, projectID, userID uuid.UUID, newRole string) error {
	memberRole, err := toProjectMemberRoleString(newRole)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}

	member, err := s.client.ProjectMember.Query().
		Where(projectmember.ProjectIDEQ(projectID), projectmember.UserIDEQ(userID), projectmember.StatusEQ(projectmember.StatusActive)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("查询成员: %w", err)
	}

	// 不能将 owner 降级
	if member.Role == projectmember.RoleOwner {
		return fmt.Errorf("%w: 不能变更项目所有者的角色", ErrForbidden)
	}

	// 不能设为 owner
	if memberRole == projectmember.RoleOwner {
		return fmt.Errorf("%w: 不能将成员设为所有者", ErrForbidden)
	}

	_, err = member.Update().SetRole(memberRole).Save(ctx)
	if err != nil {
		return fmt.Errorf("变更角色: %w", err)
	}

	slog.InfoContext(ctx, "成员角色已变更",
		"project_id", projectID,
		"user_id", userID,
		"new_role", newRole,
	)

	return nil
}

// ListMembers 列出项目所有活跃成员。
func (s *MemberService) ListMembers(ctx context.Context, projectID uuid.UUID) ([]*generated.ProjectMember, error) {
	members, err := s.client.ProjectMember.Query().
		Where(projectmember.ProjectIDEQ(projectID), projectmember.StatusEQ(projectmember.StatusActive)).
		WithUser().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询成员列表: %w", err)
	}
	return members, nil
}

// GetMemberRole 获取用户在项目中的角色。
func (s *MemberService) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (string, error) {
	member, err := s.client.ProjectMember.Query().
		Where(projectmember.ProjectIDEQ(projectID), projectmember.UserIDEQ(userID), projectmember.StatusEQ(projectmember.StatusActive)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return "", fmt.Errorf("%w: 不是项目成员", ErrForbidden)
		}
		return "", fmt.Errorf("查询成员: %w", err)
	}
	return string(member.Role), nil
}

// toInvitationRole 将字符串转换为 invitation.Role 枚举类型。
func toInvitationRole(role string) (invitation.Role, error) {
	switch role {
	case "admin":
		return invitation.RoleAdmin, nil
	case "editor":
		return invitation.RoleEditor, nil
	case "viewer":
		return invitation.RoleViewer, nil
	default:
		return "", fmt.Errorf("无效的角色 %s，可选值: admin, editor, viewer", role)
	}
}

// toProjectMemberRole 将 invitation.Role 转换为 projectmember.Role。
func toProjectMemberRole(r invitation.Role) projectmember.Role {
	switch r {
	case invitation.RoleAdmin:
		return projectmember.RoleAdmin
	case invitation.RoleEditor:
		return projectmember.RoleEditor
	case invitation.RoleViewer:
		return projectmember.RoleViewer
	default:
		return projectmember.RoleViewer
	}
}

// toProjectMemberRoleString 将字符串转换为 projectmember.Role。
func toProjectMemberRoleString(role string) (projectmember.Role, error) {
	switch role {
	case "admin":
		return projectmember.RoleAdmin, nil
	case "editor":
		return projectmember.RoleEditor, nil
	case "viewer":
		return projectmember.RoleViewer, nil
	default:
		return "", fmt.Errorf("无效的角色 %s", role)
	}
}

// Ensure that project and user packages are imported for edge queries.
var _ = project.IDEQ
var _ = user.EmailEQ