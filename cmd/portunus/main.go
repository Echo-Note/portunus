// Portunus 控制面入口。
// 启动 HTTP Server，初始化数据库、Caddy 客户端、Service 层，
// 支持优雅关闭（graceful shutdown）。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nrgao/portunus/internal/api"
	"github.com/nrgao/portunus/internal/api/handler"
	"github.com/nrgao/portunus/internal/config"
	"github.com/nrgao/portunus/internal/service"
)

func main() {
	// ── 1. 初始化日志 ──
	config.InitLogger()
	slog.Info("Portunus 控制面启动中...")

	// ── 2. 加载配置 ──
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "err", err)
		os.Exit(1)
	}

	// ── 3. 初始化数据库连接 ──
	ctx := context.Background()
	entClient, err := config.NewEntClient(ctx, cfg.Database)
	if err != nil {
		slog.Error("数据库连接失败", "err", err)
		os.Exit(1)
	}
	defer entClient.Close()

	// ── 4. 初始化 Caddy 客户端（可选，开发环境可能没有 Caddy）──
	// 注意：如果 Caddy 不可达，服务仍可启动，仅 Caddy 相关操作会失败
	// caddyClient, err := caddy.New(cfg.Caddy)
	// if err != nil {
	// 	slog.Warn("Caddy 客户端初始化失败，相关功能不可用", "err", err)
	// }

	// ── 5. 初始化 Service 层 ──
	stateMachine := service.NewStateMachine(entClient)

	userSvc, err := service.NewUserService(entClient, cfg.JWT)
	if err != nil {
		slog.Error("初始化 UserService 失败", "err", err)
		os.Exit(1)
	}

	projectSvc := service.NewProjectService(entClient, stateMachine)

	// 开发环境使用不验证 TLS 的 Caddy 客户端
	// 生产环境应使用 caddy.New(cfg.Caddy) 加载 mTLS 证书
	caddyClient := service.NewNoopCaddyClient()
	domainSvc := service.NewDomainService(entClient, stateMachine, caddyClient)
	proxySvc := service.NewProxyService(entClient, stateMachine, caddyClient)
	memberSvc := service.NewMemberService(entClient, stateMachine)
	shareSvc := service.NewShareService(entClient, stateMachine)
	auditSvc := service.NewAuditService(entClient)

	// ── 6. 初始化 Handler 层 ──
	authH := handler.NewAuthHandler(userSvc)
	projectH := handler.NewProjectHandler(projectSvc)
	domainH := handler.NewDomainHandler(domainSvc)
	proxyH := handler.NewProxyHandler(proxySvc)
	memberH := handler.NewMemberHandler(memberSvc)
	shareH := handler.NewShareHandler(shareSvc)
	auditH := handler.NewAuditHandler(auditSvc)

	// ── 7. 设置 Gin 路由 ──
	gin.SetMode(cfg.Server.GinMode)
	router := gin.New()

	api.RegisterRoutes(
		router,
		authH, projectH, domainH, proxyH, memberH, shareH, auditH,
		userSvc, memberSvc,
		cfg.RateLimit.RPM, cfg.RateLimit.Burst,
	)

	// ── 8. 启动 HTTP Server ──
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// 在 goroutine 中启动服务，主线程等待信号
	go func() {
		slog.Info("HTTP Server 启动", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP Server 错误", "err", err)
			os.Exit(1)
		}
	}()

	// ── 9. 优雅关闭 ──
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("收到关闭信号，开始优雅关闭...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭 HTTP Server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP Server 关闭失败", "err", err)
	}

	// 关闭数据库连接
	if err := entClient.Close(); err != nil {
		slog.Error("数据库连接关闭失败", "err", err)
	}

	slog.Info("Portunus 已安全关闭")
}