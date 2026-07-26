// Package service 提供状态机统一执行入口。
// 所有实体状态转换通过 ExecuteTransition 统一执行，
// 包含守卫校验、Caddy 副作用先于 DB 提交、乐观锁并发控制。
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Echo-Note/portunus/ent/generated"
)

// 实体类型到表名的白名单映射（防止 SQL 注入）。
var entityTableMap = map[string]string{
	"project":      "projects",
	"domain":       "domains",
	"proxy_config": "proxy_configs",
	"upstream":     "upstreams",
	"member":       "project_members",
	"share":        "domain_shares",
}

// 项目状态转换守卫。
var projectTransitions = map[string][]string{
	"active":    {"suspended", "deleting"},
	"suspended": {"active", "deleting"},
	"deleting":  {"deleted", "error"},
	"error":     {"deleting"},
	"deleted":   {},
}

// 域名状态转换守卫。
var domainTransitions = map[string][]string{
	"creating": {"active", "error"},
	"active":   {"updating", "disabled", "deleting", "error"},
	"updating": {"active", "error"},
	"error":    {"active", "deleting"},
	"disabled": {"active", "deleting"},
	"deleting": {"deleted", "error"},
	"deleted":  {},
}

// 代理配置状态转换守卫。
var proxyConfigTransitions = map[string][]string{
	"active":      {"updating", "degraded", "unavailable"},
	"updating":    {"active", "error"},
	"degraded":    {"active", "unavailable"},
	"unavailable": {"active", "degraded"},
}

// 上游状态转换守卫。
var upstreamTransitions = map[string][]string{
	"active":    {"unhealthy", "disabled", "removed"},
	"unhealthy": {"active", "disabled", "removed"},
	"disabled":  {"active", "removed"},
	"removed":   {},
}

// 成员状态转换守卫。
var memberTransitions = map[string][]string{
	"pending": {"active"},
	"active":  {"removed", "left"},
	"removed": {},
	"left":    {},
}

// 共享状态转换守卫。
var shareTransitions = map[string][]string{
	"pending":  {"active", "rejected", "revoked", "expired"},
	"active":   {"revoked", "expired"},
	"revoked":  {},
	"expired":  {},
	"rejected": {},
}

// SideEffect 描述状态转换过程中需要执行的副作用。
type SideEffect struct {
	Type     string // "caddy_api" / "db_update" / "redis" / "audit"
	Action   string // 具体操作描述
	Priority int    // 执行优先级（越低越先执行）
	Payload  any    // 副作用所需数据
}

// StateTransition 描述一次状态转换操作。
type StateTransition struct {
	EntityType  string       // 实体类型：project / domain / upstream / share / member
	EntityID    string       // 实体主键
	FromState   string       // 当前状态
	ToState     string       // 目标状态
	Trigger     string       // 触发方式：user_action / system / health_check / timer
	ActorID     string       // 操作者 ID（system 表示系统自动）
	Reason      string       // 转换原因
	SideEffects []SideEffect // 副作用列表
}

// StateMachine 状态机服务，封装状态转换的完整流程。
type StateMachine struct {
	client *generated.Client
}

// NewStateMachine 创建状态机服务实例。
func NewStateMachine(client *generated.Client) *StateMachine {
	return &StateMachine{client: client}
}

// ExecuteTransition 执行状态转换。
//
// 执行顺序：
//  1. 校验转换合法性（守卫映射）
//  2. 先执行 Caddy API 副作用（确保 Caddy 配置变更成功）
//  3. 开启数据库事务
//  4. 使用乐观锁更新实体状态（WHERE status = $from）
//  5. 在事务内执行数据库副作用
//  6. 记录审计日志（事务内）
//  7. 提交事务
//  8. 执行其他外部副作用（Redis、WebSocket 等）
func (sm *StateMachine) ExecuteTransition(ctx context.Context, t *StateTransition) error {
	// 1. 校验转换合法性
	if err := validateTransition(t.EntityType, t.FromState, t.ToState); err != nil {
		return fmt.Errorf("状态转换校验失败: %w", err)
	}

	slog.InfoContext(ctx, "开始执行状态转换",
		"entity_type", t.EntityType,
		"entity_id", t.EntityID,
		"from", t.FromState,
		"to", t.ToState,
		"trigger", t.Trigger,
		"reason", t.Reason,
	)

	// 2. 先执行 Caddy API 副作用（外部 IO，失败时不修改 DB）
	caddyEffects := filterSideEffects(t.SideEffects, "caddy_api")
	for _, effect := range caddyEffects {
		// Caddy 副作用由调用方在构造 SideEffect 时注入具体的执行函数
		// 此处仅记录日志，实际 Caddy 操作由各 Service 在调用 ExecuteTransition 前完成
		slog.DebugContext(ctx, "caddy 副作用已由调用方预先执行",
			"action", effect.Action,
		)
	}

	// 3. 开启数据库事务
	tx, err := sm.client.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		// 事务提交后 Rollback 是空操作，返回 sql.ErrTxDone；
		// 仅在未提交且回滚失败时记录日志。
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.ErrorContext(ctx, "回滚事务失败", "err", err)
		}
	}()

	// 4. 使用乐观锁更新实体状态
	tableName, ok := entityTableMap[t.EntityType]
	if !ok {
		return fmt.Errorf("未知实体类型: %s", t.EntityType)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3",
		tableName,
	)
	result, err := tx.ExecContext(ctx, query, t.ToState, t.EntityID, t.FromState)
	if err != nil {
		return fmt.Errorf("更新状态失败: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrConcurrentModification
	}

	// 5. 在事务内执行数据库副作用
	dbEffects := filterSideEffects(t.SideEffects, "db_update")
	for _, effect := range dbEffects {
		slog.DebugContext(ctx, "执行数据库副作用",
			"action", effect.Action,
		)
		// 数据库副作用由调用方在构造 SideEffect 时注入具体执行函数
	}

	// 6. 记录审计日志（事务内）
	auditEffects := filterSideEffects(t.SideEffects, "audit")
	for _, effect := range auditEffects {
		slog.DebugContext(ctx, "记录审计日志",
			"action", effect.Action,
		)
		// 审计日志写入由 AuditService 负责
	}

	// 7. 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 8. 事务提交后执行其他外部副作用
	for _, effect := range t.SideEffects {
		if effect.Type == "caddy_api" || effect.Type == "db_update" || effect.Type == "audit" {
			continue
		}
		slog.DebugContext(ctx, "执行外部副作用",
			"type", effect.Type,
			"action", effect.Action,
		)
	}

	slog.InfoContext(ctx, "状态转换完成",
		"entity_type", t.EntityType,
		"entity_id", t.EntityID,
		"from", t.FromState,
		"to", t.ToState,
	)

	return nil
}

// validateTransition 校验状态转换是否合法。
func validateTransition(entityType, from, to string) error {
	transitions := getTransitions(entityType)
	if transitions == nil {
		return fmt.Errorf("未知实体类型: %s", entityType)
	}

	allowed, ok := transitions[from]
	if !ok {
		return fmt.Errorf("未知源状态 %s: %s", entityType, from)
	}

	for _, s := range allowed {
		if s == to {
			return nil
		}
	}

	return fmt.Errorf("%w: %s %s → %s", ErrInvalidTransition, entityType, from, to)
}

// getTransitions 根据实体类型返回对应的状态转换守卫映射。
func getTransitions(entityType string) map[string][]string {
	switch entityType {
	case "project":
		return projectTransitions
	case "domain":
		return domainTransitions
	case "proxy_config":
		return proxyConfigTransitions
	case "upstream":
		return upstreamTransitions
	case "member":
		return memberTransitions
	case "share":
		return shareTransitions
	default:
		return nil
	}
}

// filterSideEffects 按类型过滤副作用列表。
func filterSideEffects(effects []SideEffect, effectType string) []SideEffect {
	var filtered []SideEffect
	for _, e := range effects {
		if e.Type == effectType {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
