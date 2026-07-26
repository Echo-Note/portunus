package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/ent/generated/project"
	"github.com/Echo-Note/portunus/ent/generated/user"
	"github.com/Echo-Note/portunus/internal/config"
	"github.com/Echo-Note/portunus/internal/testutil"
)

// setupProjectService 创建测试用 ProjectService 实例。
func setupProjectService(t *testing.T) (*ProjectService, *UserService, context.Context) {
	t.Helper()
	ctx := context.Background()

	cfg := config.DatabaseConfig{
		URL:          "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable",
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}
	client, err := config.NewEntClient(ctx, cfg)
	require.NoError(t, err)

	// 清理
	testutil.CleanDB(t, client)

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

	t.Cleanup(func() { testutil.CloseClient(t, client) })
	return projectSvc, userSvc, ctx
}

// createTestUser 创建并激活测试用户。
func createTestUser(t *testing.T, svc *UserService, ctx context.Context, email string) uuid.UUID {
	t.Helper()
	result, err := svc.Register(ctx, RegisterInput{Email: email, Password: "password123"})
	require.NoError(t, err)
	_, err = svc.client.User.Update().Where(user.EmailEQ(email)).SetStatus(user.StatusActive).Save(ctx)
	require.NoError(t, err)
	return result.UserID
}

// TestProjectService_Create_Success 测试成功创建项目。
func TestProjectService_Create_Success(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "owner@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID:   "test-proj",
		Name:        "测试项目",
		Description: "描述",
		OwnerID:     ownerID,
	})
	require.NoError(t, err)
	assert.Equal(t, "test-proj", p.ProjectID)
	assert.Equal(t, "测试项目", p.Name)
	assert.Equal(t, project.StatusActive, p.Status)

	// 验证 owner 成员已创建
	members, err := svc.client.ProjectMember.Query().All(ctx)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "owner", string(members[0].Role))
}

// TestProjectService_Create_DuplicateProjectID 测试重复 project_id。
func TestProjectService_Create_DuplicateProjectID(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "owner2@test.com")

	_, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "dup-proj", Name: "First", OwnerID: ownerID,
	})
	require.NoError(t, err)

	_, err = svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "dup-proj", Name: "Second", OwnerID: ownerID,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicate)
}

// TestProjectService_GetProject 测试获取项目。
func TestProjectService_GetProject(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "owner3@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "get-proj", Name: "Get Project", OwnerID: ownerID,
	})
	require.NoError(t, err)

	got, err := svc.GetProject(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
}

// TestProjectService_GetProject_NotFound 测试获取不存在的项目。
func TestProjectService_GetProject_NotFound(t *testing.T) {
	svc, _, ctx := setupProjectService(t)

	_, err := svc.GetProject(ctx, uuid.Nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestProjectService_ListUserProjects 测试列出用户项目。
func TestProjectService_ListUserProjects(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "owner4@test.com")

	// 创建两个项目
	_, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "proj-a", Name: "Project A", OwnerID: ownerID,
	})
	require.NoError(t, err)
	_, err = svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "proj-b", Name: "Project B", OwnerID: ownerID,
	})
	require.NoError(t, err)

	projects, err := svc.ListUserProjects(ctx, ownerID)
	require.NoError(t, err)
	assert.Len(t, projects, 2)
}

// TestProjectService_ListUserProjects_Empty 测试无项目用户。
func TestProjectService_ListUserProjects_Empty(t *testing.T) {
	svc, _, ctx := setupProjectService(t)

	projects, err := svc.ListUserProjects(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, projects)
}

// TestProjectService_CheckQuota_Domains 测试域名配额检查。
func TestProjectService_CheckQuota_Domains(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "owner5@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "quota-proj", Name: "Quota", OwnerID: ownerID,
	})
	require.NoError(t, err)

	// 初始配额未超
	err = svc.CheckQuota(ctx, p.ID, "domains")
	assert.NoError(t, err)
}

// TestProjectService_CheckQuota_Members 测试成员配额检查。
func TestProjectService_CheckQuota_Members(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "owner6@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "quota2-proj", Name: "Quota2", OwnerID: ownerID,
	})
	require.NoError(t, err)

	err = svc.CheckQuota(ctx, p.ID, "members")
	assert.NoError(t, err)
}

// TestProjectService_Create_Validation 测试输入校验。
func TestProjectService_Create_Validation(t *testing.T) {
	svc, _, ctx := setupProjectService(t)

	tests := []struct {
		name  string
		input CreateProjectInput
	}{
		{"空 project_id", CreateProjectInput{ProjectID: "", Name: "Test", OwnerID: uuid.New()}},
		{"空名称", CreateProjectInput{ProjectID: "test", Name: "", OwnerID: uuid.New()}},
		{"空 owner", CreateProjectInput{ProjectID: "test", Name: "Test", OwnerID: uuid.Nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateProject(ctx, tt.input)
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}

// TestProjectService_Update_Success 测试成功更新项目。
func TestProjectService_Update_Success(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "update-owner@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "update-proj", Name: "原始名称", Description: "原始描述", OwnerID: ownerID,
	})
	require.NoError(t, err)

	updated, err := svc.UpdateProject(ctx, p.ID, UpdateProjectInput{
		Name:        "新名称",
		Description: "新描述",
	})
	require.NoError(t, err)
	assert.Equal(t, "新名称", updated.Name)
	assert.Equal(t, "新描述", updated.Description)
}

// TestProjectService_Update_NotFound 测试更新不存在的项目。
func TestProjectService_Update_NotFound(t *testing.T) {
	svc, _, ctx := setupProjectService(t)

	_, err := svc.UpdateProject(ctx, uuid.New(), UpdateProjectInput{Name: "test"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestProjectService_Delete 测试删除项目。
func TestProjectService_Delete(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "delete-owner@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "delete-proj", Name: "待删除", OwnerID: ownerID,
	})
	require.NoError(t, err)

	err = svc.DeleteProject(ctx, p.ID, ownerID)
	require.NoError(t, err)

	// 验证项目已标记为删除中
	got, err := svc.GetProject(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, project.StatusDeleting, got.Status)
}

// TestProjectService_Suspend 测试冻结项目。
func TestProjectService_Suspend(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "suspend-owner@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "suspend-proj", Name: "待冻结", OwnerID: ownerID,
	})
	require.NoError(t, err)

	err = svc.SuspendProject(ctx, p.ID, ownerID, "测试冻结")
	require.NoError(t, err)

	got, err := svc.GetProject(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, project.StatusSuspended, got.Status)
}

// TestProjectService_Reactivate 测试解冻项目。
func TestProjectService_Reactivate(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "reactivate-owner@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "reactivate-proj", Name: "待解冻", OwnerID: ownerID,
	})
	require.NoError(t, err)

	err = svc.SuspendProject(ctx, p.ID, ownerID, "测试冻结")
	require.NoError(t, err)

	err = svc.ReactivateProject(ctx, p.ID, ownerID)
	require.NoError(t, err)

	got, err := svc.GetProject(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, project.StatusActive, got.Status)
}

// TestProjectService_Suspend_InvalidTransition 测试从非活跃状态冻结。
func TestProjectService_Suspend_InvalidTransition(t *testing.T) {
	svc, userSvc, ctx := setupProjectService(t)
	ownerID := createTestUser(t, userSvc, ctx, "suspend-inv@test.com")

	p, err := svc.CreateProject(ctx, CreateProjectInput{
		ProjectID: "suspend-inv", Name: "冻结无效", OwnerID: ownerID,
	})
	require.NoError(t, err)

	// 先冻结
	err = svc.SuspendProject(ctx, p.ID, ownerID, "first")
	require.NoError(t, err)

	// 再次冻结应失败
	err = svc.SuspendProject(ctx, p.ID, ownerID, "second")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTransition)
}
