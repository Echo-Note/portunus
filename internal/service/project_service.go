package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/domain"
	"github.com/Echo-Note/portunus/ent/generated/project"
	"github.com/Echo-Note/portunus/ent/generated/projectmember"
)

// ProjectService 处理项目相关的业务逻辑。
type ProjectService struct {
	client       *generated.Client
	stateMachine *StateMachine
}

// NewProjectService 创建项目服务实例。
func NewProjectService(client *generated.Client, sm *StateMachine) *ProjectService {
	return &ProjectService{
		client:       client,
		stateMachine: sm,
	}
}

// CreateProjectInput 创建项目输入参数。
type CreateProjectInput struct {
	ProjectID   string    `json:"project_id"`   // 必填，项目唯一标识
	Name        string    `json:"name"`         // 必填，项目名称
	Description string    `json:"description"`  // 可选，项目描述
	OwnerID     uuid.UUID `json:"owner_id"`     // 必填，创建者用户 ID
}

// CreateProject 创建新项目，同时创建 owner 成员记录。
func (s *ProjectService) CreateProject(ctx context.Context, input CreateProjectInput) (*generated.Project, error) {
	if input.ProjectID == "" {
		return nil, fmt.Errorf("%w: 项目标识不能为空", ErrValidation)
	}
	if input.Name == "" {
		return nil, fmt.Errorf("%w: 项目名称不能为空", ErrValidation)
	}
	if input.OwnerID == uuid.Nil {
		return nil, fmt.Errorf("%w: 创建者不能为空", ErrValidation)
	}

	// 检查 project_id 唯一性
	exists, err := s.client.Project.Query().
		Where(project.ProjectIDEQ(input.ProjectID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询项目: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: 项目标识 %s 已存在", ErrDuplicate, input.ProjectID)
	}

	// 创建项目
	p, err := s.client.Project.Create().
		SetProjectID(input.ProjectID).
		SetName(input.Name).
		SetNillableDescription(nilString(input.Description)).
		SetCreatedBy(input.OwnerID).
		SetStatus(project.StatusActive).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建项目: %w", err)
	}

	// 创建 owner 成员
	_, err = s.client.ProjectMember.Create().
		SetUserID(input.OwnerID).
		SetProjectID(p.ID).
		SetRole(projectmember.RoleOwner).
		SetStatus(projectmember.StatusActive).
		SetJoinedAt(p.CreatedAt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建 owner 成员: %w", err)
	}

	slog.InfoContext(ctx, "项目创建成功",
		"project_id", p.ID,
		"project_identifier", input.ProjectID,
		"owner_id", input.OwnerID,
	)

	return p, nil
}

// GetProject 根据 ID 查询项目。
func (s *ProjectService) GetProject(ctx context.Context, id uuid.UUID) (*generated.Project, error) {
	p, err := s.client.Project.Get(ctx, id)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 项目不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询项目: %w", err)
	}
	return p, nil
}

// GetProjectByProjectID 根据 project_id 字符串查询项目。
func (s *ProjectService) GetProjectByProjectID(ctx context.Context, projectID string) (*generated.Project, error) {
	p, err := s.client.Project.Query().
		Where(project.ProjectIDEQ(projectID)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 项目不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询项目: %w", err)
	}
	return p, nil
}

// ListUserProjects 列出用户参与的所有活跃项目。
func (s *ProjectService) ListUserProjects(ctx context.Context, userID uuid.UUID) ([]*generated.Project, error) {
	memberProjects, err := s.client.ProjectMember.Query().
		Where(projectmember.UserIDEQ(userID), projectmember.StatusEQ(projectmember.StatusActive)).
		QueryProject().
		Where(project.StatusNEQ(project.StatusDeleted)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询用户项目: %w", err)
	}
	return memberProjects, nil
}

// SuspendProject 冻结项目。
func (s *ProjectService) SuspendProject(ctx context.Context, projectID uuid.UUID, actorID uuid.UUID, reason string) error {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	if p.Status != project.StatusActive {
		return fmt.Errorf("%w: 项目当前状态为 %s，无法冻结", ErrInvalidTransition, p.Status)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "project",
		EntityID:   projectID.String(),
		FromState:  string(p.Status),
		ToState:    string(project.StatusSuspended),
		Trigger:    "user_action",
		ActorID:    actorID.String(),
		Reason:     reason,
	})
}

// ReactivateProject 重新激活已冻结的项目。
func (s *ProjectService) ReactivateProject(ctx context.Context, projectID uuid.UUID, actorID uuid.UUID) error {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	if p.Status != project.StatusSuspended {
		return fmt.Errorf("%w: 项目当前状态为 %s，无法激活", ErrInvalidTransition, p.Status)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "project",
		EntityID:   projectID.String(),
		FromState:  string(p.Status),
		ToState:    string(project.StatusActive),
		Trigger:    "user_action",
		ActorID:    actorID.String(),
		Reason:     "管理员重新激活",
	})
}

// DeleteProject 标记项目为删除中。
func (s *ProjectService) DeleteProject(ctx context.Context, projectID uuid.UUID, actorID uuid.UUID) error {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	if p.Status == project.StatusDeleted {
		return fmt.Errorf("%w: 项目已删除", ErrNotFound)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "project",
		EntityID:   projectID.String(),
		FromState:  string(p.Status),
		ToState:    string(project.StatusDeleting),
		Trigger:    "user_action",
		ActorID:    actorID.String(),
		Reason:     "用户删除项目",
	})
}

// UpdateProjectInput 更新项目输入参数。
type UpdateProjectInput struct {
	Name        string `json:"name"`        // 可选，项目名称
	Description string `json:"description"` // 可选，项目描述
}

// UpdateProject 更新项目元数据（名称和描述）。
func (s *ProjectService) UpdateProject(ctx context.Context, projectID uuid.UUID, input UpdateProjectInput) (*generated.Project, error) {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	update := p.Update()
	if input.Name != "" {
		update.SetName(input.Name)
	}
	// 描述可以为空字符串，通过指针区分"不传"和"传空"
	update.SetDescription(input.Description)

	p, err = update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新项目: %w", err)
	}

	slog.InfoContext(ctx, "项目已更新",
		"project_id", projectID,
		"name", p.Name,
	)

	return p, nil
}

// CheckQuota 检查项目某项配额是否已超限。
func (s *ProjectService) CheckQuota(ctx context.Context, projectID uuid.UUID, quotaType string) error {
	p, err := s.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	switch quotaType {
	case "domains":
		count, err := s.client.Domain.Query().
			Where(domain.ProjectIDEQ(projectID), domain.StatusNEQ(domain.StatusDeleted)).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("统计域名: %w", err)
		}
		if count >= p.MaxDomains {
			return fmt.Errorf("%w: 域名数量已达上限 %d", ErrQuotaExceeded, p.MaxDomains)
		}
	case "members":
		count, err := s.client.ProjectMember.Query().
			Where(projectmember.StatusEQ(projectmember.StatusActive), projectmember.HasProjectWith(project.IDEQ(projectID))).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("统计成员: %w", err)
		}
		if count >= p.MaxMembers {
			return fmt.Errorf("%w: 成员数量已达上限 %d", ErrQuotaExceeded, p.MaxMembers)
		}
	}

	return nil
}

// nilString 返回 *string，空字符串返回 nil。
func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}