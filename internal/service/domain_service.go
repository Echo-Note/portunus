package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/domain"
	"github.com/Echo-Note/portunus/ent/generated/project"
)

// DomainService 处理域名相关的业务逻辑。
// 包括域名 CRUD、Caddy 配置翻译与下发、状态管理。
type DomainService struct {
	client       *generated.Client
	stateMachine *StateMachine
	caddyClient  CaddyClient
}

// NewDomainService 创建域名服务实例。
func NewDomainService(client *generated.Client, sm *StateMachine, caddyClient CaddyClient) *DomainService {
	return &DomainService{
		client:       client,
		stateMachine: sm,
		caddyClient:  caddyClient,
	}
}

// CreateDomainInput 创建域名输入参数。
type CreateDomainInput struct {
	ProjectID  uuid.UUID `json:"project_id"`  // 必填，所属项目 ID
	DomainName string    `json:"domain_name"` // 必填，域名（如 example.com）
	SslEnabled bool      `json:"ssl_enabled"` // 是否启用 SSL，默认 true
}

// CreateDomain 创建域名。
// 域名全局唯一，防止同一 Caddy 实例上 host 冲突。
func (s *DomainService) CreateDomain(ctx context.Context, input CreateDomainInput) (*generated.Domain, error) {
	if input.DomainName == "" {
		return nil, fmt.Errorf("%w: 域名不能为空", ErrValidation)
	}
	if input.ProjectID == uuid.Nil {
		return nil, fmt.Errorf("%w: 项目 ID 不能为空", ErrValidation)
	}

	// 检查域名全局唯一性
	exists, err := s.client.Domain.Query().
		Where(domain.DomainNameEQ(input.DomainName)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询域名: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: 域名 %s 已被使用", ErrDuplicate, input.DomainName)
	}

	// 获取项目信息
	proj, err := s.client.Project.Get(ctx, input.ProjectID)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 项目不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询项目: %w", err)
	}

	// 检查项目状态
	if proj.Status != project.StatusActive {
		return nil, fmt.Errorf("%w: 项目状态为 %s", ErrProjectSuspended, proj.Status)
	}

	// 检查域名配额
	domainCount, err := s.client.Domain.Query().
		Where(domain.HasProjectWith(project.IDEQ(input.ProjectID)),
			domain.StatusNEQ(domain.StatusDeleted)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("统计域名: %w", err)
	}
	if domainCount >= proj.MaxDomains {
		return nil, fmt.Errorf("%w: 域名数量已达上限 %d", ErrQuotaExceeded, proj.MaxDomains)
	}

	// 生成 caddy_id
	caddyID := generateCaddyID(proj.ProjectID, "route", uuid.New().String())

	// 构建 Caddy 路由配置
	routeConfig := buildCaddyRouteConfig(input.DomainName, caddyID, input.SslEnabled)

	// 下发 Caddy 路由配置
	_, err = s.caddyClient.PostID(ctx, caddyID, routeConfig, "")
	if err != nil {
		slog.ErrorContext(ctx, "caddy 路由创建失败",
			"domain", input.DomainName,
			"caddy_id", caddyID,
			"err", err,
		)
		return nil, fmt.Errorf("caddy 路由创建失败: %w", err)
	}

	// 创建数据库记录
	d, err := s.client.Domain.Create().
		SetProjectID(input.ProjectID).
		SetDomainName(input.DomainName).
		SetCaddyID(caddyID).
		SetSslEnabled(input.SslEnabled).
		SetStatus(domain.StatusActive).
		Save(ctx)
	if err != nil {
		// Caddy 已创建但 DB 写入失败，尝试补偿删除
		slog.ErrorContext(ctx, "DB 创建域名失败，尝试补偿删除 Caddy 配置",
			"domain", input.DomainName,
			"caddy_id", caddyID,
			"err", err,
		)
		_, etag, getErr := s.caddyClient.GetID(ctx, caddyID)
		if getErr == nil {
			if delErr := s.caddyClient.DeleteID(ctx, caddyID, etag); delErr != nil {
				slog.ErrorContext(ctx, "补偿删除 Caddy 配置失败", "caddy_id", caddyID, "err", delErr)
			}
		}
		return nil, fmt.Errorf("创建域名: %w", err)
	}

	// 创建 Caddy ID 映射
	_, err = s.client.CaddyIDMapping.Create().
		SetID(caddyID).
		SetProjectID(input.ProjectID).
		SetResourceType("domain").
		SetResourceID(d.ID).
		Save(ctx)
	if err != nil {
		slog.WarnContext(ctx, "创建 Caddy ID 映射失败", "caddy_id", caddyID, "err", err)
	}

	slog.InfoContext(ctx, "域名创建成功",
		"domain_id", d.ID,
		"domain_name", input.DomainName,
		"caddy_id", caddyID,
		"project_id", input.ProjectID,
	)

	return d, nil
}

// GetDomain 根据 ID 查询域名。
func (s *DomainService) GetDomain(ctx context.Context, id uuid.UUID) (*generated.Domain, error) {
	d, err := s.client.Domain.Get(ctx, id)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 域名不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询域名: %w", err)
	}
	return d, nil
}

// ListDomainsByProject 列出项目下的所有活跃域名。
func (s *DomainService) ListDomainsByProject(ctx context.Context, projectID uuid.UUID) ([]*generated.Domain, error) {
	domains, err := s.client.Domain.Query().
		Where(domain.ProjectIDEQ(projectID), domain.StatusNEQ(domain.StatusDeleted)).
		WithProxyConfig().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询域名列表: %w", err)
	}
	return domains, nil
}

// DisableDomain 禁用域名代理。
// Caddy 路由保留但返回 503，可随时重新启用。
func (s *DomainService) DisableDomain(ctx context.Context, domainID uuid.UUID) error {
	d, err := s.GetDomain(ctx, domainID)
	if err != nil {
		return err
	}

	if d.Status != domain.StatusActive {
		return fmt.Errorf("%w: 域名状态为 %s，无法禁用", ErrInvalidTransition, d.Status)
	}

	// 更新 Caddy 路由为返回 503
	disableConfig := map[string]any{
		"handler":     "static_response",
		"status_code": 503,
		"body":        "Service Temporarily Disabled",
	}

	_, etag, err := s.caddyClient.GetID(ctx, d.CaddyID)
	if err != nil {
		return fmt.Errorf("获取 Caddy 配置: %w", err)
	}

	_, err = s.caddyClient.PatchID(ctx, d.CaddyID, disableConfig, etag)
	if err != nil {
		return fmt.Errorf("更新 Caddy 路由: %w", err)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "domain",
		EntityID:   domainID.String(),
		FromState:  string(d.Status),
		ToState:    string(domain.StatusDisabled),
		Trigger:    "user_action",
		Reason:     "手动禁用域名",
	})
}

// EnableDomain 重新启用域名代理。
func (s *DomainService) EnableDomain(ctx context.Context, domainID uuid.UUID) error {
	d, err := s.GetDomain(ctx, domainID)
	if err != nil {
		return err
	}

	if d.Status != domain.StatusDisabled {
		return fmt.Errorf("%w: 域名状态为 %s，无法启用", ErrInvalidTransition, d.Status)
	}

	// 恢复原 Caddy 路由配置
	restoreConfig := buildCaddyRouteConfig(d.DomainName, d.CaddyID, d.SslEnabled)

	_, etag, err := s.caddyClient.GetID(ctx, d.CaddyID)
	if err != nil {
		return fmt.Errorf("获取 Caddy 配置: %w", err)
	}

	_, err = s.caddyClient.PatchID(ctx, d.CaddyID, restoreConfig, etag)
	if err != nil {
		return fmt.Errorf("恢复 Caddy 路由: %w", err)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "domain",
		EntityID:   domainID.String(),
		FromState:  string(d.Status),
		ToState:    string(domain.StatusActive),
		Trigger:    "user_action",
		Reason:     "手动启用域名",
	})
}

// UpdateDomainInput 更新域名输入参数。
type UpdateDomainInput struct {
	DomainName string `json:"domain_name"` // 可选，域名名称
	SSLEnabled *bool  `json:"ssl_enabled"` // 可选，是否启用 SSL，使用指针区分"不传"和"传 false"
}

// UpdateDomain 更新域名配置（名称和 SSL 设置）。
// 如果域名名称变更，需要同步更新 Caddy 路由配置。
func (s *DomainService) UpdateDomain(ctx context.Context, domainID uuid.UUID, input UpdateDomainInput) (*generated.Domain, error) {
	d, err := s.GetDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}

	if d.Status == domain.StatusDeleted || d.Status == domain.StatusDeleting {
		return nil, fmt.Errorf("%w: 域名正在删除中", ErrInvalidTransition)
	}

	// 如果修改了域名名称，检查全局唯一性
	if input.DomainName != "" && input.DomainName != d.DomainName {
		exists, err := s.client.Domain.Query().
			Where(domain.DomainNameEQ(input.DomainName)).
			Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("查询域名: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("%w: 域名 %s 已被使用", ErrDuplicate, input.DomainName)
		}
	}

	update := d.Update()
	if input.DomainName != "" {
		update.SetDomainName(input.DomainName)
	}
	if input.SSLEnabled != nil {
		update.SetSslEnabled(*input.SSLEnabled)
	}

	d, err = update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新域名: %w", err)
	}

	// 同步更新 Caddy 路由配置
	if input.DomainName != "" || input.SSLEnabled != nil {
		routeConfig := buildCaddyRouteConfig(d.DomainName, d.CaddyID, d.SslEnabled)
		_, etag, getErr := s.caddyClient.GetID(ctx, d.CaddyID)
		if getErr == nil {
			if _, patchErr := s.caddyClient.PatchID(ctx, d.CaddyID, routeConfig, etag); patchErr != nil {
				slog.WarnContext(ctx, "同步 Caddy 路由配置失败",
					"domain_id", domainID,
					"caddy_id", d.CaddyID,
					"err", patchErr,
				)
			}
		}
	}

	slog.InfoContext(ctx, "域名配置已更新",
		"domain_id", domainID,
		"domain_name", d.DomainName,
		"ssl_enabled", d.SslEnabled,
	)

	return d, nil
}

// DeleteDomain 删除域名及关联的 Caddy 配置节点。
func (s *DomainService) DeleteDomain(ctx context.Context, domainID uuid.UUID) error {
	d, err := s.GetDomain(ctx, domainID)
	if err != nil {
		return err
	}

	if d.Status == domain.StatusDeleted {
		return fmt.Errorf("%w: 域名已删除", ErrNotFound)
	}

	// 先标记为 deleting
	if err := s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "domain",
		EntityID:   domainID.String(),
		FromState:  string(d.Status),
		ToState:    string(domain.StatusDeleting),
		Trigger:    "user_action",
		Reason:     "删除域名",
	}); err != nil {
		return err
	}

	// 删除 Caddy 路由节点
	_, etag, err := s.caddyClient.GetID(ctx, d.CaddyID)
	if err == nil {
		if err := s.caddyClient.DeleteID(ctx, d.CaddyID, etag); err != nil {
			slog.WarnContext(ctx, "删除 Caddy 路由节点失败", "caddy_id", d.CaddyID, "err", err)
		}
	}

	// 删除 Caddy ID 映射
	s.client.CaddyIDMapping.DeleteOneID(d.CaddyID).Exec(ctx) //nolint:errcheck

	// 撤销该域名的所有共享
	s.client.DomainShare.Update().
		SetStatus("revoked").
		Exec(ctx) //nolint:errcheck

	// 标记为已删除
	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "domain",
		EntityID:   domainID.String(),
		FromState:  string(domain.StatusDeleting),
		ToState:    string(domain.StatusDeleted),
		Trigger:    "system",
		Reason:     "Caddy 节点已清理",
	})
}

// generateCaddyID 生成符合规范的 Caddy @id 标识符。
func generateCaddyID(projectID, resourceType, resourceID string) string {
	return fmt.Sprintf("tenant_%s_%s_%s", projectID, resourceType, resourceID[:8])
}

// buildCaddyRouteConfig 构建 Caddy 路由配置 JSON。
func buildCaddyRouteConfig(domainName, caddyID string, sslEnabled bool) map[string]any {
	config := map[string]any{
		"@id": caddyID,
		"match": []map[string]any{
			{
				"host": []string{domainName},
			},
		},
		"handle": []map[string]any{
			{
				"handler": "reverse_proxy",
			},
		},
	}

	if !sslEnabled {
		config["auto_https"] = "off"
	}

	return config
}