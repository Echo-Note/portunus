package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/nrgao/portunus/ent/generated"
	"github.com/nrgao/portunus/ent/generated/proxyconfig"
	"github.com/nrgao/portunus/ent/generated/upstream"
)

// ProxyService 处理反向代理配置和上游管理相关的业务逻辑。
type ProxyService struct {
	client       *generated.Client
	stateMachine *StateMachine
	caddyClient  CaddyClient
}

// NewProxyService 创建代理服务实例。
func NewProxyService(client *generated.Client, sm *StateMachine, caddyClient CaddyClient) *ProxyService {
	return &ProxyService{
		client:       client,
		stateMachine: sm,
		caddyClient:  caddyClient,
	}
}

// CreateProxyConfigInput 创建代理配置输入参数。
type CreateProxyConfigInput struct {
	DomainID            uuid.UUID `json:"domain_id"`             // 必填，所属域名 ID
	LbPolicy            string    `json:"lb_policy"`             // 负载均衡策略，默认 random
	HealthCheckURI      string    `json:"health_check_uri"`      // 可选，健康检查 URI
	HealthCheckInterval string    `json:"health_check_interval"` // 可选，健康检查间隔，默认 30s
	Timeout             string    `json:"timeout"`               // 可选，超时时间，默认 0s
}

// CreateProxyConfig 为域名创建反向代理配置。
func (s *ProxyService) CreateProxyConfig(ctx context.Context, input CreateProxyConfigInput) (*generated.ProxyConfig, error) {
	if input.DomainID == uuid.Nil {
		return nil, fmt.Errorf("%w: 域名 ID 不能为空", ErrValidation)
	}

	// 获取域名信息
	d, err := s.client.Domain.Get(ctx, input.DomainID)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 域名不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询域名: %w", err)
	}

	// 检查是否已有代理配置
	exists, err := s.client.ProxyConfig.Query().
		Where(proxyconfig.DomainIDEQ(input.DomainID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询代理配置: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: 该域名已有代理配置", ErrDuplicate)
	}

	// 获取项目信息以生成 caddy_id
	proj, err := d.QueryProject().Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询项目: %w", err)
	}

	caddyProxyID := generateCaddyID(proj.ProjectID, "proxy", uuid.New().String())

	// 设置默认值
	lbPolicy := toLbPolicy(input.LbPolicy)
	healthCheckInterval := input.HealthCheckInterval
	if healthCheckInterval == "" {
		healthCheckInterval = "30s"
	}
	timeout := input.Timeout
	if timeout == "" {
		timeout = "0s"
	}

	pc, err := s.client.ProxyConfig.Create().
		SetDomainID(input.DomainID).
		SetCaddyProxyID(caddyProxyID).
		SetLbPolicy(lbPolicy).
		SetNillableHealthCheckURI(nilString(input.HealthCheckURI)).
		SetHealthCheckInterval(healthCheckInterval).
		SetTimeout(timeout).
		SetStatus(proxyconfig.StatusActive).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建代理配置: %w", err)
	}

	// 创建 Caddy ID 映射
	s.client.CaddyIDMapping.Create().
		SetID(caddyProxyID).
		SetProjectID(d.ProjectID).
		SetResourceType("proxy_config").
		SetResourceID(pc.ID).
		Save(ctx) //nolint:errcheck

	slog.InfoContext(ctx, "代理配置创建成功",
		"proxy_config_id", pc.ID,
		"domain_id", input.DomainID,
		"caddy_proxy_id", caddyProxyID,
	)

	return pc, nil
}

// AddUpstreamInput 添加上游节点输入参数。
type AddUpstreamInput struct {
	ProxyConfigID uuid.UUID `json:"proxy_config_id"` // 必填，所属代理配置 ID
	DialAddress   string    `json:"dial_address"`    // 必填，上游地址
	Weight        int       `json:"weight"`          // 权重，默认 1
	MaxRequests   int       `json:"max_requests"`    // 可选，最大请求数
}

// AddUpstream 为代理配置添加上游节点。
func (s *ProxyService) AddUpstream(ctx context.Context, input AddUpstreamInput) (*generated.Upstream, error) {
	if input.ProxyConfigID == uuid.Nil {
		return nil, fmt.Errorf("%w: 代理配置 ID 不能为空", ErrValidation)
	}
	if input.DialAddress == "" {
		return nil, fmt.Errorf("%w: 上游地址不能为空", ErrValidation)
	}

	// 检查代理配置是否存在
	pc, err := s.client.ProxyConfig.Get(ctx, input.ProxyConfigID)
	if err != nil {
		if generated.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 代理配置不存在", ErrNotFound)
		}
		return nil, fmt.Errorf("查询代理配置: %w", err)
	}

	// 检查是否已存在相同地址
	exists, err := s.client.Upstream.Query().
		Where(upstream.ProxyConfigIDEQ(input.ProxyConfigID), upstream.DialAddressEQ(input.DialAddress)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询上游: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: 上游地址 %s 已存在", ErrDuplicate, input.DialAddress)
	}

	weight := input.Weight
	if weight <= 0 {
		weight = 1
	}

	u, err := s.client.Upstream.Create().
		SetProxyConfigID(input.ProxyConfigID).
		SetDialAddress(input.DialAddress).
		SetWeight(weight).
		SetNillableMaxRequests(nilInt(input.MaxRequests)).
		SetStatus(upstream.StatusActive).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建上游: %w", err)
	}

	// 更新 Caddy 代理配置
	if err := s.syncUpstreamsToCaddy(ctx, pc.CaddyProxyID, input.ProxyConfigID); err != nil {
		slog.WarnContext(ctx, "同步上游到 Caddy 失败", "proxy_config_id", input.ProxyConfigID, "err", err)
	}


	slog.InfoContext(ctx, "上游添加成功",
		"upstream_id", u.ID,
		"proxy_config_id", input.ProxyConfigID,
		"dial_address", input.DialAddress,
	)

	return u, nil
}

// RemoveUpstream 移除上游节点。
func (s *ProxyService) RemoveUpstream(ctx context.Context, upstreamID uuid.UUID) error {
	u, err := s.client.Upstream.Get(ctx, upstreamID)
	if err != nil {
		if generated.IsNotFound(err) {
			return fmt.Errorf("%w: 上游不存在", ErrNotFound)
		}
		return fmt.Errorf("查询上游: %w", err)
	}

	if u.Status == upstream.StatusRemoved {
		return fmt.Errorf("%w: 上游已移除", ErrNotFound)
	}

	_, err = u.Update().SetStatus(upstream.StatusRemoved).Save(ctx)
	if err != nil {
		return fmt.Errorf("移除上游: %w", err)
	}

	pc, err := s.client.ProxyConfig.Get(ctx, u.ProxyConfigID)
	if err == nil {
		if err := s.syncUpstreamsToCaddy(ctx, pc.CaddyProxyID, u.ProxyConfigID); err != nil {
			slog.WarnContext(ctx, "同步上游到 Caddy 失败", "upstream_id", upstreamID, "err", err)
		}
	}

	slog.InfoContext(ctx, "上游移除成功", "upstream_id", upstreamID)
	return nil
}

// DisableUpstream 禁用上游节点。
func (s *ProxyService) DisableUpstream(ctx context.Context, upstreamID uuid.UUID) error {
	u, err := s.client.Upstream.Get(ctx, upstreamID)
	if err != nil {
		return fmt.Errorf("查询上游: %w", err)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "upstream",
		EntityID:   upstreamID.String(),
		FromState:  string(u.Status),
		ToState:    string(upstream.StatusDisabled),
		Trigger:    "user_action",
		Reason:     "手动禁用上游",
	})
}

// EnableUpstream 重新启用上游节点。
func (s *ProxyService) EnableUpstream(ctx context.Context, upstreamID uuid.UUID) error {
	u, err := s.client.Upstream.Get(ctx, upstreamID)
	if err != nil {
		return fmt.Errorf("查询上游: %w", err)
	}

	return s.stateMachine.ExecuteTransition(ctx, &StateTransition{
		EntityType: "upstream",
		EntityID:   upstreamID.String(),
		FromState:  string(u.Status),
		ToState:    string(upstream.StatusActive),
		Trigger:    "user_action",
		Reason:     "手动启用上游",
	})
}

// ListUpstreams 列出代理配置下的所有活跃上游。
func (s *ProxyService) ListUpstreams(ctx context.Context, proxyConfigID uuid.UUID) ([]*generated.Upstream, error) {
	upstreams, err := s.client.Upstream.Query().
		Where(upstream.ProxyConfigIDEQ(proxyConfigID), upstream.StatusNEQ(upstream.StatusRemoved)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询上游列表: %w", err)
	}
	return upstreams, nil
}

// syncUpstreamsToCaddy 将数据库中的上游列表同步到 Caddy 配置。
func (s *ProxyService) syncUpstreamsToCaddy(ctx context.Context, caddyProxyID string, proxyConfigID uuid.UUID) error {
	upstreams, err := s.client.Upstream.Query().
		Where(upstream.ProxyConfigIDEQ(proxyConfigID), upstream.StatusIn(upstream.StatusActive, upstream.StatusUnhealthy)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("查询上游列表: %w", err)
	}

	var dials []map[string]any
	for _, u := range upstreams {
		dial := map[string]any{
			"dial": u.DialAddress,
		}
		if u.Weight > 1 {
			dial["weight"] = u.Weight
		}
		if u.MaxRequests != 0 {
			dial["max_requests"] = u.MaxRequests
		}
		dials = append(dials, dial)
	}

	proxyConfig := map[string]any{
		"handler":   "reverse_proxy",
		"upstreams": dials,
	}

	_, etag, err := s.caddyClient.GetID(ctx, caddyProxyID)
	if err != nil {
		return fmt.Errorf("获取 Caddy 代理配置: %w", err)
	}

	_, err = s.caddyClient.PatchID(ctx, caddyProxyID, proxyConfig, etag)
	if err != nil {
		return fmt.Errorf("更新 Caddy 代理配置: %w", err)
	}

	return nil
}

// toLbPolicy 将字符串转换为 proxyconfig.LbPolicy 枚举。
func toLbPolicy(policy string) proxyconfig.LbPolicy {
	switch policy {
	case "round_robin":
		return proxyconfig.LbPolicyRoundRobin
	case "least_conn":
		return proxyconfig.LbPolicyLeastConn
	case "ip_hash":
		return proxyconfig.LbPolicyIPHash
	case "uri_hash":
		return proxyconfig.LbPolicyURIHash
	default:
		return proxyconfig.LbPolicyRandom
	}
}

// nilInt 返回 *int，零值返回 nil。
func nilInt(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}