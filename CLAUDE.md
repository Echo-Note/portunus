# CLAUDE.md

此文件为 Claude Code（claude.ai/code）提供在此仓库中工作的指导。

## 项目概述

Portunus 是一个多租户 Caddy 反向代理管理平台（Go）。提供**控制面**（Web UI + REST API + MCP Server），通过 Caddy Admin API 管理**数据面**（Caddy 实例）。多租户隔离基于 Caddy 的 `@id` 机制实现——每个租户的配置节点标记为 `@id: "tenant_<project_id>_<type>_<id>"`，通过 `/id/<name>/...` 端点精确寻址。

**关键架构决策**：控制面是独立的 Go 服务，**不是** Caddy 插件/模块。经深度研究后确认——Caddy 的配置 reload 会 Stop/Start 整个 App 模块，导致数据库连接池和 REST/MCP Server 在每次域名变更时中断。

## 常用命令

```bash
# 开发
make dev                    # 后端热重载（air）
make dev-web                # 前端开发服务器
make up                     # 启动 PostgreSQL + Redis + Caddy + MinIO（Docker）

# 代码生成（修改 ent/schema/*.go 后执行）
make generate               # Ent 代码生成：Go schema → 类型安全查询代码

# 数据库迁移
make migrate-apply          # 应用 Atlas 迁移到数据库
make migrate-diff NAME=xxx  # 从 schema 变更生成增量迁移文件
make migrate-status         # 查看迁移状态
make migrate-lint           # CI 检查迁移是否有破坏性变更

# 测试
make test                   # 全部单元测试（race + coverage）
make test-integration       # 集成测试（需要 Docker）
make coverage               # HTML 覆盖率报告
make lint                   # golangci-lint + go vet + 前端 lint

# 构建
make build                  # 编译生产二进制
make docker-build           # 构建 Docker 镜像
```

## 架构

### 项目结构

```
portunus/
├── cmd/portunus/main.go            # 程序入口
├── internal/                       # 私有包（不可外部引用）
│   ├── api/                        # 接入层
│   │   ├── handler/                # HTTP Handler（薄层，只做参数绑定和响应）
│   │   ├── middleware/             # 中间件（Auth/RBAC/Ownership/RateLimit/Audit）
│   │   ├── dto/                    # 请求/响应 DTO
│   │   └── router.go               # 路由注册
│   ├── service/                    # 业务逻辑层（框架无关）
│   ├── caddy/                      # Caddy Admin API 客户端（mTLS）
│   ├── mcp/                        # MCP Server
│   │   └── tools/                  #  14 个 MCP 工具定义
│   ├── ai/                         # Eino AI Agent
│   ├── config/                     # 配置加载（12-Factor）
│   └── scheduler/                  # 定时任务（漂移检测、过期清理）
├── ent/                            # Ent ORM 代码生成（独立于 internal/）
│   ├── schema/                     # 模型定义（真相源，Go 代码）
│   ├── generated/                  # 自动生成（make generate）
│   └── migrate/migrations/         # Atlas 迁移文件（make migrate-diff）
├── web/                            # 前端（React SPA）
│   ├── src/
│   │   ├── pages/                  # 页面组件
│   │   ├── components/             # 通用组件
│   │   │   ├── common/             # 布局/导航/表单等
│   │   │   ├── domain/             # 域名相关组件
│   │   │   └── project/            # 项目相关组件
│   │   ├── hooks/                  # 自定义 Hooks
│   │   ├── stores/                 # Zustand 状态
│   │   └── api/                    # API 客户端
│   ├── public/
│   └── package.json
├── deployments/                    # 部署配置
│   ├── docker/Dockerfile
│   └── docker-compose.yml
├── docs/                           # 项目文档
├── .editorconfig                   # 编辑器统一配置
├── .golangci.yml                   # Go Lint 配置
├── .gitignore
├── .env.example
├── atlas.hcl                       # Atlas 迁移配置
├── Makefile
├── go.mod
└── go.sum
```

### 分层架构（洋葱架构变体）

```
HTTP Handler / MCP / AI Agent  ← 接入层：参数绑定、上下文提取、错误映射
        │
Middleware (Auth → RBAC → Ownership → RateLimit → Audit)  ← 横切关注点
        │
Service  ← 业务逻辑层：框架无关，接收 context.Context + 普通结构体
        │
┌───────┴───────┐
│               │
Ent Client    Caddy Client  ← 基础设施层
(PostgreSQL)  (mTLS Admin API)
```

**分层铁律**：
- Handler 层**禁止**将 `gin.Context` 传入 Service 层，只能传 `context.Context` 和普通参数
- Service 层不 import `gin`，同一方法可被 REST Handler、MCP Tool、Eino Agent 三方调用
- Service 依赖 `*ent.Client`，通过构造函数注入

### Caddy Admin API 协议

| 操作 | 端点 | 说明 |
|------|------|------|
| **日常变更** | `POST\|PATCH\|DELETE /id/<name>/...` + `If-Match` 头 | 细粒度更新，乐观并发控制（Etag） |
| **灾难恢复** | `POST /load` | **仅用于**初始加载、灾难恢复、全量回滚。**禁止**用作预检/沙箱校验 |
| **TLS 控制** | `ssl_enabled=true` → 依赖 Caddy 自动 HTTPS；`ssl_enabled=false` → 配置 `auto_https` 禁用 |
| **健康检查** | `GET /reverse_proxy/upstreams` | 返回**全局**上游池，控制面必须按 dial address 过滤，不能按域名过滤 |

### @id 命名规范

```
tenant_<project_id>_<resource_type>_<resource_id>

约束：
  · project_id 字符集: [a-zA-Z0-9_-]
  · @id 校验正则: ^[a-zA-Z0-9_-]{1,128}$

示例:
  tenant_abc_route_001     → 路由匹配规则节点
  tenant_abc_proxy_001     → 反向代理处理器节点
  tenant_abc_upstreams_001 → 上游地址池节点
```

### RBAC 统一权限矩阵

适用于 REST API、MCP 工具、AI Agent 三层：

| 角色 | 域名 CRUD | 代理配置 | 上游管理 | 成员管理 | 项目设置 |
|------|:---:|:---:|:---:|:---:|:---:|
| **owner** | 全部 | 全部 | 全部 | 全部 | 全部 |
| **admin** | 全部 | 全部 | 全部 | 除 owner | 只读 |
| **editor** | 创建+更新 | 更新 | 添加 | 无 | 无 |
| **viewer** | 只读 | 只读 | 只读 | 无 | 无 |

### 状态机关键规则

- **Domain**：`creating → active`（Caddy 路由创建成功）或 `creating → error`（由控制面补偿，**不依赖** Caddy 自动回滚）
- **Upstream**：包含 `unhealthy` 状态，由 Caddy 被动健康检查自动驱动
- **ProxyConfig**：`active / updating / degraded / unavailable`
- **Project**：删除为异步流程（标记 `deleting` → 清理 Caddy @id 节点 → 标记 `deleted`）
- 所有状态转换通过 `ExecuteTransition()` 统一执行，经守卫映射校验 → 乐观锁更新 DB（`WHERE status = $from`）→ Caddy 副作用先于 DB 提交

## 代码规范

### Go 后端

**命名规范**：
```go
// 包名：小写单数，简短有意义
package service       // ✅
package services      // ❌ 不用复数
package domain_svc    // ❌ 不用下划线

// 接口：单方法接口以 -er 结尾
type Validator interface { Validate() error }
// 多方法接口：描述性名称
type DomainRepository interface { ... }

// 变量/参数：驼峰命名，缩写全大写或全小写
var projectID string   // ✅
var projectId string   // ❌
var httpClient *http.Client  // ✅
var HTTPClient *http.Client  // ❌ 导出才大写
```

**文件组织**：
- 一个文件对应一个核心类型，文件名与类型名一致（小写）
- 测试文件与被测文件同目录，`xxx_test.go`
- 表驱动测试优先，使用 `testify/assert` + `testify/require`

**错误处理**：
```go
// ✅ 使用 fmt.Errorf 包装上下文
if err != nil {
    return fmt.Errorf("domain create: caddy route: %w", err)
}

// ❌ 不要吞掉错误
caddyClient.CreateRoute(ctx, config)  // 忽略返回值

// ✅ 编译期确保错误类型
var ErrConcurrentModification = errors.New("concurrent modification detected")
```

**Context 传播**：
```go
// ✅ 所有 IO 操作必须接收 context.Context
func (s *DomainService) Create(ctx context.Context, input *CreateDomainInput) (*Domain, error)

// ❌ 不要用 context.Background() 在业务代码中，从调用方传入
```

**Ent 查询模式**：
```go
// ✅ 链式查询（类型安全）
domains, err := client.Domain.Query().
    Where(domain.ProjectIDEQ(projectID), domain.StatusEQ("active")).
    WithProxyConfig(func(q *ent.ProxyConfigQuery) {
        q.WithUpstreams()
    }).
    All(ctx)

// ✅ 复杂查询回退原生 SQL
rows, err := client.QueryContext(ctx, `
    SELECT d.domain_name, COUNT(u.id) FROM domains d
    LEFT JOIN upstreams u ON ...
    GROUP BY d.domain_name
`)
```

**Ent Schema 约定**：
- `created_at`：`.Default(time.Now).Immutable()`
- `updated_at`：`.Default(time.Now).UpdateDefault(time.Now)`
- 外键字段显式声明在 `Fields()` 中，并在 `Edges()` 中通过 `.Field("xxx")` 关联
- PostgreSQL 数组：`field.Ints()` / `field.Strings()`
- PostgreSQL JSONB：`field.JSON("name", &SomeType{})` 或 `field.JSON("name", map[string]any{})`
- INET 列：使用 `field.String`（Ent 无原生 INET 类型）
- BIGSERIAL PK：`field.Int64("id").StorageKey("id").SchemaType(map[string]string{"postgres": "BIGSERIAL"}).Immutable()`

**格式化**：
- 使用 `gofmt` 或 `goimports` 自动格式化
- CI 中 `golangci-lint` 检查，配置见 `.golangci.yml`，**零警告**方可通过
- 编辑器统一配置见 `.editorconfig`

**注释规范**：
```go
// ✅ 所有导出的函数、类型、变量必须有文档注释，以声明名称开头
// NewDomainService 创建域名服务实例，注入所需的 Ent Client 和 Caddy Client。
func NewDomainService(client *ent.Client, caddy *caddy.Client) *DomainService { ... }

// DomainService 处理域名相关的所有业务逻辑，包括创建、更新、删除和共享。
type DomainService struct { ... }

// ErrDomainExists 当尝试创建已存在的域名时返回。
var ErrDomainExists = errors.New("domain already exists")

// ✅ 包级注释：每个包的第一个文件（通常是 doc.go）包含包概述
// Package service 提供业务逻辑层，所有方法框架无关，
// 可被 REST Handler、MCP Tool、AI Agent 三方调用。

// ✅ 复杂逻辑注释"为什么"而非"做什么"
// 先下发 Caddy 配置再提交 DB 事务：确保 Caddy 成功后再持久化，
// 如果 Caddy 失败则 DB 状态不变，避免不一致。
// 这与状态机文档中"Caddy 同步优先"原则一致。

// ✅ TODO 注释格式：TODO(owner): 描述
// TODO(zhangsan): 阶段 2 替换为 Redis 分布式锁

// ❌ 不要写废话注释
// SetName 设置名称  ← 废话
// SetName 设置项目的显示名称，长度限制 255 字符。  ← 有用
```

### TypeScript 前端

**命名规范**：
```typescript
// 文件名：kebab-case
user-profile.tsx        // ✅
UserProfile.tsx         // ❌

// 组件：PascalCase
export function DomainList() { ... }  // ✅

// Hooks：use 前缀
export function useDomains(projectId: string) { ... }

// Store：use 前缀 + Store 后缀
export const useAuthStore = create<AuthState>(...)
```

**组件结构**：
```typescript
// src/components/domain/DomainForm.tsx
// 文件内顺序：imports → types → component → styles（如有）

import { useState } from 'react';
import { Form, Input, Button } from 'antd';

interface DomainFormProps {
  projectId: string;
  onSubmit: (values: DomainFormValues) => Promise<void>;
}

export function DomainForm({ projectId, onSubmit }: DomainFormProps) {
  // 1. hooks
  // 2. state
  // 3. effects
  // 4. handlers
  // 5. render
  return <Form>...</Form>;
}
```

**API 调用**：
```typescript
// ✅ 使用 TanStack Query（缓存 + 重试 + 乐观更新）
export function useDomains(projectId: string) {
  return useQuery({
    queryKey: ['domains', projectId],
    queryFn: () => api.getDomains(projectId),
    staleTime: 30 * 1000,
  });
}

// ✅ 乐观更新模式
const mutation = useMutation({
  mutationFn: api.createDomain,
  onMutate: async (input) => { /* 立即更新缓存 */ },
  onError: (_err, _input, ctx) => { /* 回滚缓存 */ },
  onSettled: () => { /* 刷新列表 */ },
});
```

**状态管理**：
- 全局状态使用 Zustand（`useAuthStore`、`useUIStore`）
- 服务端数据使用 TanStack Query（自动缓存、去重请求）
- 组件本地状态使用 `useState` / `useReducer`

**格式化**：
- 使用 Prettier + ESLint 自动格式化
- CI 中 `npm run lint` 检查，**零警告**方可通过

**注释规范**：
```typescript
// ✅ 组件和导出函数使用 JSDoc 注释
/**
 * 域名列表页面，展示当前项目下所有域名及其健康状态。
 * 支持按状态筛选和按名称搜索。
 */
export function DomainListPage() { ... }

// ✅ Props 接口注释
interface DomainFormProps {
  /** 项目 ID，用于创建域名时关联项目 */
  projectId: string;
  /** 创建成功后的回调 */
  onSubmit: (values: DomainFormValues) => Promise<void>;
}

// ✅ 复杂逻辑注释"为什么"
// 使用乐观更新而非等待服务端响应：
// 用户感知的创建延迟从 ~500ms 降到 ~0ms

// ✅ TODO 注释格式：TODO(owner): 描述
// TODO(zhangsan): 阶段 2 添加批量导入功能
```

### 通用规范

**Git 提交信息**：
```
<type>: <简要描述>

feat: 添加域名批量导入
fix: 修复上游健康检查轮询空指针
docs: 更新 API 文档
refactor: 重构状态转换守卫
test: 补充域名创建失败回滚测试
```

**分支命名**：
```
feat/domain-batch-import
fix/health-check-nil-pointer
docs/api-update
```

**代码审查门禁**（合并前必须通过）：
- `make lint` 零错误
- `make test` 全部通过
- `make migrate-lint` 无破坏性变更
- 覆盖率不低于 80%

## 关键设计决策

1. **Ent 而非 sqlc**：Ent 提供 Django 式的"定义 Go 模型 → 自动生成一切"体验。sqlc 在业务扩展时 SQL 文件数量线性增长，动态查询需要额外的 squirrel+pgx 系统，且无内置迁移管理。
2. **Atlas 而非 golang-migrate**：Atlas 从 Ent schema 变更自动生成增量迁移，替代手写 `ALTER TABLE` SQL。
3. **JWT RS256**：access_token 15 分钟 + refresh_token 7 天。access_token 存储在内存中（禁止 localStorage），refresh_token 存储在 HttpOnly Cookie。
4. **`POST /load` 仅用于灾难恢复**：不会用作日常配置更新的预检/沙箱。日常变更走 `/id/` 或 `/config/` 路径 + `If-Match`。
5. **域名全局唯一**：防止同一 Caddy 实例上 host 冲突（不同项目不能配置相同域名）。

## 文档索引

| 文档 | 用途 |
|------|------|
| `docs/caddy-multi-tenant-architecture.md` | 系统架构、数据模型、MCP 工具、安全策略 |
| `docs/caddy-system-architecture.md` | 前后端分层架构、代码示例、部署方案 |
| `docs/caddy-tech-stack-selection.md` | Go/Gin/Ent/Eino 选型论证、ADR |
| `docs/caddy-state-transitions.md` | 9 类实体状态机、转换守卫、定时任务 |
| `docs/product-requirements.md` | PRD、用户故事、验收标准 |
| `docs/security-threat-model.md` | STRIDE 分析、OWASP 映射、安全配置清单 |
| `docs/test-strategy.md` | 测试金字塔、状态机测试矩阵、CI 流水线 |
| `docs/openapi.yaml` | OpenAPI 3.1 规范（49 端点含 operationId） |