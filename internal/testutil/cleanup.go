// Package testutil 提供测试专用的辅助函数，可被不同包的测试代码导入。
package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated"
)

// CleanDB 清空数据库中的所有表数据，按外键依赖顺序删除以避免违反外键约束。
// 在测试的 setup 阶段调用，确保每个测试用例在干净的状态下运行。
// 任何删除失败都会直接 fail 当前测试（require.NoError）。
func CleanDB(t *testing.T, client *generated.Client) {
	t.Helper()
	ctx := context.Background()

	// 按外键依赖顺序删除：被引用的表后删，引用其他表的表先删。
	n, err := client.CaddyIDMapping.Delete().Exec(ctx)
	require.NoError(t, err, "清理 CaddyIDMapping 表失败")
	t.Logf("CleanDB: deleted %d rows from CaddyIDMapping", n)

	n, err = client.Upstream.Delete().Exec(ctx)
	require.NoError(t, err, "清理 Upstream 表失败")
	t.Logf("CleanDB: deleted %d rows from Upstream", n)

	n, err = client.ProxyConfig.Delete().Exec(ctx)
	require.NoError(t, err, "清理 ProxyConfig 表失败")
	t.Logf("CleanDB: deleted %d rows from ProxyConfig", n)

	n, err = client.DomainShare.Delete().Exec(ctx)
	require.NoError(t, err, "清理 DomainShare 表失败")
	t.Logf("CleanDB: deleted %d rows from DomainShare", n)

	n, err = client.Domain.Delete().Exec(ctx)
	require.NoError(t, err, "清理 Domain 表失败")
	t.Logf("CleanDB: deleted %d rows from Domain", n)

	n, err = client.ProjectAuditLog.Delete().Exec(ctx)
	require.NoError(t, err, "清理 ProjectAuditLog 表失败")
	t.Logf("CleanDB: deleted %d rows from ProjectAuditLog", n)

	n, err = client.Invitation.Delete().Exec(ctx)
	require.NoError(t, err, "清理 Invitation 表失败")
	t.Logf("CleanDB: deleted %d rows from Invitation", n)

	n, err = client.ProjectMember.Delete().Exec(ctx)
	require.NoError(t, err, "清理 ProjectMember 表失败")
	t.Logf("CleanDB: deleted %d rows from ProjectMember", n)

	n, err = client.ApiToken.Delete().Exec(ctx)
	require.NoError(t, err, "清理 ApiToken 表失败")
	t.Logf("CleanDB: deleted %d rows from ApiToken", n)

	n, err = client.ConfigSnapshot.Delete().Exec(ctx)
	require.NoError(t, err, "清理 ConfigSnapshot 表失败")
	t.Logf("CleanDB: deleted %d rows from ConfigSnapshot", n)

	n, err = client.Project.Delete().Exec(ctx)
	require.NoError(t, err, "清理 Project 表失败")
	t.Logf("CleanDB: deleted %d rows from Project", n)

	n, err = client.User.Delete().Exec(ctx)
	require.NoError(t, err, "清理 User 表失败")
	t.Logf("CleanDB: deleted %d rows from User", n)
}

// CloseClient 在测试清理阶段关闭 Ent Client，错误不会 fail 测试（仅记录日志）。
// 用于 t.Cleanup 回调中，避免 errcheck 警告的同时不因关闭失败中断测试。
func CloseClient(t *testing.T, client *generated.Client) {
	t.Helper()
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		t.Logf("failed to close ent client: %v", err)
	}
}
