# 技术栈与技术框架选型文档

> **版本**: v1.1  
> **日期**: 2026-07-24  
> **关联文档**: `caddy-multi-tenant-architecture.md` v1.3  
> **状态**: 选型评审中

---

## 目录

- [1. 选型背景与约束](#1-选型背景与约束)
- [2. 控制面后端语言选型](#2-控制面后端语言选型)
  - [2.1 Go vs Python 全面对比](#21-go-vs-python-全面对比)
  - [2.2 最终结论](#22-最终结论)
- [3. 控制面 Web 框架选型](#3-控制面-web-框架选型)
  - [3.1 候选框架对比](#31-候选框架对比)
  - [3.2 最终结论](#32-最终结论)
- [4. 数据层选型](#4-数据层选型)
  - [4.1 关系型数据库](#41-关系型数据库)
  - [4.2 缓存与会话存储](#42-缓存与会话存储)
  - [4.3 ORM / 数据访问层](#43-orm--数据访问层)
- [5. MCP Server 框架选型](#5-mcp-server-框架选型)
  - [5.1 MCP SDK 生态调研](#51-mcp-sdk-生态调研)
  - [5.2 最终结论](#52-最终结论)
- [6. 认证与安全组件](#6-认证与安全组件)
- [7. 可观测性技术栈](#7-可观测性技术栈)
- [AI 智能助手框架选型（新增）](#ai-智能助手框架选型新增)
  - [调研背景](#调研背景)
  - [三大框架全面对比](#三大框架全面对比)
  - [关键决策因子：MCP 协议原生集成](#关键决策因子mcp-协议原生集成)
  - [最终结论](#最终结论)
  - [AI 助手架构设计（预览）](#ai-助手架构设计预览)
  - [后续 AI 能力扩展路径](#后续-ai-能力扩展路径)
  - [新增依赖](#新增依赖)
- [8. 前端技术栈](#8-前端技术栈)
- [9. 基础设施与部署](#9-基础设施与部署)
- [10. 完整技术栈总览](#10-完整技术栈总览)
- [11. 项目工程结构](#11-项目工程结构)
- [12. 第三方依赖清单](#12-第三方依赖清单)

---

## 1. 选型背景与约束

### 1.1 架构约束（来自架构设计文档 v1.3）

| 约束维度 | 具体要求 | 来源 |
|---|---|---|
| **数据面** | Caddy v2.11.0+（Go 编写），通过 Admin API 管理 | §1.1 |
| **控制面** | 需要 HTTP 反向代理 Caddy Admin API、翻译 JSON 配置、校验 Schema | §1.5 |
| **认证** | JWT (RS256) + mTLS 双向证书 | §1.4 |
| **MCP** | 需要实现 MCP Server（JSON-RPC 2.0 over HTTP+SSE） | §4.1 |
| **数据库** | PostgreSQL（已有 DDL，使用数组类型 `INT[]`、`TEXT[]`、`JSONB`、`GIN` 索引） | §3.1 |
| **缓存** | Redis（会话、速率限制、分布式锁回退） | §1.5 |
| **并发** | 需要 HTTP 客户端调用 Caddy API，处理 Etag/If-Match | §3.3 |
| **可观测性** | Prometheus 指标 + Grafana 仪表盘 | §5.3 |

### 1.2 关键决策因子

| 因子 | 权重 | 说明 |
|---|:---:|---|
| **与 Caddy 生态契合度** | ⭐⭐⭐⭐⭐ | Caddy 本身是 Go 项目，Go 可直接引用其内部包 |
| **MCP SDK 成熟度** | ⭐⭐⭐⭐⭐ | 需要 Tier 1 官方 SDK 支持 |
| **API 吞吐性能** | ⭐⭐⭐⭐ | 控制面需代理所有 Caddy API 调用 |
| **类型安全** | ⭐⭐⭐⭐ | JSON 配置翻译需要严格类型保障 |
| **团队招聘难度** | ⭐⭐⭐ | 影响长期维护 |
| **开发效率** | ⭐⭐⭐ | 12 周交付周期 |

---

## 2. 控制面后端语言选型

### 2.1 Go vs Python 全面对比

架构文档原始建议为"Go (Gin) 或 Python (FastAPI)"。经深度调研，以下是全面对比：

#### 2.1.1 性能对比（AWS 生产级基准测试）

| 指标 | Go (标准库) | Python (FastAPI) | 倍数差异 |
|---|---|---|:---:|
| **纯 API 吞吐量** | 62,000 RPS | 13,000 RPS | **4.8x** |
| **带数据库+缓存** | 35,000 RPS | 800 RPS | **43.8x** |
| **P99 延迟** | 低（基线） | 2x 以上 | 2x |
| **CPU 耗尽点** | ~60,000 RPS | ~13,000 RPS | 4.6x |
| **内存效率** | 高（GC 紧凑） | 高内存占用 | Go 优 |

> 数据来源：Anton Putra 基准测试（AWS EKS, m7a.large, PostgreSQL + Memcached）

> **注意**：43.8x 差异为特定压测条件下的峰值数据（完整请求链路含数据库+缓存），不可直接外推至所有场景。通用场景下 Go 吞吐量优势约为 3-10 倍。纯 API 吞吐层面差异约为 4.8x。实际差异因测试条件（连接池大小、查询复杂度、序列化开销）不同而在 10x-44x 范围内波动。

#### 2.1.2 生态契合度对比

| 维度 | Go | Python | 优势方 |
|---|---|---|:---:|
| **Caddy 原生集成** | 可引用 Caddy 公开包 `github.com/caddyserver/caddy/v2` 的类型定义（注意：引入会增加二进制体积，建议仅引用必要子包） | 仅能通过 HTTP 调用 Admin API | **Go** |
| **MCP SDK** | 官方 Tier 1 SDK（支持 SSE/Streamable HTTP） | 官方 Tier 1 SDK（支持 SSE/Streamable HTTP） | 平手 |
| **JSON Schema 校验** | `github.com/santhosh-tekuri/jsonschema/v6`（推荐）<br>`github.com/xeipuuv/gojsonschema`（已停止维护，建议替换） | `jsonschema` 库（更成熟） | **Python** |
| **PostgreSQL 驱动** | `pgx`（高性能，原生协议） | `asyncpg`（高性能异步） | 平手 |
| **mTLS 支持** | `crypto/tls` 标准库（原生支持） | `ssl` 模块（需额外配置） | **Go** |
| **并发模型** | Goroutine（轻量级，原生并发） | asyncio（单线程事件循环） | **Go** |
| **类型安全** | 静态类型编译（编译期捕获错误） | 类型注解可选（运行时错误） | **Go** |
| **部署形态** | 单二进制（无运行时依赖） | 需 Python 运行时 + 依赖管理 | **Go** |
| **开发速度** | 略慢（强类型约束） | 快（动态语言灵活） | **Python** |
| **招聘难度** | 中等 | 较低 | **Python** |

#### 2.1.3 关键场景分析

**场景 1：Caddy JSON 配置翻译**

```
业务模型 → Caddy JSON 翻译流程需要：
1. 将数据库记录映射为 Caddy JSON 结构
2. 校验 JSON Schema 合法性
3. 通过 HTTP PATCH/POST 下发到 Caddy Admin API
```

Go 可以引用 Caddy 的公开包 (`github.com/caddyserver/caddy/v2`) 中的类型定义（如 `caddyhttp.Route`、`reverseproxy.Handler`），在编译期保证 JSON 结构的正确性。但需注意引入 Caddy 包会增加约 30-50MB 的二进制体积，且与 Caddy 版本强耦合。替代方案是自行定义 Caddy JSON 的 Go 结构体映射，避免重量级依赖。Python 只能通过手工维护 JSON 结构映射，出错概率更高。

**场景 2：并发配置下发**

多个租户同时修改配置时，控制面需要并发调用 Caddy API 并处理 Etag/If-Match 冲突重试。Go 的 goroutine + channel 模型天然适合这种场景，而 Python 的 asyncio 在 CPU 密集的 JSON 序列化场景下会阻塞事件循环。

**场景 3：mTLS 客户端**

控制面作为 Caddy Admin API 的 mTLS 客户端，Go 的 `crypto/tls` 标准库提供完整的 TLS 1.3 支持，证书加载和验证 API 简洁直接。Python 需要处理 `ssl.SSLContext` 的配置，在异步框架中更容易出错。

### 2.2 最终结论

> **选型决定：Go**

| 决策因子 | 结论 |
|---|---|
| Caddy 生态契合度 | Go 可直接引用 Caddy 内部包，编译期类型保障 |
| 性能 | Go 吞吐量 5-44 倍领先，尤其在数据库操作场景 |
| MCP SDK | Go 官方 Tier 1 SDK，生产就绪 |
| 部署 | 单二进制，容器化极简 |
| 并发 | Goroutine 天然适合 API 代理 + 并发下发 |
| 唯一劣势 | 开发速度略慢，但 12 周周期可控 |

---

## 3. 控制面 Web 框架选型

### 3.1 候选框架对比

| 框架 | 定位 | 优势 | 劣势 | 适用场景 |
|---|---|---|---|---|
| **Gin** | 高性能、成熟、最大生态 | 社区大、中间件多、招人成本低 | `gin.Context` 易污染业务层 | 通用 REST API ✅ |
| **Echo** | 均衡、简洁、可扩展 | 设计克制、中间件完整 | 生态略小于 Gin | 团队 API ✅ |
| **Chi** | 标准库风格、轻量 | `net/http` 兼容、长期维护好 | 无参数绑定/校验，需自建 | 标准库派 ✅ |
| **Fiber** | Express 风格、高性能 | 开发手感好 | 基于 `fasthttp`，非 `net/http` | Node 转Go |
| **Huma** | OpenAPI 3.1 优先 | 自动文档、类型化 | 约束强、灵活度低 | 契约优先 |

### 3.2 最终结论

> **选型决定：Gin（v1.12.0+）**

**理由**：

1. **生态最大**：中间件最丰富（JWT、CORS、限流、Prometheus 指标），社区文档最多
2. **招人成本最低**：Go 后端开发者几乎都熟悉 Gin
3. **性能足够**：控制面不是性能瓶颈（Caddy 才是数据面），Gin 的性能完全满足 API Gateway 需求
4. **架构约束适配**：文档中已大量使用 `PATCH /api/v1/...` 风格的 REST API，Gin 的路由分组和参数绑定天然适配

**工程纪律要求**：
```go
// ✅ 正确：gin.Context 只留在 Web 层
func handler(c *gin.Context) {
    result, err := svc.UpdateProxy(c.Request.Context(), req)  // 传标准库 context
    ...
}

// ❌ 错误：不要把 gin.Context 传进 Service 层
func handler(c *gin.Context) {
    result, err := svc.UpdateProxy(c)  // 业务层被框架绑定
    ...
}
```

---

## 4. 数据层选型

### 4.1 关系型数据库

> **选型决定：PostgreSQL 16+**（已在架构文档中确定）

| 特性 | 使用方式 | 文档引用 |
|---|---|---|
| 数组类型 `INT[]` / `TEXT[]` | `ports_exposed`、`ports_internal`、`tags` | §3.1.2 |
| JSONB | `scope`、`changes_before`、`changes_after`、`request_body` | §3.1.2 |
| GIN 索引 | `tags` 字段的数组索引 | §3.1.2 |
| ON DELETE CASCADE | 所有外键级联删除 | §3.1.2 |
| UUID | `gen_random_uuid()` 作为主键 | §3.1.2 |

**连接池配置建议**：

```go
// Go pgxpool 配置
poolConfig, _ := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/caddy_mgmt")
poolConfig.MaxConns = 25              // 最大连接数
poolConfig.MinConns = 5              // 最小空闲连接
poolConfig.MaxConnLifetime = 30 * time.Minute
poolConfig.MaxConnIdleTime = 5 * time.Minute
```

### 4.2 缓存与会话存储

> **选型决定：Redis 7.x**

| 用途 | 数据结构 | 过期策略 |
|---|---|---|
| JWT 撤销名单 | SET (jti → blacklist) | 与 JWT 过期时间一致 |
| API 速率限制 | 计数器（INCR + EXPIRE） | 60 秒滚动窗口 |
| 分布式锁（Etag 回退） | Redlock（` SET NX EX`） | 30 秒自动释放 |
| MCP 会话状态 | Hash（session_id → 上下文） | 24 小时 |
| 配置 Etag 缓存 | String（caddy_id → etag） | 5 分钟 |

### 4.3 ORM / 数据访问层

> **选型决定：Ent (entgo.io)**

| 方案 | 优势 | 劣势 | 结论 |
|---|---|---|---|
| **GORM** | 功能全、自动迁移 | 性能差、N+1 问题、隐藏 SQL | ❌ 不选 |
| **Ent** | Schema 即代码、类型安全、链式查询 API、通过 Atlas 自动生成迁移 | 学习曲线高 | ✅ **选型** |
| **sqlc** | 从 SQL 生成 Go 类型安全代码、零运行时开销 | 只支持静态 SQL，动态查询需手写；无 schema 即代码能力 | ❌ 不选 |
| **squirrel + pgx** | 灵活构建动态查询 | 无类型安全 | ❌ 不选 |

**Ent 工作流**：

```go
// ent/schema/project.go (代码优先的 schema 定义)
func (Project) Fields() []ent.Field {
    return []ent.Field{
        field.String("project_id").Unique(),
        field.String("name"),
        field.Text("description").Optional(),
        field.String("repository_url").Optional(),
        // ...
    }
}
```

```go
// ent 自动生成的代码（类型安全的链式查询 API）
func (c *Client) GetProjectByID(ctx context.Context, projectID string) (*Project, error) {
    return c.Project.Query().Where(project.ProjectID(projectID)).Only(ctx)
}

func (c *Client) CreateProject(ctx context.Context, p *Project) (*Project, error) {
    return c.Project.Create().
        SetProjectID(p.ProjectID).
        SetName(p.Name).
        Save(ctx)
}
```

> **理由**：Ent 采用代码优先（Schema as Code）的方式定义数据模型，自动生成类型安全的查询代码，开发效率高；通过 Atlas 集成可自动生成数据库迁移文件，避免手写 SQL DDL。Ent 原生支持 PostgreSQL 的数组类型、JSONB 等，并通过 Atlas 实现 schema 与迁移的一体化管理。复杂查询（如多表联查、原生 SQL 函数）可回退到 `client.QueryContext(ctx, rawSQL)` 执行。

**Ent 动态查询能力**：

本系统存在多处动态查询需求（如审计日志多条件过滤、项目列表按标签/环境过滤），Ent 的链式查询 API 天然支持动态构建条件：

```go
// 动态组合查询条件
query := c.Project.Query()
if env != "" {
    query = query.Where(project.EnvironmentEQ(env))
}
if len(tags) > 0 {
    query = query.Where(project.HasTags(tags...))
}
projects, err := query.All(ctx)
```

对于无法用 Ent 表达的复杂查询（如跨表聚合、原生 SQL 函数），可回退到原生 SQL：

```go
rows, err := c.QueryContext(ctx, `SELECT ... FROM ... WHERE ...`)
```

---

## 5. MCP Server 框架选型

### 5.1 MCP SDK 生态调研

经查证 [MCP 官方文档](https://modelcontextprotocol.io)，截至 2026 年 7 月，官方 SDK 生态如下：

| 语言 | 仓库 | 成熟度等级 | 版本 | 传输层支持 |
|---|---|:---:|---|---|
| **TypeScript** | `modelcontextprotocol/typescript-sdk` | **Tier 1** | v1.x | stdio / SSE / Streamable HTTP |
| **Python** | `modelcontextprotocol/python-sdk` | **Tier 1** | v1.x | stdio / SSE / Streamable HTTP |
| **Go** | `modelcontextprotocol/go-sdk` | **Tier 1** | v1.6.1 | stdio / SSE / Streamable HTTP |
| **C#** | `modelcontextprotocol/csharp-sdk` | **Tier 1** | v1.x | stdio / SSE / Streamable HTTP |
| Java | `modelcontextprotocol/java-sdk` | Tier 2 | — | — |
| Rust | `modelcontextprotocol/rust-sdk` | Tier 2 | — | — |
| Swift / Ruby / PHP / Kotlin | 各自仓库 | Tier 3 | — | 不稳定 |

> **Tier 1 定义**：100% 协议一致性测试通过率、稳定 v1.0+ 版本、2 个工作日内问题响应、7 天内 P0 缺陷修复。

### 5.2 最终结论

> **选型决定：Go MCP SDK (`github.com/modelcontextprotocol/go-sdk`) v1.6.1+**

**理由**：

1. **与控制面同语言**：MCP Server 与控制面 API Gateway 部署在同一进程内，Go SDK 自然集成

> **注**：SDK 版本号和 Tier 分级以 MCP 官方仓库（https://github.com/modelcontextprotocol/go-sdk）最新发布为准，开发前需 `go get` 确认实际可用版本。
2. **Tier 1 官方支持**：100% 协议一致性，生产就绪
3. **传输层完备**：支持架构文档要求的 HTTP+SSE 和 stdio 两种传输方式
4. **OAuth 支持**：SDK 内置 `auth` 和 `oauthex` 包，可用于 MCP 会话认证

**集成架构**：

```
┌─────────────────────────────────────────────────────┐
│                  Go 控制面进程                       │
│                                                     │
│  ┌──────────────┐    ┌──────────────┐              │
│  │ Gin HTTP     │    │ MCP Server   │              │
│  │ Server       │    │ (go-sdk)     │              │
│  │ :8080        │    │ :8081        │              │
│  │              │    │              │              │
│  │ REST API     │    │ JSON-RPC 2.0 │              │
│  │ (用户/Web UI) │    │ (AI Agent)   │              │
│  └──────┬───────┘    └──────┬───────┘              │
│         │                   │                      │
│         └─────────┬─────────┘                      │
│                   ▼                                │
│         ┌─────────────────┐                        │
│         │ Service Layer   │  ← 共享业务逻辑         │
│         │ (项目/域名/代理) │                        │
│         └────────┬────────┘                        │
│                  ▼                                 │
│         ┌─────────────────┐                        │
│         │ Caddy Client     │  ← mTLS HTTP Client   │
│         └─────────────────┘                        │
└─────────────────────────────────────────────────────┘
```

MCP Server 与 Gin HTTP Server 共享 Service 层，确保 REST API 和 AI Agent 操作走相同的权限校验和审计路径。

> **MCP SDK 兼容提示**：Eino 的 MCP 组件（`eino-ext/components/tool/mcp`）底层依赖 `github.com/mark3labs/mcp-go`，而非官方 `modelcontextprotocol/go-sdk`。若后续在内部 AI 助手中使用 Eino 消费本系统的 MCP 工具，且仍坚持使用官方 Go SDK 暴露服务，则需自行实现 MCP client 适配层；更简单的方案是统一使用 `github.com/mark3labs/mcp-go` 同时实现 Server 与 Client。

---

## 6. 认证与安全组件

| 组件 | 选型 | 用途 | 版本 |
|---|---|---|---|
| **JWT 签发与验证** | `golang-jwt/jwt/v5` | RS256 非对称签名 | v5.3.1+ |
| **密码哈希** | `golang.org/x/crypto/argon2` | Argon2id 密码存储 | 最新 |
| **2FA TOTP** | `pquerna/otp` | TOTP 时间一次性密码 | v1.5.0 |
| **OAuth 2.0** | `golang.org/x/oauth2` | GitHub/Google/SSO | 最新 |
| **mTLS 客户端** | Go `crypto/tls` 标准库 | 控制面→Caddy 双向 TLS | 原生 |
| **证书管理** | Caddy 内部 CA (`tls.issuance.internal`) | mTLS 证书签发 | Caddy 原生 |
| **输入校验** | `go-playground/validator/v10` | 请求参数校验（与 Gin 集成） | v10.30.0+ |
| **CSRF 防护** | 不需要（JWT Bearer Token 认证方案天然免疫 CSRF） | 纯 Token 认证无 Cookie，不受 CSRF 攻击 | — |

---

## 7. 可观测性技术栈

| 维度 | 选型 | 说明 |
|---|---|---|
| **指标采集** | Prometheus + `prometheus/client_golang` v1.24.0+ | Go 原生客户端，Gin 中间件已有 |
| **仪表盘** | Grafana | Prometheus 数据源可视化 |
| **结构化日志** | `log/slog`（Go 1.24+ 标准库） | 结构化 JSON 日志，替代 logrus/zap |
| **链路追踪** | OpenTelemetry Go SDK v1.44.0+ | `go.opentelemetry.io/otel` + Jaeger/Tempo |
| **审计日志** | PostgreSQL `project_audit_logs` 表 | 结构化存储，支持 JSONB diff |
| **告警** | Prometheus AlertManager | 配额超限、错误率、上游不健康告警 |

**Go slog 使用示例**：

```go
// 结构化日志配置
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
    AddSource: true,
}))
slog.SetDefault(logger)

// 带上下文的日志
slog.InfoContext(ctx, "config updated",
    "project_id", projectID,
    "caddy_id", caddyID,
    "action", "domain.create",
    "duration_ms", elapsedMs,
)
```

---

## AI 智能助手框架选型（新增）

> **背景**：系统后续将内置 AI 智能助手，不仅对外暴露 MCP 工具供外部 AI 调用，还需要在系统内部实现一个自主 Agent，能够理解自然语言指令、调用 Caddy 管理工具、执行多步操作流程。本节选型针对这一内部 Agent 框架。

### 调研背景

Go 语言在 AI/LLM 领域的生态在 2025-2026 年快速成熟，目前已有三大生产级框架：

| 框架 | 出品方 | 仓库 | 定位 |
|---|---|---|---|
| **Eino** | 字节跳动 CloudWeGo | `github.com/cloudwego/eino` | 工程化 LLM 应用开发框架 |
| **LangChainGo** | 社区 | `github.com/tmc/langchaingo` | LangChain 的 Go 实现 |
| **Google ADK-Go** | Google | `github.com/google/adk-go` | 官方 Agent Development Kit |

### 三大框架全面对比

| 维度 | Eino (CloudWeGo) | LangChainGo | Google ADK-Go |
|---|---|---|---|
| **出品方** | 字节跳动 | 社区（tmc） | Google 官方 |
| **GitHub Stars** | ~12,500+（截至 2026.07，持续增长） | ~5,000+ | ~8,500+ |
| **版本状态** | v0.9.13（pre-1.0） | v0.1.14 | v2.1.0 稳定版 |
| **生产验证** | 字节内部大规模使用后开源 | 社区驱动，有生产案例但无大厂背书 | Google Cloud 原生支持 |
| **Agent 类型** | ReAct Agent + DeepAgent（多 Agent 编排） | MRKL / OpenAI Functions / Conversational | 模块化多 Agent 系统 |
| **工具调用** | ✅ 原生 Tool 接口 + ToolsConfig | ✅ OpenAI Functions + 并行工具调用 | ✅ 丰富工具生态 |
| **MCP 协议集成** | ✅ 支持（通过 eino-ext/mcp 组件，底层使用 `github.com/mark3labs/mcp-go`） | ❌ 无原生 MCP 集成 | ✅ 支持（v2.1.0+ 新增 McpToolset、RemoteAgent） |
| **RAG 支持** | ✅ Retriever + Embedding 抽象 | ✅ 10+ 向量数据库集成 | ✅ 基础支持 |
| **LLM Provider** | OpenAI / Claude / Gemini / Ark / Ollama | 14+ Provider（最全面） | 主要面向 Gemini（model-agnostic 但优化 Gemini） |
| **流式输出** | ✅ 一等公民（自动拼接、合并、复制） | ✅ SSE 支持 | ✅ 支持 |
| **记忆/对话** | ✅ 上下文管理 + 摘要中间件 + 中断/恢复 | ✅ Token buffer memory | ✅ 支持 |
| **多 Agent 编排** | ✅ Graph DAG 编排 + DeepAgent | ❌ 无原生多 Agent | ✅ A2A 协议 + Supervisor |
| **Human-in-the-loop** | ✅ 中断/恢复 + 状态持久化 | ❌ 无原生支持 | ✅ 支持 |
| **中间件系统** | ✅ 摘要、工具缩减、文件系统等 | ❌ 无 | ✅ 支持 |
| **部署友好度** | Go 原生，CloudWeGo 生态 | Go 原生 | Go + Google Cloud 深度集成 |
| **厂商锁定** | 无 | 无 | 有（Google Cloud 优化） |
| **中文社区** | ✅ 强（字节跳动 + CloudWeGo 中文文档） | 弱 | 弱 |

> **稳定性风险**：Eino 当前仍为 pre-1.0 阶段，API 可能在 1.0 前发生破坏性变更，建议锁定 minor 版本并关注官方升级说明。

### 关键决策因子：MCP 协议集成

本系统的 AI 助手需要调用已定义的 14 个 MCP 工具（`caddy_domain_create`、`project_create` 等）。当前框架中，**Eino 通过 eino-ext/mcp 组件支持 MCP 协议**，ADK-Go 自 v2.1.0 起新增 McpToolset / RemoteAgent 也支持 MCP。但需要注意，Eino 的 MCP 组件底层依赖 `github.com/mark3labs/mcp-go`，并非官方 `modelcontextprotocol/go-sdk`：

> **MCP SDK 兼容提示**：若系统对外使用官方 Go SDK（`modelcontextprotocol/go-sdk`）暴露 MCP Server，同时又希望使用 Eino 作为内部 AI 助手消费这些工具，需要自行实现 MCP client 适配层；或者统一使用 `github.com/mark3labs/mcp-go` 同时实现 MCP Server 与 Client。

```go
// Eino Agent 通过 MCP Client 直接消费本系统的 MCP 工具
import (
    "github.com/cloudwego/eino-ext/components/tool/mcp"
    "github.com/cloudwego/eino/adk"
)

// 1. 连接到本系统自身的 MCP Server
mcpClient, _ := client.NewSSEMCPClient("http://localhost:8081/sse")
mcpClient.Start(ctx)
mcpClient.Initialize(ctx, initRequest)

// 2. 自动发现所有 MCP 工具
tools, _ := mcp.GetTools(ctx, &mcp.Config{Cli: mcpClient})

// 3. 创建 ReAct Agent，自动选择并调用工具
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Model: chatModel,        // OpenAI / Claude / Gemini
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: tools,    // 直接注入 MCP 工具
        },
    },
})

// 4. 用户自然语言 → Agent 自主决策 → 调用 MCP 工具 → 返回结果
runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
result := runner.Query(ctx, "帮我的电商项目加一个上游 10.0.1.6:8080")
// Agent 自动: 解析意图 → 调用 caddy_upstream_add → 返回确认
```

> **架构意义**：Eino 的 MCP 集成使系统形成**自引用 AI 架构**——系统既对外暴露 MCP Server 供外部 AI（如 Claude、ChatGPT）调用，又内部使用 Eino Agent 消费自身 MCP 工具。这意味着所有工具逻辑只需实现一次（在 MCP Server 中），内外两个 AI 入口共享同一套工具定义和权限校验。

### 最终结论

> **选型决定：Eino（`github.com/cloudwego/eino`）**

| 决策因子 | 结论 |
|---|---|
| **MCP 原生集成** | Eino 通过 eino-ext/mcp 支持 MCP，Agent 可直接消费系统 14 个 MCP 工具，零适配成本（底层依赖 `github.com/mark3labs/mcp-go`） |
| **生产验证** | 字节跳动内部大规模使用后开源，经过真实流量验证 |
| **多 Agent 编排** | DeepAgent + Graph DAG 编排，适合未来"智能运维"场景（如自动扩容、故障自愈） |
| **Human-in-the-loop** | 中断/恢复 + 状态持久化，完美匹配 MCP 工具要求的"写操作需用户确认"设计 |
| **LLM Provider** | 支持 OpenAI / Claude / Gemini / Ollama，无厂商锁定 |
| **中文生态** | CloudWeGo 中文文档完善，社区活跃 |
| **唯一劣势** | 相比 LangChainGo 的 14+ Provider，Eino 的 Provider 数量略少但覆盖主流 |

### AI 助手架构设计（预览）

```
┌─────────────────────────────────────────────────────────────┐
│                    AI 智能助手架构                           │
│                                                             │
│  ┌─ 外部 AI 入口 ────────────────────────────────────────┐  │
│  │  Claude / ChatGPT / 其他 LLM                          │  │
│  │    │                                                   │  │
│  │    │ MCP 协议 (HTTP+SSE)                              │  │
│  │    ▼                                                   │  │
│  │  ┌─────────────────┐                                  │  │
│  │  │ MCP Server      │  ← 暴露 14 个工具               │  │
│  │  │ (go-sdk v1.6.1) │     caddy_domain_create 等      │  │
│  │  └────────┬────────┘                                  │  │
│  │           │                                            │  │
│  └───────────┼────────────────────────────────────────────┘  │
│              │ 共享                                        │
│  ┌─ 内部 AI 入口 ───────────────────────────────────────┐  │
│  │           │                                            │  │
│  │  │        ▼                                            │  │
│  │  ┌─────────────────┐    ┌──────────────────┐          │  │
│  │  │ Eino Agent      │───→│ MCP Client       │          │  │
│  │  │ (ReAct/Deep)    │    │ (eino-ext/mcp)   │          │  │
│  │  │                 │    │                  │          │  │
│  │  │ · 自然语言理解   │    │ 自动发现并调用   │          │  │
│  │  │ · 多步推理       │    │ 14 个 MCP 工具  │          │  │
│  │  │ · Human-in-loop │    │                  │          │  │
│  │  │ · 上下文记忆     │    └────────┬─────────┘          │  │
│  │  └─────────────────┘             │                    │  │
│  └──────────────────────────────────┼────────────────────┘  │
│                                     │                        │
│                              ┌──────▼──────┐                 │
│                              │ Service Layer│ ← 统一权限校验  │
│                              │ (Go)         │   和审计记录     │
│                              └──────┬──────┘                 │
│                                     │                        │
│                              ┌──────▼──────┐                 │
│                              │ Caddy Admin  │                 │
│                              │ API (mTLS)   │                 │
│                              └─────────────┘                 │
└─────────────────────────────────────────────────────────────┘
```

**两种 AI 交互模式**：

| 模式 | 入口 | 工具发现 | 适用场景 |
|---|---|---|---|
| **外部 AI 模式** | 用户在 Claude/ChatGPT 中配置 MCP Server 地址 | LLM 通过 `tools/list` 自动发现 | 用户已有 AI 助手，想通过它操作 Caddy |
| **内置 AI 模式** | 用户在系统 Web UI 中直接对话 | Eino Agent 通过 MCP Client 自动发现 | 用户不需要外部 AI，系统自带智能助手 |

> **关键设计**：无论哪种模式，工具逻辑和权限校验都只实现一次（在 MCP Server 中）。Eino Agent 作为 MCP Client 消费工具，与外部 Claude 作为 MCP Client 消费工具，走完全相同的代码路径。这是 Eino 独有 MCP 集成带来的架构优势。

### 后续 AI 能力扩展路径

| 阶段 | AI 能力 | 实现方式 |
|---|---|---|
| **MCP 阶段（W9-12）** | 工具暴露 | MCP Server 暴露 14 个工具，外部 AI 可调用 |
| **内置助手阶段（W13-16）** | 单轮对话 | Eino ReAct Agent + MCP Client，用户在 Web UI 中自然语言操作 |
| **多步编排阶段（W17-20）** | 复杂工作流 | Eino DeepAgent + Graph 编排，如"给所有不健康上游换新地址并通知" |
| **智能运维阶段（W21+）** | 自动化运维 | Eino Agent + 定时触发 + 告警联动，如"上游不健康时自动切换备份并创建工单" |

### 新增依赖

| 包 | 用途 | 版本 |
|---|---|---|
| `github.com/cloudwego/eino` | AI Agent 框架核心 | 最新 |
| `github.com/cloudwego/eino-ext` | 扩展组件（MCP 集成、LLM Provider 等） | 最新 |

---

## 8. 前端技术栈

> **定位**：管理控制台 Web UI，非核心交付物，MVP 阶段后开发

| 组件 | 选型 | 理由 |
|---|---|---|
| **框架** | React 18 + TypeScript | 生态最大，组件库丰富 |
| **构建工具** | Vite | 极快的 HMR，轻量配置 |
| **UI 组件库** | Ant Design 5.x | 企业级管理后台首选，表格/表单/树组件完备 |
| **状态管理** | Zustand | 轻量，无需 Redux 样板代码 |
| **数据请求** | TanStack Query (React Query) | 缓存、重试、乐观更新 |
| **路由** | React Router v6 | 标准选择 |
| **图表** | Recharts | 上游健康状态可视化 |
| **API 客户端** | openapi-typescript-codegen | 从后端 OpenAPI 自动生成类型化客户端 |

---

## 9. 基础设施与部署

| 组件 | 选型 | 说明 |
|---|---|---|
| **容器** | Docker + Alpine 基础镜像 | Go 静态编译二进制，镜像 < 20MB |
| **编排** | Docker Compose（MVP）→ Kubernetes（生产） | 渐进式部署 |
| **CI/CD** | GitHub Actions | Go 交叉编译 + Docker 构建 + 部署 |
| **配置管理** | 环境变量 + `.env` | 12-Factor App 原则 |
| **对象存储** | MinIO（自建）或 AWS S3 | 配置快照外部备份 |
| **反向代理** | Caddy 本身 | 控制面 API 也可由 Caddy 前置代理 |
| **数据库迁移** | Ent + Atlas | Ent schema 变更自动生成迁移文件，Atlas 执行版本管理 |

**Dockerfile 示例**：

```dockerfile
# 构建阶段
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /caddy-mgmt ./cmd/server

# 运行阶段
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1000 appuser
COPY --from=builder /caddy-mgmt /usr/local/bin/
USER appuser
EXPOSE 8080 8081
ENTRYPOINT ["caddy-mgmt"]
```

---

## 10. 完整技术栈总览

```
╔═══════════════════════════════════════════════════════════════╗
║                    完整技术栈架构图                               ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║  ┌─── AI 交互层 ──────────────────────────────────────────┐   ║
║  │  MCP Server: Go MCP SDK v1.6.1 (Tier 1)              │   ║
║  │  传输: HTTP+SSE / stdio                                │   ║
║  └────────────────────────────────────────────────────────┘   ║
║                                                               ║
║  ┌─── 控制面 (Go 单二进制) ───────────────────────────────┐    ║
║  │                                                        │   ║
║  │  Web 框架:      Gin v1.12.0+                          │   ║
║  │  MCP 框架:      go-sdk v1.6.1+                         │   ║
║  │  数据访问:      Ent (entgo.io)                          │   ║
║  │  认证:          golang-jwt/jwt v5.3.1+ (RS256)        │   ║
║  │  密码:          x/crypto/argon2                       │   ║
║  │  2FA:           pquerna/otp v1.5.0                     │   ║
║  │  OAuth:         x/oauth2                               │   ║
║  │  校验:          go-playground/validator v10.30.0+      │   ║
║  │  日志:          log/slog (Go 1.24+ 标准库)             │   ║
║  │  指标:          prometheus/client_golang v1.24.0+      │   ║
║  │  追踪:          OpenTelemetry Go SDK v1.44.0+         │   ║
║  │  Caddy 客户端:  net/http + crypto/tls (mTLS)          │   ║
║  │  Caddy 类型:    github.com/caddyserver/caddy/v2 (引用)  │   ║
║  │  迁移:          Ent + Atlas                            │   ║
║  │                                                        │   ║
║  └────────────────────────────────────────────────────────┘   ║
║                                                               ║
║  ┌─── 数据面 ─────────────────────────────────────────────┐   ║
║  │  Caddy v2.11.0+ (Go 编写, 原生 Admin API)             │   ║
║  │  mTLS: admin.remote :2021                              │   ║
║  └────────────────────────────────────────────────────────┘   ║
║                                                               ║
║  ┌─── 存储层 ─────────────────────────────────────────────┐   ║
║  │  数据库:    PostgreSQL 16+ (Ent 驱动)                  │   ║
║  │  缓存:      Redis 7.x (go-redis v9.21.0+)           │   ║
║  │  对象存储:  MinIO / AWS S3                             │   ║
║  └────────────────────────────────────────────────────────┘   ║
║                                                               ║
║  ┌─── 可观测性 ───────────────────────────────────────────┐   ║
║  │  指标:    Prometheus                                   │   ║
║  │  仪表盘:  Grafana                                      │   ║
║  │  追踪:    Jaeger / Tempo (OpenTelemetry)              │   ║
║  │  告警:    AlertManager                                  │   ║
║  └────────────────────────────────────────────────────────┘   ║
║                                                               ║
║  ┌─── 前端 (阶段 3) ──────────────────────────────────────┐   ║
║  │  React 18 + TypeScript + Vite                         │   ║
║  │  Ant Design 5.x + Zustand + React Query              │   ║
║  └────────────────────────────────────────────────────────┘   ║
║                                                               ║
║  ┌─── 基础设施 ───────────────────────────────────────────┐   ║
║  │  容器:    Docker (Alpine)                             │   ║
║  │  编排:    Docker Compose → Kubernetes                  │   ║
║  │  CI/CD:  GitHub Actions                                │   ║
║  └────────────────────────────────────────────────────────┘   ║
╚═══════════════════════════════════════════════════════════════╝
```

---

## 11. 项目工程结构

```
caddy-mgmt/
├── cmd/
│   ├── server/              # 控制面主服务入口
│   │   └── main.go
│   └── migrate/             # 数据库迁移工具入口
│       └── main.go
│
├── internal/
│   ├── api/                 # HTTP Handler 层 (Gin)
│   │   ├── router.go        # 路由注册
│   │   ├── middleware/      # 中间件
│   │   │   ├── auth.go      # JWT 认证
│   │   │   ├── rbac.go      # RBAC 授权
│   │   │   ├── ownership.go # @id 归属校验
│   │   │   ├── ratelimit.go # 速率限制
│   │   │   └── audit.go     # 审计记录
│   │   ├── handler/         # 请求处理器
│   │   │   ├── project.go
│   │   │   ├── domain.go
│   │   │   ├── proxy.go
│   │   │   ├── member.go
│   │   │   └── share.go
│   │   └── dto/             # 请求/响应 DTO
│   │
│   ├── mcp/                 # MCP Server 层 (go-sdk)
│   │   ├── server.go        # MCP Server 初始化
│   │   └── tools/           # MCP 工具实现
│   │       ├── caddy_domain.go
│   │       ├── caddy_proxy.go
│   │       ├── caddy_status.go
│   │       ├── project_crud.go
│   │       ├── project_member.go
│   │       └── project_audit.go
│   │
│   ├── agent/               # AI 智能助手层 (Eino)
│   │   ├── agent.go         # Eino ReAct Agent 初始化
│   │   ├── mcp_client.go    # MCP Client (消费自身工具)
│   │   ├── memory.go         # 对话记忆与上下文管理
│   │   └── middleware/       # Agent 中间件
│   │       ├── confirm.go   # Human-in-the-loop 确认
│   │       └── audit.go      # AI 操作审计记录
│   │
│   ├── service/             # 业务逻辑层 (框架无关)
│   │   ├── project_service.go
│   │   ├── domain_service.go
│   │   ├── proxy_service.go
│   │   ├── member_service.go
│   │   ├── share_service.go
│   │   └── audit_service.go
│   │
│   ├── repository/          # 数据访问层 (Ent 生成)
│   │   ├── querier.go       # 仓储接口封装
│   │   ├── ent/             # Ent 代码（schema + 生成代码 + 迁移）
│   │   │   ├── schema/      # Ent schema 定义（Go 代码）
│   │   │   ├── generated/   # Ent 自动生成的查询代码
│   │   │   └── migrate/     # Atlas 迁移文件（由 Ent 生成）
│   │   └── custom/         # 自定义复杂查询封装
│   │
│   ├── caddy/               # Caddy Admin API 客户端
│   │   ├── client.go        # HTTP 客户端 (mTLS)
│   │   ├── config.go        # JSON 配置翻译
│   │   ├── validator.go     # 配置校验
│   │   └── sync.go          # 配置同步与恢复
│   │
│   ├── auth/                # 认证与授权
│   │   ├── jwt.go           # JWT 签发与验证
│   │   ├── password.go      # Argon2id 密码
│   │   ├── totp.go          # 2FA TOTP
│   │   ├── oauth.go         # OAuth 2.0
│   │   └── mtls.go          # mTLS 证书管理
│   │
│   └── observability/      # 可观测性
│       ├── metrics.go       # Prometheus 指标
│       ├── tracing.go       # OpenTelemetry 追踪
│       └── logger.go        # slog 日志配置
│
├── migrations/              # Atlas 迁移 SQL 文件（由 Ent schema 生成）
│   ├── 001_create_users.up.sql
│   ├── 001_create_users.down.sql
│   ├── 002_create_projects.up.sql
│   └── ...
│
├── deployments/
│   ├── docker-compose.yml   # MVP 部署
│   ├── docker/
│   │   └── Dockerfile
│   └── k8s/                 # Kubernetes 部署 (生产)
│       ├── control-plane.yaml
│       ├── caddy.yaml
│       └── postgres.yaml
│
├── web/                     # 前端项目 (阶段 3)
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│
├── go.mod
├── go.sum
├── Makefile
└── ent/                    # Ent 代码生成入口
    ├── entc.go             # Ent 代码生成器入口
    └── generate.go         # go:generate 指令
```

---

## 12. 第三方依赖清单

### 12.1 Go 核心依赖

| 包 | 版本 | 用途 |
|---|---|---|
| `github.com/gin-gonic/gin` | v1.12.0+ | Web 框架 |
| `github.com/modelcontextprotocol/go-sdk` | v1.6.1+ | MCP Server SDK（以官方仓库最新发布为准） |
| `github.com/mark3labs/mcp-go` | 最新 | 统一实现 MCP Server 与 Client（Eino 默认依赖） |
| `github.com/cloudwego/eino` | 最新 | AI Agent 框架（ReAct Agent + 编排） |
| `github.com/cloudwego/eino-ext` | 最新 | Eino 扩展（MCP 集成 + LLM Provider） |
| `github.com/caddyserver/caddy/v2` | v2.11.0+ | Caddy 类型定义引用 |
| `entgo.io/ent` | 最新 | Ent ORM（schema 即代码、类型安全查询代码生成） |
| `ariga.io/atlas` | 最新 | Atlas 迁移引擎（与 Ent 集成，自动生成迁移文件） |
| `github.com/jackc/pgx/v5` | v5.10.0+ | PostgreSQL 驱动（Ent 底层使用） |
| `github.com/redis/go-redis/v9` | v9.21.0+ | Redis 客户端 |
| `github.com/golang-jwt/jwt/v5` | v5.3.1+ | JWT 签发与验证 |
| `github.com/go-playground/validator/v10` | v10.30.0+ | 输入校验 |
| `golang.org/x/crypto` | 最新 | Argon2id 密码哈希 |
| `golang.org/x/oauth2` | 最新 | OAuth 2.0 |
| `github.com/pquerna/otp` | v1.5.0 | TOTP 2FA |
| `github.com/prometheus/client_golang` | v1.24.0+ | Prometheus 指标 |
| `go.opentelemetry.io/otel` | v1.44.0+ | 链路追踪 |
| `github.com/santhosh-tekuri/jsonschema/v6` | 最新 | JSON Schema 校验 |
| `github.com/xeipuuv/gojsonschema` | v1.2+ | JSON Schema 校验（已停止维护，建议替换为 santhosh-tekuri/jsonschema/v6） |
| `github.com/google/uuid` | v1.6+ | UUID 生成 |
| `github.com/stretchr/testify` | 最新 | 单元测试断言 |
| `github.com/testcontainers/testcontainers-go` | 最新 | 集成/数据库测试容器 |
| `github.com/caarlos0/env/v11` | 最新 | 配置读取（环境变量） |
| `golang.org/x/time/rate` | 最新 | 限流器 |

### 12.2 开发工具链

| 工具 | 用途 |
|---|---|
| Go 1.24+ | 编译器（`log/slog` 需 1.21+，路由模式需 1.22+） |
| Ent CLI (`entgo.io/ent/cmd/ent`) | 代码生成（schema → 类型安全查询代码） |
| Atlas (`ariga.io/atlas/cmd/atlas`) | 数据库迁移（由 Ent schema 自动生成并应用） |
| golangci-lint | 代码静态检查 |
| Docker | 容器化 |
| Make | 构建/测试/迁移自动化 |

### 12.3 Makefile 核心目标

```makefile
.PHONY: all build test migrate generate docker

build:
	go build -o bin/caddy-mgmt ./cmd/server

test:
	go test -v -race -cover ./...

migrate-apply:
	atlas migrate apply --dir "file://ent/migrate/migrations" \
		--url "$(DATABASE_URL)"

migrate-diff:
	go run ent/entc.go --migrate-diff

generate:
	go run ent/entc.go

docker:
	docker build -t caddy-mgmt:latest -f deployments/docker/Dockerfile .

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server
```

---

## 附录: 选型决策记录 (ADR)

| ADR # | 决策 | 状态 | 日期 |
|---|---|---|---|
| ADR-001 | 控制面使用 Go 而非 Python | 已批准 | 2026-07-24 |
| ADR-002 | Web 框架选用 Gin | 已批准 | 2026-07-24 |
| ADR-003 | 数据访问使用 Ent (entgo.io) | 已批准 | 2026-07-24 |
| ADR-004 | MCP SDK 使用 Go 官方 SDK | 已批准 | 2026-07-24 |
| ADR-005 | 日志使用 slog 而非 logrus/zap | 已批准 | 2026-07-24 |
| ADR-006 | 前端使用 React + Ant Design | 已批准 | 2026-07-24 |
| ADR-007 | AI 框架选用 Eino（MCP 原生集成） | 已批准 | 2026-07-24 |
| ADR-008 | MCP SDK 兼容方案：官方 go-sdk 与 mark3labs/mcp-go 的选择 | 已批准 | 2026-07-26 |

