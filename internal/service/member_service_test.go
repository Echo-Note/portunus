package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated/invitation"
	"github.com/Echo-Note/portunus/ent/generated/projectmember"
	"github.com/Echo-Note/portunus/internal/config"
)

// setupMemberService 创建测试用 MemberService 实例。
func setupMemberService(t *testing.T) (*MemberService, *ProjectService, *UserService, context.Context) {
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
	memberSvc := NewMemberService(client, sm)

	t.Cleanup(func() { client.Close() })
	return memberSvc, projectSvc, userSvc, ctx
}

// TestMemberService_GetInvitation 测试获取邀请详情。
func TestMemberService_GetInvitation(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupMemberService(t)
	ownerID := createTestUser(t, userSvc, ctx, "inv-owner@test.com")
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "inv-get", "inv-get@test.com")

	inv, err := svc.InviteMember(ctx, InviteMemberInput{
		ProjectID: projectID,
		Email:     "invited@test.com",
		Role:      "editor",
		InvitedBy: ownerID,
	})
	require.NoError(t, err)

	got, err := svc.GetInvitation(ctx, inv.InvitationToken)
	require.NoError(t, err)
	assert.Equal(t, inv.ID, got.ID)
	assert.Equal(t, "invited@test.com", got.Email)
	assert.Equal(t, invitation.RoleEditor, got.Role)
	assert.Equal(t, invitation.StatusPending, got.Status)
}

// TestMemberService_GetInvitation_NotFound 测试获取不存在的邀请。
func TestMemberService_GetInvitation_NotFound(t *testing.T) {
	svc, _, _, ctx := setupMemberService(t)

	_, err := svc.GetInvitation(ctx, "non-existent-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestMemberService_AcceptInvitation 测试接受邀请。
func TestMemberService_AcceptInvitation(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupMemberService(t)
	ownerID := createTestUser(t, userSvc, ctx, "accept-owner@test.com")
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "acc-inv", "acc-inv@test.com")

	// 创建被邀请的用户
	invitedUserID := createTestUser(t, userSvc, ctx, "invited-user@test.com")

	inv, err := svc.InviteMember(ctx, InviteMemberInput{
		ProjectID: projectID,
		Email:     "invited-user@test.com",
		Role:      "viewer",
		InvitedBy: ownerID,
	})
	require.NoError(t, err)

	err = svc.AcceptInvitation(ctx, inv.InvitationToken, invitedUserID)
	require.NoError(t, err)

	// 验证邀请已标记为接受
	got, err := svc.GetInvitation(ctx, inv.InvitationToken)
	require.NoError(t, err)
	assert.Equal(t, invitation.StatusAccepted, got.Status)
}

// TestMemberService_AcceptInvitation_Expired 测试接受过期的邀请。
func TestMemberService_AcceptInvitation_Expired(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupMemberService(t)
	ownerID := createTestUser(t, userSvc, ctx, "exp-owner@test.com")
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "exp-inv", "exp-inv@test.com")

	invitedUserID := createTestUser(t, userSvc, ctx, "exp-user@test.com")

	inv, err := svc.InviteMember(ctx, InviteMemberInput{
		ProjectID: projectID,
		Email:     "exp-user@test.com",
		Role:      "viewer",
		InvitedBy: ownerID,
	})
	require.NoError(t, err)

	// 手动将邀请设为过期
	_, err = svc.client.Invitation.UpdateOneID(inv.ID).
		SetExpiresAt(time.Now().Add(-1 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	err = svc.AcceptInvitation(ctx, inv.InvitationToken, invitedUserID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrValidation)
}

// TestMemberService_RejectInvitation 测试拒绝邀请。
func TestMemberService_RejectInvitation(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupMemberService(t)
	ownerID := createTestUser(t, userSvc, ctx, "reject-owner@test.com")
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "rej-inv", "rej-inv@test.com")

	inv, err := svc.InviteMember(ctx, InviteMemberInput{
		ProjectID: projectID,
		Email:     "rejected@test.com",
		Role:      "editor",
		InvitedBy: ownerID,
	})
	require.NoError(t, err)

	err = svc.RejectInvitation(ctx, inv.InvitationToken)
	require.NoError(t, err)

	got, err := svc.GetInvitation(ctx, inv.InvitationToken)
	require.NoError(t, err)
	assert.Equal(t, invitation.StatusRejected, got.Status)
}

// TestMemberService_LeaveProject 测试退出项目。
func TestMemberService_LeaveProject(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupMemberService(t)
	ownerID := createTestUser(t, userSvc, ctx, "leave-owner@test.com")
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "leave-proj", "leave-proj@test.com")

	// 创建另一个用户并邀请加入
	invitedUserID := createTestUser(t, userSvc, ctx, "leave-user@test.com")
	inv, err := svc.InviteMember(ctx, InviteMemberInput{
		ProjectID: projectID,
		Email:     "leave-user@test.com",
		Role:      "editor",
		InvitedBy: ownerID,
	})
	require.NoError(t, err)

	err = svc.AcceptInvitation(ctx, inv.InvitationToken, invitedUserID)
	require.NoError(t, err)

	// 成员退出
	err = svc.LeaveProject(ctx, projectID, invitedUserID)
	require.NoError(t, err)

	// 验证成员已标记为 left
	member, err := svc.client.ProjectMember.Query().
		Where(projectmember.UserIDEQ(invitedUserID), projectmember.ProjectIDEQ(projectID)).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, projectmember.StatusLeft, member.Status)
}

// TestMemberService_LeaveProject_Owner 测试所有者退出（应拒绝）。
func TestMemberService_LeaveProject_Owner(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupMemberService(t)
	// createTestProject 内部会创建 owner 用户并返回项目 ID
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "owner-stay", "owner-stay@test.com")

	// 通过查询成员表获取 owner 的 userID
	members, err := svc.ListMembers(ctx, projectID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	ownerID := members[0].UserID

	err = svc.LeaveProject(ctx, projectID, ownerID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrForbidden)
}

// TestMemberService_LeaveProject_NotMember 测试非成员退出。
func TestMemberService_LeaveProject_NotMember(t *testing.T) {
	svc, projectSvc, userSvc, ctx := setupMemberService(t)
	projectID := createTestProject(t, projectSvc, userSvc, ctx, "no-member", "no-member@test.com")

	randomUser := uuid.New()
	err := svc.LeaveProject(ctx, projectID, randomUser)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}