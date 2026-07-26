// Package integration 提供 Portunus 的端到端集成测试。
// 测试覆盖完整的 HTTP 请求 → 中间件 → Handler → Service → DB 链路。
//
// 运行方式：
//
//	make test-integration    # 本地运行（需要 Docker PostgreSQL + Redis）
//	go test -race -v -count=1 ./test/integration/...   # CI 命令
//
// 环境变量：
//
//	DATABASE_URL — PostgreSQL 连接串（默认 postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable）
//	REDIS_URL — Redis 连接串（默认 redis://localhost:6379/0）
//	JWT_PRIVATE_KEY_FILE — JWT 私钥文件路径（默认 ../../certs/jwt-private.pem）
//	JWT_PUBLIC_KEY_FILE — JWT 公钥文件路径（默认 ../../certs/jwt-public.pem）
//	SKIP_INTEGRATION — 设置任意值跳过所有集成测试
package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/internal/api"
	"github.com/Echo-Note/portunus/internal/api/handler"
	"github.com/Echo-Note/portunus/internal/config"
	"github.com/Echo-Note/portunus/internal/service"
)

// ── 包级共享状态（由 TestMain 初始化）──

var (
	testRouter    *gin.Engine
	testClient    *generated.Client
	testUserSvc   *service.UserService
	testMemberSvc *service.MemberService
	testProxySvc  *service.ProxyService
)

// TestMain 初始化集成测试环境。
// 1. 连接数据库并运行 Ent 自动迁移
// 2. 初始化所有 Service 和 Handler
// 3. 注册路由
// 4. 运行所有测试
// 5. 清理测试数据并关闭连接
func TestMain(m *testing.M) {
	// 跳过门控
	if os.Getenv("SKIP_INTEGRATION") != "" {
		log.Println("SKIP_INTEGRATION 已设置，跳过集成测试")
		os.Exit(0)
	}

	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	// ── 1. 连接数据库 ──
	dsn := getEnvOrDefault("DATABASE_URL", "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable")
	dbCfg := config.DatabaseConfig{
		URL:          dsn,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}
	client, err := config.NewEntClient(ctx, dbCfg)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	testClient = client

	// ── 2. 运行自动迁移（幂等，CI 已运行过则无操作）──
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("Schema 迁移失败: %v", err)
	}

	// ── 3. 加载 JWT 配置 ──
	jwtCfg := config.JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	}
	privateKeyFile := getEnvOrDefault("JWT_PRIVATE_KEY_FILE", "../../certs/jwt-private.pem")
	publicKeyFile := getEnvOrDefault("JWT_PUBLIC_KEY_FILE", "../../certs/jwt-public.pem")
	if data, err := os.ReadFile(privateKeyFile); err == nil {
		jwtCfg.PrivateKey = string(data)
	} else {
		log.Printf("警告: 无法读取 JWT 私钥文件 %s: %v", privateKeyFile, err)
	}
	if data, err := os.ReadFile(publicKeyFile); err == nil {
		jwtCfg.PublicKey = string(data)
	} else {
		log.Printf("警告: 无法读取 JWT 公钥文件 %s: %v", publicKeyFile, err)
	}

	// ── 4. 初始化 Service 层 ──
	userSvc, err := service.NewUserService(client, jwtCfg)
	if err != nil {
		log.Fatalf("初始化 UserService 失败: %v", err)
	}
	testUserSvc = userSvc

	stateMachine := service.NewStateMachine(client)
	caddyClient := service.NewNoopCaddyClient()
	projectSvc := service.NewProjectService(client, stateMachine)
	domainSvc := service.NewDomainService(client, stateMachine, caddyClient)
	proxySvc := service.NewProxyService(client, stateMachine, caddyClient)
	memberSvc := service.NewMemberService(client, stateMachine)
	shareSvc := service.NewShareService(client, stateMachine)
	auditSvc := service.NewAuditService(client)
	apiTokenSvc := service.NewApiTokenService(client)
	snapshotSvc := service.NewSnapshotService(client)

	testMemberSvc = memberSvc
	testProxySvc = proxySvc

	// ── 5. 初始化 Handler 层 ──
	authH := handler.NewAuthHandler(userSvc)
	projectH := handler.NewProjectHandler(projectSvc)
	domainH := handler.NewDomainHandler(domainSvc)
	proxyH := handler.NewProxyHandler(proxySvc, domainSvc)
	memberH := handler.NewMemberHandler(memberSvc)
	shareH := handler.NewShareHandler(shareSvc)
	auditH := handler.NewAuditHandler(auditSvc)
	snapshotH := handler.NewSnapshotHandler(snapshotSvc)

	// ── 6. 注册路由 ──
	router := gin.New()
	api.RegisterRoutes(
		router,
		authH, projectH, domainH, proxyH, memberH, shareH, auditH, snapshotH,
		userSvc, memberSvc, apiTokenSvc,
		6000, 1000, // 高限流值用于测试
	)
	testRouter = router

	// ── 7. 运行测试 ──
	code := m.Run()

	// ── 8. 清理 ──
	cleanAllTables(ctx, client)
	if err := client.Close(); err != nil {
		log.Printf("关闭数据库连接失败: %v", err)
	}

	os.Exit(code)
}

// cleanAllTables 按外键依赖顺序清空所有表。
// 与 testutil.CleanDB 逻辑一致，但不依赖 *testing.T（用于 TestMain 阶段）。
func cleanAllTables(ctx context.Context, client *generated.Client) {
	// 按外键依赖顺序删除
	client.CaddyIDMapping.Delete().Exec(ctx)
	client.Upstream.Delete().Exec(ctx)
	client.ProxyConfig.Delete().Exec(ctx)
	client.DomainShare.Delete().Exec(ctx)
	client.Domain.Delete().Exec(ctx)
	client.ProjectAuditLog.Delete().Exec(ctx)
	client.Invitation.Delete().Exec(ctx)
	client.ProjectMember.Delete().Exec(ctx)
	client.ApiToken.Delete().Exec(ctx)
	client.ConfigSnapshot.Delete().Exec(ctx)
	client.Project.Delete().Exec(ctx)
	client.User.Delete().Exec(ctx)
}

// getEnvOrDefault 获取环境变量或返回默认值。
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}