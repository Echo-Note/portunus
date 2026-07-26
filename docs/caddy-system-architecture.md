# 前后端系统架构文档

> **版本**: v1.0  
> **日期**: 2026-07-24  
> **关联文档**:  
> - `caddy-multi-tenant-architecture.md` v1.3 — 系统架构与技术设计  
> - `caddy-tech-stack-selection.md` v1.1 — 技术栈与框架选型  
> - ~~`caddy-architecture-review.md`~~ — 待创建
> **状态**: 架构评审中

---

## 目录

- [1. 文档概述](#1-文档概述)
  - [1.1 文档定位](#11-文档定位)
  - [1.2 系统全景](#12-系统全景)
  - [1.3 技术栈速查](#13-技术栈速查)
- [2. 后端系统架构](#2-后端系统架构)
  - [2.1 分层架构总览](#21-分层架构总览)
  - [2.2 接入层：Gin HTTP Server](#22-接入层gin-http-server)
  - [2.3 MCP 服务层](#23-mcp-服务层)
  - [2.4 AI 智能助手层](#24-ai-智能助手层)
  - [2.5 业务逻辑层](#25-业务逻辑层)
  - [2.6 数据访问层](#26-数据访问层)
  - [2.7 Caddy 客户端层](#27-caddy-客户端层)
  - [2.8 认证与安全层](#28-认证与安全层)
  - [2.9 可观测性层](#29-可观测性层)
  - [2.10 后端启动与初始化流程](#210-后端启动与初始化流程)
- [3. 前端系统架构](#3-前端系统架构)
  - [3.1 前端分层架构](#31-前端分层架构)
  - [3.2 页面结构与路由设计](#32-页面结构与路由设计)
  - [3.3 状态管理设计](#33-状态管理设计)
  - [3.4 数据请求与缓存策略](#34-数据请求与缓存策略)
  - [3.5 API 客户端生成](#35-api-客户端生成)
  - [3.6 前端安全策略](#36-前端安全策略)
  - [3.7 实时数据推送](#37-实时数据推送)
- [4. 前后端交互协议](#4-前后端交互协议)
  - [4.1 REST API 规范](#41-rest-api-规范)
  - [4.2 统一响应格式](#42-统一响应格式)
  - [4.3 错误码体系](#43-错误码体系)
  - [4.4 分页与过滤](#44-分页与过滤)
  - [4.5 认证流程](#45-认证流程)
- [5. 部署架构](#5-部署架构)
  - [5.1 MVP 部署拓扑](#51-mvp-部署拓扑)
  - [5.2 生产部署拓扑](#52-生产部署拓扑)
  - [5.3 环境矩阵](#53-环境矩阵)
- [6. 横切关注点](#6-横切关注点)
  - [6.1 配置管理](#61-配置管理)
  - [6.2 日志规范](#62-日志规范)
  - [6.3 链路追踪](#63-链路追踪)
  - [6.4 优雅关闭](#64-优雅关闭)

---

## 1. 文档概述

### 1.1 文档定位

本文档是前后端工程实施的**落地指导书**，在架构设计文档（定义"做什么"）和技术栈选型文档（定义"用什么"）的基础上，回答"怎么搭"的问题：

| 维度 | 关注问题 | 对应文档 |
|---|---|---|
| **业务架构** | 系统有哪些功能模块？多租户如何隔离？ | 架构设计文档 §1-§5 |
| **数据模型** | 数据库表怎么设计？配置怎么翻译？ | 架构设计文档 §2-§3 |
| **技术选型** | 用什么语言、框架、中间件？ | 技术栈选型文档 §1-§12 |
| **后端架构** | 代码怎么分层？请求怎么流转？中间件怎么编排？ | **本文档 §2** |
| **前端架构** | 页面怎么组织？状态怎么管理？API 怎么调用？ | **本文档 §3** |
| **交互协议** | 前后端怎么通信？错误码怎么定义？ | **本文档 §4** |
| **部署架构** | 服务怎么部署？环境怎么区分？ | **本文档 §5** |

### 1.2 系统全景

```
╔══════════════════════════════════════════════════════════════════════════╗
║                    基于 Caddy 的分布式权限管理系统                           ║
╠══════════════════════════════════════════════════════════════════════════╣
║                                                                          ║
║  ┌─── 用户层 ───────────────────────────────────────────────────────┐   ║
║  │  浏览器 (Web UI)          AI 助手 (Claude/ChatGPT)   CLI 工具     │   ║
║  │  React 18 SPA             MCP Client               HTTP Client   │   ║
║  └────────┬─────────────────────┬────────────────────────┬──────────┘   ║
║           │ HTTPS + JWT          │ MCP (HTTP+SSE)         │ Bearer Token  ║
║           ▼                     ▼                        ▼              ║
║  ┌─── 接入层 ────────────────────────────────────────────────────────┐   ║
║  │  Gin HTTP Server (:8080)    MCP Server (:8081)     Eino Agent    │   ║
║  │  REST API                   JSON-RPC 2.0            内置 AI 助手   │   ║
║  └─────────────────────────┬────────────────────────────────────────┘   ║
║                            │ 共享 Service Layer                         ║
║  ┌─── 控制面 (Go 单二进制) ─┴────────────────────────────────────────┐   ║
║  │  Middleware: Auth → RBAC → Ownership → RateLimit → Audit        │   ║
║  │  Service:    Project → Domain → Proxy → Member → Share → Audit  │   ║
║  │  Repository: Ent 生成 (类型安全)                                  │   ║
║  │  Caddy Client: mTLS → Admin API (:2021)                          │   ║
║  └──────────────────────────────────┬────────────────────────────────┘   ║
║                                     │                                     ║
║  ┌─── 数据面 ───────────────────────▼──────────────────────────────┐   ║
║  │  Caddy v2.11.0+ (Admin API + JSON Config Tree + @id 机制)      │   ║
║  └──────────────────────────────────────────────────────────────────┘   ║
║                                     │                                     ║
║  ┌─── 存储层 ───────────────────────▼──────────────────────────────┐   ║
║  │  PostgreSQL 16+        Redis 7.x         MinIO / S3              │   ║
║  │  (业务数据 + 快照)      (缓存+锁+会话)    (配置备份)               │   ║
║  └──────────────────────────────────────────────────────────────────┘   ║
║                                                                          ║
║  ┌─── 可观测性 ──────────────────────────────────────────────────────┐   ║
║  │  Prometheus → Grafana    OpenTelemetry → Jaeger    slog → Loki   │   ║
║  └──────────────────────────────────────────────────────────────────┘   ║
╚══════════════════════════════════════════════════════════════════════════╝
```

### 1.3 技术栈速查

| 层次 | 技术 | 版本 |
|---|---|---|
| 前端 | React + TypeScript + Vite + Ant Design | 18 / 5.x / 5.x / 5.x |
| 后端语言 | Go | 1.24+ |
| Web 框架 | Gin | v1.12+ |
| MCP 框架 | go-sdk | v1.6.1+ |
| AI 框架 | Eino (CloudWeGo) | 最新 |
| 数据访问 | Ent (entgo.io) | 最新 |
| 数据库 | PostgreSQL | 16+ |
| 缓存 | Redis | 7.x |
| 认证 | JWT (RS256) + Argon2id + TOTP + OAuth2 | — |
| 数据面 | Caddy | v2.11.0+ |
| 可观测性 | Prometheus + Grafana + OpenTelemetry + slog | — |
| 容器 | Docker (Alpine) | — |
| 编排 | Docker Compose → Kubernetes | — |

---

## 2. 后端系统架构

### 2.1 分层架构总览

后端采用**洋葱架构（Onion Architecture）**的变体，从外到内依次为：

```
                    请求入站
                       │
    ┌──────────────────▼──────────────────┐
    │         接入层 (Delivery)           │  ← Gin / MCP Server / Eino Agent
    │  负责协议解析、请求路由、响应序列化    │
    └──────────────────┬──────────────────┘
                       │
    ┌──────────────────▼──────────────────┐
    │         中间件层 (Middleware)        │  ← Auth / RBAC / Ownership / RateLimit / Audit
    │  负责横切关注点：认证、授权、限流、审计 │
    └──────────────────┬──────────────────┘
                       │
    ┌──────────────────▼──────────────────┐
    │         业务逻辑层 (Service)        │  ← ProjectService / DomainService / ...
    │  负责核心业务规则、配置翻译、校验编排   │  ← 框架无关，不依赖 Gin
    └──────┬───────────┬──────────────────┘
           │           │
    ┌──────▼──────┐ ┌──▼─────────────────┐
    │  数据访问层   │ │  Caddy 客户端层     │  ← Repository (Ent)
    │  (Repository)│ │  (Caddy Client)    │  ← mTLS HTTP Client
    │  PostgreSQL  │ │  Caddy Admin API   │
    └─────────────┘ └────────────────────┘
           │                   │
    ┌──────▼───────────────────▼──────────┐
    │         基础设施层 (Infrastructure)  │  ← DB / Redis / MinIO / Prometheus
    └─────────────────────────────────────┘
```

**分层规则**：

| 规则 | 说明 |
|---|---|
| 依赖方向 | 外层依赖内层，内层不依赖外层 |
| Service 层框架无关 | 不 import `gin`，只接收 `context.Context` 和普通参数 |
| Repository 层接口化 | Service 依赖 Repository 接口，不依赖具体实现 |
| Handler 层薄 | 负责参数绑定、上下文提取、错误映射和响应序列化，不包含核心业务规则 |

### 2.2 接入层：Gin HTTP Server

#### 2.2.1 路由注册

```go
// internal/api/router.go
package api

func NewRouter(h *Handlers, mw *Middleware) *gin.Engine {
    r := gin.New()
    
    // 全局中间件
    r.Use(gin.Recovery())
    r.Use(mw.RequestID())        // 生成 X-Request-ID
    r.Use(mw.Logging())          // 结构化日志
    r.Use(mw.Metrics())          // Prometheus 指标
    r.Use(mw.Tracing())         // OpenTelemetry 追踪
    r.Use(mw.CORS())            // 跨域
    r.Use(mw.RateLimit())       // 全局速率限制
    
    // 健康检查（无需认证）
    r.GET("/health", h.Health.Check)
    r.GET("/ready", h.Health.Ready)
    
    // API v1
    v1 := r.Group("/api/v1")
    {
        // 认证
        auth := v1.Group("/auth")
        {
            auth.POST("/register", h.Auth.Register)
            auth.POST("/login", h.Auth.Login)
            auth.POST("/refresh", h.Auth.Refresh)
            auth.POST("/logout", h.Auth.Logout)
            auth.GET("/oauth/:provider", h.Auth.OAuthRedirect)
            auth.GET("/oauth/:provider/callback", h.Auth.OAuthCallback)
        }
        
        // 需认证的 API
        authed := v1.Group("")
        authed.Use(mw.Auth())           // JWT 验证
        {
            // 项目管理
            projects := authed.Group("/projects")
            projects.Use(mw.ProjectContext())  // 注入 project_id 上下文
            {
                projects.GET("", h.Project.List)
                projects.POST("", mw.RequireRole("owner", "admin"), h.Project.Create)
                projects.GET("/:id", h.Project.Get)
                projects.PATCH("/:id", mw.RequireRole("owner"), h.Project.Update)
                projects.DELETE("/:id", mw.RequireRole("owner"), h.Project.Delete)
                
                // 项目成员
                projects.GET("/:id/members", h.Member.List)
                projects.POST("/:id/members", mw.RequireRole("owner", "admin"), h.Member.Invite)
                projects.PATCH("/:id/members/:uid", mw.RequireRole("owner", "admin"), h.Member.UpdateRole)
                projects.DELETE("/:id/members/:uid", mw.RequireRole("owner", "admin"), h.Member.Remove)
                
                // 项目审计
                projects.GET("/:id/audit-logs", mw.RequireRole("owner", "admin"), h.Audit.Query)
            }
            
            // 域名管理
            domains := authed.Group("/projects/:id/domains")
            domains.Use(mw.ProjectContext())  // 注入 project_id 上下文
            {
                domains.GET("", h.Domain.List)
                domains.POST("", mw.RequireRole("owner", "admin", "editor"), h.Domain.Create)
                domains.GET("/:did", h.Domain.Get)
                domains.PATCH("/:did", mw.RequireRole("owner", "admin", "editor"), h.Domain.Update)
                domains.DELETE("/:did", mw.RequireRole("owner", "admin"), h.Domain.Delete)
                
                // 反向代理
                domains.GET("/:did/proxy", h.Proxy.Get)
                domains.PATCH("/:did/proxy", mw.RequireRole("owner", "admin", "editor"), h.Proxy.Update)
                domains.POST("/:did/proxy/upstreams", mw.RequireRole("owner", "admin", "editor"), h.Proxy.AddUpstream)
                domains.DELETE("/:did/proxy/upstreams/:uid", mw.RequireRole("owner", "admin"), h.Proxy.RemoveUpstream)
                
                // 上游健康状态
                domains.GET("/:did/status", h.Proxy.Status)
                
                // 共享
                domains.POST("/:did/shares", mw.RequireRole("owner", "admin"), h.Share.Create)
                domains.GET("/:did/shares", h.Share.List)
            }
            
            // 共享管理
            authed.DELETE("/shares/:sid", mw.RequireRole("owner", "admin"), h.Share.Revoke)
            
            // 用户管理
            authed.GET("/me", h.User.Profile)
            authed.PATCH("/me", h.User.Update)
            authed.POST("/me/tokens", h.User.CreateToken)
            authed.GET("/me/tokens", h.User.ListTokens)
            authed.DELETE("/me/tokens/:tid", h.User.RevokeToken)
        }
    }
    
    return r
}
```

#### 2.2.2 Handler 层规范

```go
// internal/api/handler/domain.go
package handler

type DomainHandler struct {
    svc *service.DomainService
}

// Create 创建域名
// POST /api/v1/projects/:id/domains
func (h *DomainHandler) Create(c *gin.Context) {
    // 1. 绑定参数
    var req dto.CreateDomainRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, dto.Error(ErrInvalidRequest, err.Error()))
        return
    }
    
    // 2. 从上下文提取身份信息（由中间件注入）
    ctx := c.Request.Context()
    projectID := middleware.GetProjectID(c)   // 从 URL path
    userID := middleware.GetUserID(c)          // 从 JWT
    role := middleware.GetRole(c)              // 从 JWT
    
    if projectID == "" {
        c.JSON(400, dto.Error(ErrInvalidRequest, "project_id is required"))
        return
    }
    
    // 3. 调用 Service（传标准库 context，不传 gin.Context）
    domain, err := h.svc.Create(ctx, &service.CreateDomainInput{
        ProjectID:  projectID,
        UserID:     userID,
        Role:       role,
        Domain:     req.Domain,
        Upstreams:  req.Upstreams,
        LbPolicy:   req.LbPolicy,
        SSLEnabled: req.SSLEnabled,
    })
    if err != nil {
        handleError(c, err)  // 统一错误处理
        return
    }
    
    // 4. 序列化响应
    c.JSON(201, dto.Success(domain))
}
```

### 2.3 MCP 服务层

#### 2.3.1 MCP Server 初始化

```go
// internal/mcp/server.go
package mcp

func NewMCPServer(svc *service.Services, addr string) *MCPServer {
    // 使用 Go MCP SDK 创建 Server
    s := server.NewMCPServer("caddy-mgmt", "1.0.0")
    
    // 注册所有工具
    registerCaddyTools(s, svc)       // 7 个 Caddy 工具
    registerProjectTools(s, svc)     // 4 个项目工具
    registerMemberTools(s, svc)      // 2 个成员工具
    registerShareTool(s, svc)       // 1 个共享工具
    
    return &MCPServer{server: s, addr: addr}
}

func (m *MCPServer) Start(ctx context.Context) error {
    // SSE 传输（远程 AI Agent 访问）
    sseServer := server.NewSSEServer(m.server, server.WithBaseURL(m.addr))
    return sseServer.Start(m.addr)
}
```

#### 2.3.2 MCP 工具与 Service 层共享

```go
// internal/mcp/tools/caddy_domain.go
package tools

func registerCaddyTools(s *server.MCPServer, svc *service.Services) {
    // caddy_domain_create 工具
    s.AddTool(mcp.NewTool("caddy_domain_create",
        mcp.WithDescription("为当前项目创建新的域名路由..."),
        mcp.WithString("domain", mcp.Required(), mcp.Description("域名")),
        mcp.WithArray("upstreams", mcp.Required(), mcp.Description("上游列表")),
        // ... 其他参数
    ), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // 1. 从 MCP 会话上下文提取身份
        session := mcpSessionFromContext(ctx)  // 包含 user_id, project_id, role
        
        // 2. 调用同一个 Service（与 REST API 共享）
        domain, err := svc.Domain.Create(ctx, &service.CreateDomainInput{
            ProjectID: session.ProjectID,
            UserID:    session.UserID,
            Role:      session.Role,
            Domain:    req.Params.Arguments["domain"].(string),
            // ... 映射其他参数
        })
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }
        
        // 3. 返回结构化结果
        return mcp.NewToolResultText(JSON(domain)), nil
    })
}
```

> **认证机制**：MCP 会话通过 API Token 或 JWT 建立。客户端连接 MCP Server 时，在 `InitializeRequest` 或 HTTP 头中携带凭证；Server 校验后将 `user_id`、`project_id`、`role` 写入会话上下文。工具调用时从上下文提取这些信息，再调用共享 Service，从而与 REST API 保持一致的权限校验和审计。

> **关键设计**：MCP 工具和 REST Handler 调用**同一个 Service 方法**，确保权限校验、审计记录、业务逻辑完全一致。

### 2.4 AI 智能助手层

#### 2.4.1 Eino Agent 架构

```go
// internal/agent/agent.go
package agent

type Assistant struct {
    reactAgent *adk.ChatModelAgent
    runner     *adk.Runner
}

func NewAssistant(
    llmProvider string,       // "openai" / "claude" / "gemini"
    apiKey     string,
    mcpServerURL string,      // 本系统 MCP Server 地址
) (*Assistant, error) {
    ctx := context.Background()
    
    // 1. 创建 LLM 客户端
    chatModel, err := createLLM(llmProvider, apiKey)
    if err != nil {
        return nil, err
    }
    
    // 2. 连接到本系统 MCP Server（作为 MCP Client）
    mcpClient, err := client.NewSSEMCPClient(mcpServerURL)
    if err != nil {
        return nil, err
    }
    mcpClient.Start(ctx)
    mcpClient.Initialize(ctx, mcp.InitializeRequest{
        Params: mcp.InitializeParams{
            ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
            ClientInfo: mcp.Implementation{Name: "eino-agent", Version: "1.0.0"},
        },
    })
    
    // 3. 自动发现所有 MCP 工具（14 个）
    tools, err := mcpext.GetTools(ctx, &mcpext.Config{Cli: mcpClient})
    if err != nil {
        return nil, err
    }
    
    // 4. 创建 ReAct Agent
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model: chatModel,
        ToolsConfig: adk.ToolsNodeConfig{
            Tools: tools,    // 注入 14 个 MCP 工具
        },
        Middleware: []adk.Middleware{
            confirmMiddleware(),   // Human-in-the-loop 确认
            auditMiddleware(),     // AI 操作审计
        },
    })
    
    // 5. 创建 Runner
    runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
    
    return &Assistant{reactAgent: agent, runner: runner}, nil
}

// Query 处理用户自然语言请求，返回 Eino AgentEventIterator
func (a *Assistant) Query(ctx context.Context, userInput string) (adk.AgentEventIterator, error) {
    // Agent 自主决策：理解意图 → 选择工具 → 调用 → 返回结果
    return a.runner.Query(ctx, userInput)
}

// StreamText 消费 AgentEventIterator 并将文本事件输出到 channel（示例）
func (a *Assistant) StreamText(ctx context.Context, iter adk.AgentEventIterator, out chan<- string) error {
    defer close(out)
    for iter.Next(ctx) {
        event := iter.Cur()
        if event.Type == adk.AgentEventType_STREAM {
            out <- event.Content
        }
    }
    return iter.Err()
}
```

#### 2.4.2 Human-in-the-loop 中间件

```go
// internal/agent/middleware/confirm.go
package middleware

func confirmMiddleware() adk.Middleware {
    return func(next adk.AgentFunc) adk.AgentFunc {
        return func(ctx context.Context, input *adk.AgentInput) (*adk.AgentOutput, error) {
            // 检测 Agent 是否要调用写操作工具
            if isWriteOperation(input.ToolCall) {
                // 推送确认请求到前端（通过 WebSocket）
                confirmed := waitForUserConfirmation(ctx, input.ToolCall)
                if !confirmed {
                    return &adk.AgentOutput{
                        Content: "用户取消了操作",
                    }, nil
                }
            }
            return next(ctx, input)
        }
    }
}
```

### 2.5 业务逻辑层

#### 2.5.1 Service 接口定义

```go
// internal/service/domain_service.go
package service

type DomainService struct {
    client    *ent.Client
    caddy     *caddy.Client
    audit     *AuditService
    validator *Validator
}

// NewDomainService 通过构造函数注入 *ent.Client 和其他依赖
func NewDomainService(
    client *ent.Client,
    caddy *caddy.Client,
    audit *AuditService,
    validator *Validator,
) *DomainService {
    return &DomainService{
        client:    client,
        caddy:     caddy,
        audit:     audit,
        validator: validator,
    }
}

> **依赖倒置**：Service 层依赖 `*ent.Client`（Ent 生成的类型安全客户端），通过构造函数注入，便于单元测试和实现替换。

type CreateDomainInput struct {
    ProjectID  string
    UserID     string
    Role       string
    Domain     string
    Upstreams  []UpstreamInput
    LbPolicy   string
    SSLEnabled bool
}

type CreateDomainOutput struct {
    DomainID   string
    RouteID    string    // Caddy @id
    ProxyID    string    // Caddy @id
    Status     string
}

func (s *DomainService) Create(ctx context.Context, in *CreateDomainInput) (*CreateDomainOutput, error) {
    // 1. 业务校验
    if err := s.validator.ValidateDomain(in); err != nil {
        return nil, err
    }
    
    // 2. 配额检查
    if err := s.checkQuota(ctx, in.ProjectID); err != nil {
        return nil, err
    }
    
    // 3. 端口白名单校验
    if err := s.checkPortWhitelist(ctx, in.ProjectID, in.Upstreams); err != nil {
        return nil, err
    }
    
    // 4. 生成 @id
    routeID := caddy.GenerateRouteID(in.ProjectID, in.Domain)
    proxyID := caddy.GenerateProxyID(in.ProjectID, in.Domain)
    
    // 5. 翻译为 Caddy JSON
    caddyConfig := caddy.TranslateDomain(routeID, proxyID, in)
    
    // 6. JSON Schema 校验
    if err := s.validator.ValidateCaddyJSON(caddyConfig); err != nil {
        return nil, err
    }
    
    // 7. 获取 Etag（乐观锁）
    etag, err := s.caddy.GetEtag(ctx, "/config/apps/http/servers/main/routes")
    if err != nil {
        return nil, err
    }
    
    // 8. 下发到 Caddy（含 If-Match 头）
    if err := s.caddy.CreateRoute(ctx, caddyConfig, etag); err != nil {
        return nil, err
    }
    
    // 9. 存储数据库映射
    domain, err := s.repo.CreateDomain(ctx, &repository.CreateDomainParams{
        ProjectID:  in.ProjectID,
        DomainName: in.Domain,
        CaddyID:    routeID,
        RouteID:    routeID,
        SSLEnabled: in.SSLEnabled,
    })
    if err != nil {
        // 数据库失败 → 回滚 Caddy 配置
        s.caddy.DeleteRoute(ctx, routeID)
        return nil, err
    }
    
    // 10. 存储上游
    for _, up := range in.Upstreams {
        s.repo.CreateUpstream(ctx, domain.ID, up)
    }
    
    // 11. 配置快照
    s.saveConfigSnapshot(ctx, in.ProjectID)
    
    // 12. 审计日志
    s.audit.Log(ctx, &AuditEntry{
        ProjectID:    in.ProjectID,
        ActorID:      in.UserID,
        Action:       "domain.create",
        ResourceType: "domain",
        ResourceID:   domain.ID,
        Via:          "web_ui",  // 或 "mcp_tool"
        Result:       "success",
    })
    
    return &CreateDomainOutput{
        DomainID: domain.ID,
        RouteID:  routeID,
        ProxyID:  proxyID,
        Status:   "active",
    }, nil
}
```

> **关键设计**：Service 方法接收 `context.Context` 和普通结构体参数，不依赖任何 Web 框架类型。同一个方法可被 REST Handler、MCP Tool、Eino Agent 三方调用。

### 2.6 数据访问层

#### 2.6.1 Ent 生成的类型安全代码

数据访问层基于 [Ent](https://entgo.io)（entgo.io）实现。Ent 采用代码优先（Schema as Code）的方式，在 `ent/schema/*.go` 中以 Go 代码定义数据模型，自动生成类型安全的链式查询 API，无需手写 SQL 查询文件。

```go
// ent/schema/domain.go (schema 定义)
func (Domain) Fields() []ent.Field {
    return []ent.Field{
        field.String("domain_name").Unique(),
        field.String("project_id"),
        field.String("caddy_id"),
        field.String("route_id"),
        field.Bool("ssl_enabled").Default(false),
        field.Enum("status").Values("creating", "active", "updating", "disabled", "deleting", "deleted", "error"),
        field.Time("created_at").Default(time.Now),
    }
}
```

```go
// Ent 自动生成的客户端（类型安全的链式查询 API）
package repository

type DomainQueries interface {
    // 域名
    GetDomainByName(ctx context.Context, domainName string) (*ent.Domain, error)
    ListDomainsByProject(ctx context.Context, projectID string) ([]*ent.Domain, error)
    CreateDomain(ctx context.Context, params CreateDomainParams) (*ent.Domain, error)

    // @id 映射
    GetCaddyIDMapping(ctx context.Context, caddyID string) (*ent.CaddyIDMapping, error)
    CreateCaddyIDMapping(ctx context.Context, params CreateCaddyIDMappingParams) error

    // 审计日志
    CreateProjectAuditLog(ctx context.Context, params CreateProjectAuditLogParams) error
    ListProjectAuditLogs(ctx context.Context, params ListProjectAuditLogsParams) ([]*ent.ProjectAuditLog, error)

    // 配置快照
    CreateConfigSnapshot(ctx context.Context, params CreateConfigSnapshotParams) error
    GetLatestConfigSnapshot(ctx context.Context, projectID string) (*ent.ConfigSnapshot, error)
}

// 链式查询示例
func (r *domainRepo) GetDomainByName(ctx context.Context, name string) (*ent.Domain, error) {
    return r.client.Domain.Query().
        Where(domain.DomainName(name)).
        Only(ctx)
}

func (r *domainRepo) ListDomainsByProject(ctx context.Context, projectID string) ([]*ent.Domain, error) {
    return r.client.Domain.Query().
        Where(domain.And(
            domain.ProjectIDEQ(projectID),
            domain.StatusEQ(domain.StatusActive),
        )).
        Order(ent.Desc(domain.FieldCreatedAt)).
        All(ctx)
}
```

> **复杂查询回退原生 SQL**：对于无法用 Ent 表达式表达的复杂查询（跨表聚合、原生 SQL 函数等），可通过 `client.QueryContext(ctx, rawSQL)` 直接执行原生 SQL，并将结果扫描到目标结构体。

```go
// 复杂聚合查询回退原生 SQL
rows, err := client.QueryContext(ctx, `
    SELECT project_id, COUNT(*) AS domain_count
    FROM domains WHERE status = 'active'
    GROUP BY project_id
`)
```

> **迁移管理**：schema 变更后，通过 Ent 集成的 Atlas 自动生成迁移文件并执行，无需手写 SQL DDL，保证 schema 与迁移文件的一致性。

### 2.7 Caddy 客户端层

```go
// internal/caddy/client.go
package caddy

type Client struct {
    httpClient *http.Client    // mTLS 配置
    baseURL    string          // https://caddy.internal:2021
}

func NewClient(certFile, keyFile, caFile, baseURL string) (*Client, error) {
    // 加载 mTLS 证书
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("load x509 key pair: %w", err)
    }

    caCert, err := os.ReadFile(caFile)
    if err != nil {
        return nil, fmt.Errorf("read CA cert: %w", err)
    }

    caPool := x509.NewCertPool()
    if !caPool.AppendCertsFromPEM(caCert) {
        return nil, fmt.Errorf("append CA certs from PEM failed")
    }

    // 从 baseURL 提取 ServerName 用于 TLS 校验
    u, err := url.Parse(baseURL)
    if err != nil {
        return nil, fmt.Errorf("parse base URL: %w", err)
    }

    transport := &http.Transport{
        TLSClientConfig: &tls.Config{
            Certificates: []tls.Certificate{cert},
            RootCAs:      caPool,
            ServerName:   u.Hostname(),
            MinVersion:   tls.VersionTLS13,
        },
    }
    
    return &Client{
        httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second},
        baseURL:    baseURL,
    }, nil
}

// CreateRoute 通过 @id 机制创建路由
func (c *Client) CreateRoute(ctx context.Context, config map[string]interface{}, etag string) error {
    body, err := json.Marshal(config)
    if err != nil {
        return fmt.Errorf("marshal caddy config: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, "POST",
        c.baseURL+"/config/apps/http/servers/main/routes",
        bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    if etag != "" {
        req.Header.Set("If-Match", etag)  // 乐观锁
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 412 {
        return ErrConcurrentModification  // 需要重试
    }
    if resp.StatusCode != 200 {
        return fmt.Errorf("caddy error: %s", resp.Status)
    }
    return nil
}
```

### 2.8 认证与安全层

#### 2.8.1 中间件链

```go
// 请求处理流水线
Request → Recovery → RequestID → Logging → CORS → Tracing → Metrics
       → Auth(JWT) → ProjectContext → RateLimit(per user)
       → OwnershipCheck → Audit(post)
       → Handler → Response
```

#### 2.8.2 各中间件职责

| 中间件 | 职责 | 失败行为 |
|---|---|---|
| `Recovery` | 捕获 panic | 500 |
| `RequestID` | 生成唯一请求 ID | — |
| `Logging` | 结构化请求日志 | — |
| `Metrics` | Prometheus 计数 | — |
| `Tracing` | OpenTelemetry span | — |
| `CORS` | 跨域头 | — |
| `RateLimit` | 速率限制 (Redis) | 429 |
| `Auth` | JWT 验证 | 401 |
| `ProjectContext` | 从 URL/path 提取 project_id | 400 |
| `RequireRole` | 角色校验 | 403 |
| `OwnershipCheck` | @id 归属校验 | 403 |
| `Audit` | 审计记录 | — |

### 2.9 可观测性层

```go
// internal/observability/logger.go
func NewLogger() *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level:     slog.LevelInfo,
        AddSource: true,
    }))
}

// internal/observability/metrics.go
var (
    ConfigUpdates = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "caddy_config_updates_total",
            Help: "Total number of Caddy config updates.",
        },
        []string{"plan", "tier", "action", "result"},
    )
    CaddyAPILatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "caddy_api_latency_seconds",
            Help:    "Latency of Caddy Admin API calls.",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
    MCPToolCalls = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mcp_tool_calls_total",
            Help: "Total number of MCP tool calls.",
        },
        []string{"tool_name", "result"},
    )
)

func init() {
    prometheus.MustRegister(ConfigUpdates, CaddyAPILatency, MCPToolCalls)
}

// internal/observability/tracing.go
func InitTracing(serviceName, jaegerEndpoint string) (func(), error) {
    ctx := context.Background()

    // 使用标准 OpenTelemetry SDK + OTLP/gRPC 导出到 Jaeger
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(jaegerEndpoint),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String(serviceName),
            semconv.ServiceVersionKey.String("1.0.0"),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("create resource: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )
    otel.SetTracerProvider(tp)

    return func() {
        if err := tp.Shutdown(ctx); err != nil {
            slog.Error("shutdown tracing provider failed", "err", err)
        }
    }, nil
}
```

### 2.10 后端启动与初始化流程

```go
// cmd/server/main.go
func main() {
    ctx := context.Background()
    
    // 1. 加载配置
    cfg := config.Load()
    
    // 2. 初始化日志
    logger := observability.NewLogger()
    slog.SetDefault(logger)
    
    // 3. 初始化可观测性（Tracing 必须在 DB/Service 之前）
    shutdownTracing, err := observability.InitTracing("caddy-mgmt", cfg.JaegerURL)
    if err != nil {
        slog.Error("init tracing failed", "err", err)
        os.Exit(1)
    }
    // 注意：shutdownTracing 由 waitForShutdown 统一调用，避免重复关闭
    
    
    // 4. 初始化数据库连接池
    db := database.NewPool(cfg.DatabaseURL)
    
    // 5. 执行数据库迁移
    if err := database.Migrate(cfg.DatabaseURL, "ent/migrate"); err != nil {
        slog.Error("migration failed", "err", err)
        os.Exit(1)
    }
    
    // 6. 初始化 Ent Client（类型安全的数据访问层）
    client := ent.NewClient(ent.Driver(sql.OpenDB("postgres", db)))
    defer client.Close()
    
    // 7. 初始化 Redis
    redis := cache.NewRedis(cfg.RedisURL)
    
    // 8. 初始化 Caddy 客户端 (mTLS)
    caddyClient, err := caddy.NewClient(
        cfg.CaddyCertFile, cfg.CaddyKeyFile,
        cfg.CaddyCAFile, cfg.CaddyAdminURL,
    )
    if err != nil {
        slog.Error("init caddy client failed", "err", err)
        os.Exit(1)
    }
    
    // 9. 配置同步：确保 Caddy 运行态与数据库一致
    syncService := caddy.NewSyncService(caddyClient, client)
    if err := syncService.SyncOnStartup(ctx); err != nil {
        slog.Error("config sync failed", "err", err)
    }
    
    // 10. 初始化 Service 层
    services := service.NewServices(client, caddyClient, redis)
    
    // 11. 初始化 Handler 和 Middleware
    handlers := api.NewHandlers(services)
    mw := api.NewMiddleware(cfg, redis)
    
    // 12. 注册所有路由（HTTP Server 启动前完成）
    router := api.NewRouter(handlers, mw)
    
    // 13. 启动 AI 助手（可选，阶段 3）—— 在 HTTP Server 启动前注册路由
    if cfg.EnableAIAssistant {
        assistant, err := agent.NewAssistant(
            cfg.LLMProvider, cfg.LLMAPIKey, cfg.MCPServerURL,
        )
        if err != nil {
            slog.Error("init assistant failed", "err", err)
            os.Exit(1)
        }
        // 注册 WebSocket 处理器供前端对话
        router.GET("/ws/assistant", wsHandler(assistant))
    }
    
    // 14. 启动 HTTP Server (Gin)
    srv := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }
    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            slog.Error("HTTP server failed", "err", err)
        }
    }()
    
    // 15. 启动 MCP Server
    mcpServer := mcp.NewMCPServer(services, ":8081")
    go func() {
        if err := mcpServer.Start(ctx); err != nil {
            slog.Error("MCP server failed", "err", err)
        }
    }()
    
    // 16. 优雅关闭
    waitForShutdown(srv, mcpServer, db, syncService, shutdownTracing)
}
```

---

## 3. 前端系统架构

### 3.1 前端分层架构

```
┌─────────────────────────────────────────────────────────────┐
│                     前端应用架构                              │
│                                                             │
│  ┌─── 页面层 (Pages) ───────────────────────────────────┐  │
│  │  Login · Dashboard · ProjectDetail · DomainDetail    │  │
│  │  Settings · AuditLogs · Assistant                     │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                    │
│  ┌─── 组件层 (Components) ──────────────────────────────┐  │
│  │  布局: Header · Sidebar · Breadcrumb                   │  │
│  │  业务: DomainForm · ProxyConfig · UpstreamTable       │  │
│  │  通用: ConfirmDialog · StatusBadge · JsonViewer       │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                    │
│  ┌─── 状态层 (Store) ────────────────────────────────────┐  │
│  │  全局: useAuthStore (用户/Token/权限)                  │  │
│  │  项目: useProjectStore (当前项目切换)                   │  │
│  │  UI:   useUIStore (主题/侧边栏/加载)                   │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                    │
│  ┌─── 数据层 (Data) ─────────────────────────────────────┐  │
│  │  React Query: domains · projects · proxy · audit      │  │
│  │  API Client:  openapi-typescript-codegen 自动生成       │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                    │
│  ┌─── 基础设施 (Infra) ──────────────────────────────────┐  │
│  │  HTTP: Axios (拦截器/Token 刷新)                       │  │
│  │  WS:   WebSocket (AI 助手实时对话)                      │  │
│  │  i18n: react-i18next (中英文)                         │  │
│  │  主题: Ant Design ConfigProvider                       │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 页面结构与路由设计

```tsx
// src/App.tsx — 路由定义
const router = createBrowserRouter([
  // 公共路由
  { path: "/login", element: <LoginPage /> },
  { path: "/register", element: <RegisterPage /> },
  { path: "/oauth/callback", element: <OAuthCallback /> },
  
  // 需认证路由
  {
    path: "/",
    element: <ProtectedRoute><AppLayout /></ProtectedRoute>,
    children: [
      { index: true, element: <DashboardPage /> },
      
      // 项目管理
      { path: "projects", element: <ProjectListPage /> },
      { path: "projects/new", element: <ProjectCreatePage /> },
      { path: "projects/:id", element: <ProjectDetailPage />,
        children: [
          { index: true, element: <ProjectOverviewTab /> },
          { path: "domains", element: <DomainListTab /> },
          { path: "members", element: <MemberListTab /> },
          { path: "audit", element: <AuditLogTab /> },
          { path: "settings", element: <ProjectSettingsTab /> },
        ]
      },
      
      // 域名详情
      { path: "projects/:id/domains/:did", element: <DomainDetailPage />,
        children: [
          { index: true, element: <ProxyConfigTab /> },
          { path: "upstreams", element: <UpstreamListTab /> },
          { path: "status", element: <HealthStatusTab /> },
          { path: "ssl", element: <SSLConfigTab /> },
        ]
      },
      
      // 用户设置
      { path: "settings", element: <UserSettingsPage /> },
      { path: "settings/tokens", element: <APITokensPage /> },
      
      // AI 助手
      { path: "assistant", element: <AssistantPage /> },
    ]
  }
])
```

### 3.3 状态管理设计

```tsx
// src/stores/auth.ts — 全局认证状态
import { create } from 'zustand'

interface AuthState {
  token: string | null
  user: User | null
  currentProjectId: string | null
  setAuth: (token: string, user: User) => void
  logout: () => void
  switchProject: (projectId: string) => void
}

export const useAuthStore = create<AuthState>()(
  (set) => ({
    token: null,
    user: null,
    currentProjectId: null,
    setAuth: (token, user) => set({ token, user }),
    logout: () => set({ token: null, user: null }),
    switchProject: (projectId) => set({ currentProjectId: projectId }),
  })
)

// src/stores/ui.ts — UI 状态
interface UIState {
  sidebarCollapsed: boolean
  theme: 'light' | 'dark'
  language: 'zh' | 'en'
  toggleSidebar: () => void
  setTheme: (theme: 'light' | 'dark') => void
}
```

### 3.4 数据请求与缓存策略

```tsx
// src/hooks/useDomains.ts — 域名数据
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

// 查询域名列表
export function useDomains(projectId: string) {
  return useQuery({
    queryKey: ['domains', projectId],
    queryFn: () => api.getDomains(projectId),
    enabled: !!projectId,
    staleTime: 30 * 1000,      // 30 秒内不重新请求
    refetchInterval: 60 * 1000, // 每分钟自动刷新
  })
}

// 创建域名（乐观更新）
export function useCreateDomain(projectId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateDomainInput) => api.createDomain(projectId, input),
    
    // 乐观更新：立即在列表中插入
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey: ['domains', projectId] })
      const prev = qc.getQueryData<Domain[]>(['domains', projectId])
      qc.setQueryData<Domain[]>(['domains', projectId], (old = []) => [
        ...old,
        { id: 'temp', domain_name: input.domain, status: 'pending' }
      ])
      return { prev }
    },
    
    // 失败回滚
    onError: (_err, _input, ctx) => {
      qc.setQueryData(['domains', projectId], ctx?.prev)
    },
    
    // 成功后刷新
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ['domains', projectId] })
    },
  })
}

// 查询上游健康状态（高频刷新）
export function useUpstreamStatus(domainId: string) {
  return useQuery({
    queryKey: ['upstream-status', domainId],
    queryFn: () => api.getUpstreamStatus(domainId),
    refetchInterval: 10 * 1000,  // 每 10 秒刷新
  })
}
```

### 3.5 API 客户端生成

```yaml
# openapi-config.yaml — 从后端 OpenAPI 自动生成前端客户端
openapi-config.yaml:
  input: http://localhost:8080/openapi.json
  output: src/api/generated
  client: axios
  name: ApiClient
```

```bash
# 生成命令
openapi-typescript-codegen --input http://localhost:8080/openapi.json \
  --output src/api/generated --client axios
```

```tsx
// src/api/client.ts — Axios 实例 + 拦截器
import axios from 'axios'
import { useAuthStore } from '../stores/auth'

const client = axios.create({ baseURL: '/api/v1', timeout: 30000 })

// 请求拦截器：自动附加 JWT
client.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// 响应拦截器：自动刷新 Token
client.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      const refreshed = await refreshToken()
      if (refreshed) {
        error.config.headers.Authorization = `Bearer ${refreshed}`
        return client(error.config)  // 重试原请求
      }
      useAuthStore.getState().logout()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)
```

### 3.6 前端安全策略

| 策略 | 实现 |
|---|---|
| **XSS 防护** | React 默认转义 + DOMPurify 清洗用户输入 |
| **CSRF 防护** | JWT Bearer Token 认证（无 Cookie，天然防 CSRF） |
| **Access Token 存储** | 内存存储（如 React state / closure），不持久化，短期有效 |
| **Refresh Token 存储** | HttpOnly、Secure、SameSite=Strict Cookie，由后端写入，前端不可读取 |
| **路由守卫** | `<ProtectedRoute>` 组件检查认证状态 |
| **权限控制** | `<RequireRole role="admin">` 组件控制 UI 元素显示 |
| **敏感信息脱敏** | API Token 展示时仅显示前缀 `tok_abc1****` |

### 3.7 实时数据推送

```tsx
// src/hooks/useAssistant.ts — AI 助手 WebSocket 连接
const WS_URL = import.meta.env.VITE_WS_URL || 'wss://localhost:8080/ws/assistant'
const RECONNECT_DELAY = 3000
const HEARTBEAT_INTERVAL = 30000

export function useAssistant() {
  const [messages, setMessages] = useState<Message[]>([])
  const [streaming, setStreaming] = useState(false)
  const [pendingConfirm, setPendingConfirm] = useState<PendingConfirm | null>(null)
  const ws = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const heartbeatTimer = useRef<ReturnType<typeof setInterval> | null>(null)
  const shouldReconnect = useRef(true)

  const clearTimers = () => {
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current)
      reconnectTimer.current = null
    }
    if (heartbeatTimer.current) {
      clearInterval(heartbeatTimer.current)
      heartbeatTimer.current = null
    }
  }

  const connect = () => {
    if (ws.current?.readyState === WebSocket.OPEN) return

    shouldReconnect.current = true
    // 使用 wss:// 并通过后端 session / cookie 鉴权，避免在 URL 中暴露 token
    const socket = new WebSocket(WS_URL)

    socket.onopen = () => {
      setStreaming(true)
      // 心跳
      heartbeatTimer.current = setInterval(() => {
        if (socket.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({ type: 'ping' }))
        }
      }, HEARTBEAT_INTERVAL)
    }

    socket.onmessage = (event) => {
      let data
      try {
        data = JSON.parse(event.data)
      } catch {
        return
      }
      if (data.type === 'pong') return

      switch (data.type) {
        case 'text':
          setMessages(prev => appendChunk(prev, data.chunk))
          break
        case 'tool_call':
          setPendingConfirm({
            tool: data.tool_name,
            args: data.arguments,
            onConfirm: () => socket.send(JSON.stringify({ type: 'confirm' })),
            onCancel: () => socket.send(JSON.stringify({ type: 'cancel' })),
          })
          break
        case 'tool_result':
          setMessages(prev => appendToolResult(prev, data.result))
          break
        case 'done':
          setStreaming(false)
          break
      }
    }

    socket.onclose = () => {
      setStreaming(false)
      clearTimers()
      if (shouldReconnect.current) {
        reconnectTimer.current = setTimeout(() => {
          connect()
        }, RECONNECT_DELAY)
      }
    }

    socket.onerror = (err) => {
      console.error('WebSocket error', err)
      socket.close()
    }

    ws.current = socket
  }

  useEffect(() => {
    connect()
    return () => {
      shouldReconnect.current = false
      clearTimers()
      ws.current?.close()
    }
  }, [])

  const send = (text: string) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      setStreaming(true)
      ws.current.send(JSON.stringify({ type: 'query', text }))
    }
  }

  const confirm = (confirmed: boolean) => {
    ws.current?.send(JSON.stringify({ type: confirmed ? 'confirm' : 'cancel' }))
  }

  return { messages, streaming, send, confirm, pendingConfirm }
}
```

---

## 4. 前后端交互协议

### 4.1 REST API 规范

| 规则 | 说明 | 示例 |
|---|---|---|
| URL 风格 | RESTful 资源嵌套 | `/api/v1/projects/:id/domains/:did/proxy` |
| HTTP 方法语义 | GET 查询、POST 创建、PATCH 更新、DELETE 删除 | `PATCH /api/v1/projects/123` |
| 版本控制 | URL 前缀 | `/api/v1/` |
| 嵌套深度 | 最多 2 级 | `projects/:id/domains/:did` |
| 列表查询 | Query 参数 | `?page=1&size=20&tag=api-gateway` |
| ID 格式 | UUID | `550e8400-e29b-41d4-a716-446655440000` |

### 4.2 统一响应格式

```jsonc
// 成功响应
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "domain_name": "api.abc.com",
    "status": "active"
  },
  "request_id": "req_abc123",
  "timestamp": "2026-07-24T14:00:00Z"
}

// 列表响应（含分页）
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [...],
    "total": 42,
    "page": 1,
    "size": 20
  },
  "request_id": "req_abc123"
}

// 错误响应
{
  "code": 40301,
  "message": "Access denied: resource does not belong to your project",
  "data": null,
  "request_id": "req_abc123",
  "timestamp": "2026-07-24T14:00:00Z"
}
```

### 4.3 错误码体系

| 范围 | 类别 | 示例 |
|---|---|---|
| `0` | 成功 | `0` = OK |
| `400xx` | 客户端请求错误 | `40001` = 参数校验失败 |
| `401xx` | 认证错误 | `40101` = Token 无效 / `40102` = Token 过期 |
| `403xx` | 授权错误 | `40301` = 越权访问 / `40302` = 角色不足 / `40303` = 配额超限 |
| `404xx` | 资源不存在 | `40401` = 项目不存在 / `40402` = 域名不存在 |
| `409xx` | 冲突 | `40901` = 域名已存在 / `40902` = 并发修改冲突 |
| `412xx` | 前置条件失败 | `41201` = Caddy Etag 不匹配 |
| `422xx` | 业务校验失败 | `42201` = 端口不在白名单 / `42202` = 配置校验失败 |
| `429xx` | 请求速率超限 | `42901` = 超过 per-user 限流阈值 / `42902` = 超过 per-project 限流阈值 |
| `500xx` | 服务器错误 | `50001` = Caddy 不可达 / `50002` = 数据库错误 |
| `503xx` | 服务不可用 | `50301` = 系统维护中 |

### 4.4 分页与过滤

```
GET /api/v1/projects/:id/domains
  ?page=1                    # 页码（从 1 开始）
  &size=20                   # 每页条数（最大 100）
  &sort=created_at:desc      # 排序字段:方向
  &status=active              # 状态过滤
  &search=api                # 搜索关键词
```

### 4.5 认证流程

```
┌────────┐                         ┌────────┐                    ┌──────────┐
│ 前端    │                         │ 后端    │                    │ Redis    │
└───┬────┘                         └───┬────┘                    └────┬─────┘
    │                                  │                              │
    │  POST /auth/login                │                              │
    │  { email, password }             │                              │
    ├─────────────────────────────────>│                              │
    │                                  │  验证密码 (Argon2id)          │
    │                                  │  生成 JWT (RS256, 15min)      │
    │                                  │  生成 Refresh Token (7d)      │
    │                                  │  存储 Refresh Token ─────────>│
    │  200 OK                          │                              │
    │  { access_token, refresh_token } │                              │
    │<─────────────────────────────────┤                              │
    │                                  │                              │
    │  GET /api/v1/projects            │                              │
    │  Authorization: Bearer <jwt>     │                              │
    ├─────────────────────────────────>│                              │
    │                                  │  验证 JWT 签名               │
    │                                  │  提取 user_id, project_id    │
    │  200 OK                          │                              │
    │<─────────────────────────────────┤                              │
    │                                  │                              │
    │  (JWT 过期后)                     │                              │
    │  POST /auth/refresh              │                              │
    │  { refresh_token }               │                              │
    ├─────────────────────────────────>│                              │
    │                                  │  验证 Refresh Token <────────│
    │                                  │  生成新 JWT                   │
    │  200 OK                          │                              │
    │  { access_token }                │                              │
    │<─────────────────────────────────┤                              │
    │                                  │                              │
    │  POST /auth/logout               │                              │
    ├─────────────────────────────────>│                              │
    │                                  │  撤销 JWT (jti → blacklist)──>│
    │                                  │  删除 Refresh Token ─────────>│
    │  204 No Content                  │                              │
    │<─────────────────────────────────┤                              │
```

---

## 5. 部署架构

### 5.1 MVP 部署拓扑

```yaml
# deployments/docker-compose.yml
version: '3.8'

services:
  # 控制面（Go 单二进制）
  control-plane:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile
    ports:
      - "8080:8080"    # REST API
      - "8081:8081"    # MCP Server
    environment:
      - DATABASE_URL=postgres://caddy:caddy@postgres:5432/caddy_mgmt
      - REDIS_URL=redis://redis:6379
      - CADDY_ADMIN_URL=https://caddy:2021
      - CADDY_CERT_FILE=/certs/client.crt
      - CADDY_KEY_FILE=/certs/client.key
      - CADDY_CA_FILE=/certs/ca.crt
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      caddy:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    restart: unless-stopped
  
  # Caddy 数据面
  caddy:
    image: caddy:2.7-alpine
    ports:
      - "80:80"        # HTTP
      - "443:443"      # HTTPS
      # Admin API (:2021) 不暴露到宿主机，仅容器内网络访问
      # 如需调试，使用: - "127.0.0.1:2021:2021"
    volumes:
      - ./caddy/Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
      - ./certs:/certs:ro
    healthcheck:
      test: ["CMD", "caddy", "version"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    restart: unless-stopped
  
  # 数据库
  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_DB=caddy_mgmt
      - POSTGRES_USER=caddy
      - POSTGRES_PASSWORD=caddy
    volumes:
      - pg_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U caddy -d caddy_mgmt"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped
  
  # 缓存
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped
  
  # 前端（阶段 3）
  web:
    build:
      context: ./web
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      - VITE_API_URL=http://localhost:8080
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    restart: unless-stopped
  
  # 可观测性（可选，MVP 阶段可省略）
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:9090/-/healthy"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    restart: unless-stopped
  
  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    depends_on:
      prometheus:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:3000/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    restart: unless-stopped

volumes:
  caddy_data:
  caddy_config:
  pg_data:
```

### 5.2 生产部署拓扑

```
                    ┌─── 入口 ──────────────────────────────┐
                    │  CloudFlare / AWS ALB (DNS + WAF)     │
                    └──────────────┬───────────────────────┘
                                   │
                    ┌──────────────▼───────────────────────┐
                    │  K8s Ingress (Caddy)                 │
                    │  :80 / :443 (TLS 自动化)             │
                    └──┬──────────────┬────────────────────┘
                       │              │
            ┌──────────▼──┐    ┌──────▼──────────────────────┐
            │ 控制面集群   │    │ Caddy 数据面集群            │
            │ (3 replicas)│    │ (2 replicas, 主写副本读)    │
            │ K8s Service │    │ K8s Service                 │
            └──────┬──────┘    └──────────┬──────────────────┘
                   │                      │
         ┌─────────┼─────────┐            │ mTLS
         │         │         │            │
    ┌────▼───┐ ┌───▼──┐ ┌───▼──┐    ┌────▼────┐
    │PG 主   │ │Redis │ │MinIO │    │ Caddy   │
    │+ 只读  │ │Cluster│ │(S3)  │    │ Admin   │
    │副本    │ │      │ │      │    │ API     │
    └────────┘ └──────┘ └──────┘    └─────────┘
```

### 5.3 环境矩阵

| 环境 | 前端 | 控制面 | Caddy | 数据库 | Redis | 用途 |
|---|---|---|---|---|---|---|
| **development** | Vite dev | `go run` | Docker | Docker | Docker | 本地开发 |
| **staging** | Docker | Docker | Docker | Docker | Docker | 集成测试 |
| **production** | K8s | K8s (3 replicas) | K8s (2 replicas) | RDS/CloudSQL | Elasticache | 生产环境 |

---

## 6. 横切关注点

### 6.1 配置管理

```go
// internal/config/config.go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    Caddy    CaddyConfig
    Auth     AuthConfig
    AI       AIConfig
    Observability ObservabilityConfig
}

// 环境变量映射（12-Factor App）
func Load() *Config {
    return &Config{
        Server: ServerConfig{
            HTTPPort: env("HTTP_PORT", "8080"),
            MCPPort:  env("MCP_PORT", "8081"),
        },
        Database: DatabaseConfig{
            URL:         env("DATABASE_URL", "postgres://localhost:5432/caddy_mgmt"),
            MaxConns:    envInt("DB_MAX_CONNS", 25),
        },
        Caddy: CaddyConfig{
            AdminURL:   env("CADDY_ADMIN_URL", "https://localhost:2021"),
            CertFile:   env("CADDY_CERT_FILE", "/certs/client.crt"),
            KeyFile:    env("CADDY_KEY_FILE", "/certs/client.key"),
            CAFile:     env("CADDY_CA_FILE", "/certs/ca.crt"),
        },
        Auth: AuthConfig{
            JWTPrivateKey: env("JWT_PRIVATE_KEY", ""),
            JWTExpiresIn:  envDuration("JWT_EXPIRES_IN", 24*time.Hour),
        },
        AI: AIConfig{
            EnableAssistant: envBool("ENABLE_AI_ASSISTANT", false),
            LLMProvider:     env("LLM_PROVIDER", "openai"),
            LLMAPIKey:       env("LLM_API_KEY", ""),
        },
    }
}
```

### 6.2 日志规范

```go
// 日志格式（JSON）
{
  "time": "2026-07-24T14:00:00.123Z",
  "level": "INFO",
  "source": "internal/service/domain_service.go:42",
  "msg": "domain created",
  "request_id": "req_abc123",
  "project_id": "proj_abc",
  "user_id": "user_001",
  "domain": "api.abc.com",
  "caddy_id": "tenant_abc_route_001",
  "duration_ms": 45,
  "trace_id": "trace_xyz789"   // OpenTelemetry 追踪 ID
}

// 日志级别使用规范
slog.DebugContext(ctx, "ent query", "op", op)          // 开发调试
slog.InfoContext(ctx, "domain created", "domain", d)    // 业务事件
slog.WarnContext(ctx, "etag mismatch, retrying")         // 可恢复异常
slog.ErrorContext(ctx, "caddy unreachable", "err", err) // 系统错误
```

### 6.3 链路追踪

```
前端请求 (X-Request-ID: req_abc123)
    │
    ├─ Span: HTTP Handler (Gin)
    │   ├─ Span: Auth Middleware (JWT 验证)
    │   ├─ Span: RBAC Check
    │   ├─ Span: DomainService.Create
    │   │   ├─ Span: PostgreSQL Query (Ent)
    │   │   ├─ Span: Caddy API Call (mTLS)
    │   │   │   ├─ Span: GET /config/ (获取 Etag)
    │   │   │   └─ Span: POST /config/.../routes (下发配置)
    │   │   ├─ Span: Redis (RateLimit check)
    │   │   └─ Span: PostgreSQL (审计日志写入)
    │   └─ Span: Response Serialization
    │
    总耗时: 45ms (Caddy API 占 30ms, DB 占 10ms, 其他 5ms)
```

### 6.4 优雅关闭

```go
// cmd/server/main.go
func waitForShutdown(srv *http.Server, mcpServer *mcp.MCPServer, db *sql.DB, syncService *caddy.SyncService, shutdownTracing func()) {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    slog.Info("shutting down...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // 1. 停止接收新请求
    srv.Shutdown(ctx)
    mcpServer.Shutdown(ctx)
    
    // 2. 保存配置快照
    syncService.SaveSnapshot(ctx)
    
    // 3. 关闭数据库连接
    db.Close()
    
    // 4. 关闭链路追踪
    if shutdownTracing != nil {
        shutdownTracing()
    }
    
    slog.Info("server stopped")
}
```

---

## 附录: 文档关联关系

```
caddy-multi-tenant-architecture.md (v1.3)
  │  系统架构、数据模型、MCP 工具、安全策略
  │
  ├──> caddy-architecture-review.md
  │      审查报告、技术正确性验证
  │
  ├──> caddy-tech-stack-selection.md (v1.1)
  │      语言/框架/中间件选型、项目工程结构
  │
  └──> caddy-system-architecture.md (本文档)
         前后端分层架构、交互协议、部署拓扑
```
