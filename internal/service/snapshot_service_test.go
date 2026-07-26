package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/internal/config"
)

// setupSnapshotService 创建测试用 SnapshotService 实例。
func setupSnapshotService(t *testing.T) (*SnapshotService, *ProjectService, *UserService, context.Context) {
	t.Helper()
	ctx := context.Background()

	cfg := config.DatabaseConfig{
		URL:             "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
	}
	client, err := config.NewEntClient(ctx, cfg)
	require.NoError(t, err)

	// 清理
	client.CaddyIDMapping.Delete().Exec(ctx) //nolint:errcheck
	client.Upstream.Delete().Exec(ctx)       //nolint:errcheck
	client.ProxyConfig.Delete().Exec(ctx)    //nolint:errcheck
	client.DomainShare.Delete().Exec(ctx)    //nolint:errcheck
	client.Domain.Delete().Exec(ctx)         //nolint:errcheck
	client.ProjectAuditLog.Delete().Exec(ctx) //nolint:errcheck
	client.Invitation.Delete().Exec(ctx)     //nolint:errcheck
	client.ProjectMember.Delete().Exec(ctx)  //nolint:errcheck
	client.Project.Delete().Exec(ctx)        //nolint:errcheck
	client.ApiToken.Delete().Exec(ctx)       //nolint:errcheck
	client.ConfigSnapshot.Delete().Exec(ctx) //nolint:errcheck
	client.User.Delete().Exec(ctx)           //nolint:errcheck

	jwtCfg := config.JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	}
	privateKeyFile := os.Getenv("JWT_PRIVATE_KEY_FILE")
	if privateKeyFile == "" {
		privateKeyFile = "../../certs/jwt-private.pem"
	}
	publicKeyFile := os.Getenv("JWT_PUBLIC_KEY_FILE")
	if publicKeyFile == "" {
		publicKeyFile = "../../certs/jwt-public.pem"
	}
	if data, err := os.ReadFile(privateKeyFile); err == nil {
		jwtCfg.PrivateKey = string(data)
	}
	if data, err := os.ReadFile(publicKeyFile); err == nil {
		jwtCfg.PublicKey = string(data)
	}

	userSvc, err := NewUserService(client, jwtCfg)
	require.NoError(t, err)

	sm := NewStateMachine(client)
	projectSvc := NewProjectService(client, sm)
	snapshotSvc := NewSnapshotService(client)

	t.Cleanup(func() { client.Close() })
	return snapshotSvc, projectSvc, userSvc, ctx
}

// TestSnapshotService_List_Empty 测试空快照列表。
func TestSnapshotService_List_Empty(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupSnapshotService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "snap-empty", "snap-empty@test.com")

	snapshots, err := svc.ListSnapshots(ctx, projectID, 50, 0)
	require.NoError(t, err)
	assert.Empty(t, snapshots)
}

// TestSnapshotService_List_WithData 测试有快照的列表。
func TestSnapshotService_List_WithData(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupSnapshotService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "snap-list", "snap-list@test.com")

	users, err := userSvc.client.User.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	userID := users[0].ID

	// 创建两个快照
	_, err = svc.client.ConfigSnapshot.Create().
		SetProjectID(projectID).
		SetCaddyJSON("{}").
		SetVersion(1).
		SetChecksum("abc123").
		SetCreatedBy(userID).
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.client.ConfigSnapshot.Create().
		SetProjectID(projectID).
		SetCaddyJSON("{}").
		SetVersion(2).
		SetChecksum("def456").
		SetCreatedBy(userID).
		Save(ctx)
	require.NoError(t, err)

	snapshots, err := svc.ListSnapshots(ctx, projectID, 50, 0)
	require.NoError(t, err)
	assert.Len(t, snapshots, 2)
}

// TestSnapshotService_Rollback_NotFound 测试回滚到不存在的版本。
func TestSnapshotService_Rollback_NotFound(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupSnapshotService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "snap-rb", "snap-rb@test.com")

	err := svc.RollbackSnapshot(ctx, projectID, "999")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestSnapshotService_Rollback_InvalidVersion 测试无效版本号。
func TestSnapshotService_Rollback_InvalidVersion(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupSnapshotService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "snap-inv", "snap-inv@test.com")

	err := svc.RollbackSnapshot(ctx, projectID, "not-a-number")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidation)
}