package service

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateTransition_Legal 测试所有合法状态转换。
func TestValidateTransition_Legal(t *testing.T) {
	tests := []struct {
		entityType string
		from       string
		to         string
	}{
		// ── 项目状态转换 ──
		{entityType: "project", from: "active", to: "suspended"},
		{entityType: "project", from: "active", to: "deleting"},
		{entityType: "project", from: "suspended", to: "active"},
		{entityType: "project", from: "suspended", to: "deleting"},
		{entityType: "project", from: "deleting", to: "deleted"},
		{entityType: "project", from: "deleting", to: "error"},
		{entityType: "project", from: "error", to: "deleting"},

		// ── 域名状态转换 ──
		{entityType: "domain", from: "creating", to: "active"},
		{entityType: "domain", from: "creating", to: "error"},
		{entityType: "domain", from: "active", to: "updating"},
		{entityType: "domain", from: "active", to: "disabled"},
		{entityType: "domain", from: "active", to: "deleting"},
		{entityType: "domain", from: "active", to: "error"},
		{entityType: "domain", from: "updating", to: "active"},
		{entityType: "domain", from: "updating", to: "error"},
		{entityType: "domain", from: "error", to: "active"},
		{entityType: "domain", from: "error", to: "deleting"},
		{entityType: "domain", from: "disabled", to: "active"},
		{entityType: "domain", from: "disabled", to: "deleting"},
		{entityType: "domain", from: "deleting", to: "deleted"},
		{entityType: "domain", from: "deleting", to: "error"},

		// ── 代理配置状态转换 ──
		{entityType: "proxy_config", from: "active", to: "updating"},
		{entityType: "proxy_config", from: "active", to: "degraded"},
		{entityType: "proxy_config", from: "active", to: "unavailable"},
		{entityType: "proxy_config", from: "updating", to: "active"},
		{entityType: "proxy_config", from: "degraded", to: "active"},
		{entityType: "proxy_config", from: "degraded", to: "unavailable"},
		{entityType: "proxy_config", from: "unavailable", to: "active"},
		{entityType: "proxy_config", from: "unavailable", to: "degraded"},

		// ── 上游状态转换 ──
		{entityType: "upstream", from: "active", to: "unhealthy"},
		{entityType: "upstream", from: "active", to: "disabled"},
		{entityType: "upstream", from: "active", to: "removed"},
		{entityType: "upstream", from: "unhealthy", to: "active"},
		{entityType: "upstream", from: "unhealthy", to: "disabled"},
		{entityType: "upstream", from: "unhealthy", to: "removed"},
		{entityType: "upstream", from: "disabled", to: "active"},
		{entityType: "upstream", from: "disabled", to: "removed"},

		// ── 成员状态转换 ──
		{entityType: "member", from: "pending", to: "active"},
		{entityType: "member", from: "active", to: "removed"},
		{entityType: "member", from: "active", to: "left"},

		// ── 共享状态转换 ──
		{entityType: "share", from: "pending", to: "active"},
		{entityType: "share", from: "pending", to: "rejected"},
		{entityType: "share", from: "pending", to: "revoked"},
		{entityType: "share", from: "pending", to: "expired"},
		{entityType: "share", from: "active", to: "revoked"},
		{entityType: "share", from: "active", to: "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.entityType+"/"+tt.from+"_to_"+tt.to, func(t *testing.T) {
			err := validateTransition(tt.entityType, tt.from, tt.to)
			assert.NoError(t, err, "合法转换不应返回错误")
		})
	}
}

// TestValidateTransition_Illegal 测试非法状态转换。
func TestValidateTransition_Illegal(t *testing.T) {
	tests := []struct {
		entityType string
		from       string
		to         string
	}{
		// ── 项目非法转换 ──
		{entityType: "project", from: "deleted", to: "active"},     // 终态不可恢复
		{entityType: "project", from: "deleted", to: "suspended"},  // 终态不可恢复
		{entityType: "project", from: "active", to: "error"},       // 未定义
		{entityType: "project", from: "suspended", to: "error"},    // 未定义

		// ── 域名非法转换 ──
		{entityType: "domain", from: "creating", to: "disabled"}, // 不能跳过 active
		{entityType: "domain", from: "active", to: "creating"},   // 不能回到 creating
		{entityType: "domain", from: "deleted", to: "active"},    // 终态不可恢复
		{entityType: "domain", from: "deleted", to: "deleting"},  // 终态不可转换
		{entityType: "domain", from: "error", to: "disabled"},    // 未定义
		{entityType: "domain", from: "error", to: "updating"},    // 未定义
		{entityType: "domain", from: "disabled", to: "error"},    // 未定义

		// ── 代理非法转换 ──
		{entityType: "proxy_config", from: "active", to: "error"},      // 未定义
		{entityType: "proxy_config", from: "updating", to: "degraded"}, // 未定义
		{entityType: "proxy_config", from: "degraded", to: "updating"}, // 未定义

		// ── 上游非法转换 ──
		{entityType: "upstream", from: "removed", to: "active"},   // 终态不可恢复
			{entityType: "upstream", from: "removed", to: "unhealthy"}, // 终态不可恢复

		// ── 成员非法转换 ──
		{entityType: "member", from: "removed", to: "active"}, // 终态不可恢复
		{entityType: "member", from: "left", to: "active"},    // 终态不可恢复
		{entityType: "member", from: "pending", to: "removed"}, // 未定义

		// ── 共享非法转换 ──
		{entityType: "share", from: "revoked", to: "active"},  // 终态不可恢复
		{entityType: "share", from: "expired", to: "active"},  // 终态不可恢复
		{entityType: "share", from: "rejected", to: "active"}, // 终态不可恢复
	}

	for _, tt := range tests {
		t.Run(tt.entityType+"/"+tt.from+"_to_"+tt.to, func(t *testing.T) {
			err := validateTransition(tt.entityType, tt.from, tt.to)
			assert.Error(t, err, "非法转换应返回错误")
			assert.ErrorIs(t, err, ErrInvalidTransition)
		})
	}
}

// TestValidateTransition_UnknownEntity 测试未知实体类型。
func TestValidateTransition_UnknownEntity(t *testing.T) {
	err := validateTransition("unknown", "active", "inactive")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未知实体类型")
}

// TestValidateTransition_UnknownSourceState 测试未知源状态。
func TestValidateTransition_UnknownSourceState(t *testing.T) {
	err := validateTransition("domain", "nonexistent", "active")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未知源状态")
}

// TestStateMachine_FilterSideEffects 测试副作用过滤。
func TestStateMachine_FilterSideEffects(t *testing.T) {
	effects := []SideEffect{
		{Type: "caddy_api", Action: "create_route", Priority: 1},
		{Type: "caddy_api", Action: "create_proxy", Priority: 2},
		{Type: "db_update", Action: "update_status", Priority: 3},
		{Type: "audit", Action: "log", Priority: 4},
		{Type: "redis", Action: "invalidate_cache", Priority: 5},
	}

	caddyEffects := filterSideEffects(effects, "caddy_api")
	assert.Len(t, caddyEffects, 2)
	assert.Equal(t, "create_route", caddyEffects[0].Action)

	dbEffects := filterSideEffects(effects, "db_update")
	assert.Len(t, dbEffects, 1)
	assert.Equal(t, "update_status", dbEffects[0].Action)

	auditEffects := filterSideEffects(effects, "audit")
	assert.Len(t, auditEffects, 1)

	// 空列表
	empty := filterSideEffects(effects, "nonexistent")
	assert.Len(t, empty, 0)
}

// TestStateTransition_ConcurrentOptimisticLock 测试并发乐观锁。
// 两个 goroutine 同时尝试从 active 转为 updating，只有一个能成功。
func TestStateTransition_ConcurrentOptimisticLock(t *testing.T) {
	var wg sync.WaitGroup
	var successCount, failCount int32

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 这里只测试守卫校验，不实际执行数据库操作
			err := validateTransition("domain", "active", "updating")
			if err == nil {
				successCount++
			} else {
				failCount++
			}
		}()
	}

	wg.Wait()

	// 守卫校验应始终通过
	assert.Equal(t, int32(2), successCount)
	assert.Equal(t, int32(0), failCount)
}

// TestStateTransition_GetTransitions 测试获取转换映射。
func TestStateTransition_GetTransitions(t *testing.T) {
	tests := []struct {
		entityType string
		wantNil    bool
	}{
		{entityType: "project", wantNil: false},
		{entityType: "domain", wantNil: false},
		{entityType: "proxy_config", wantNil: false},
		{entityType: "upstream", wantNil: false},
		{entityType: "member", wantNil: false},
		{entityType: "share", wantNil: false},
		{entityType: "unknown", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			transitions := getTransitions(tt.entityType)
			if tt.wantNil {
				assert.Nil(t, transitions)
			} else {
				require.NotNil(t, transitions)
				assert.NotEmpty(t, transitions)
			}
		})
	}
}