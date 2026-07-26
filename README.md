# Portunus

基于 Caddy 的多租户反向代理管理平台，提供项目级隔离的域名管理、反向代理配置、上游健康监控和 AI 辅助运维能力。

## 技术栈

| 层级 | 技术 | 版本 |
|---|---|---|
| 后端 | Go / Gin / Ent | Go 1.22+ / Gin v1.10.0+ |
| 前端 | React / Vite / Ant Design | React 18 / AntD 5.x |
| 数据面 | Caddy Admin API | v2.7+ |
| 数据库 | PostgreSQL + Redis + MinIO | PG 16+ / Redis 7+ |
| AI | Eino (CloudWeGo) + MCP SDK | — |

## 快速开始

### 前置条件

| 工具 | 最低版本 | 用途 |
|---|---|---|
| Go | 1.22+ | 后端编译运行 |
| Node.js | 20+ | 前端构建 |
| Docker & Docker Compose | 24+ / v2 | 本地依赖服务 |
| Caddy | 2.7+ | 数据面（Docker 内已包含） |
| Ent CLI | 最新 | Schema 代码生成（entgo.io） |
| Atlas | 最新 | 数据库迁移（与 Ent 集成） |
| `uv` | 0.4+ | 文档工具链（如需） |

### 1. 克隆项目

```bash
git clone <repo-url> portunus
cd portunus
```

### 2. 启动依赖服务

```bash
docker compose up -d postgres redis caddy minio
```

验证服务状态：

```bash
# PostgreSQL
docker compose exec postgres pg_isready

# Redis
docker compose exec redis redis-cli ping

# Caddy Admin API（mTLS，预期返回 401 或证书错误即为正常监听）
curl -k https://localhost:2021/config/

# MinIO
curl http://localhost:9000/minio/health/live
```

### 3. 初始化数据库

```bash
# 生成 Ent 代码（schema → 类型安全的查询代码）
go run ent/entc.go

# 执行数据库迁移（基于 Atlas）
make migrate-apply
```

### 4. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env，填写 JWT 密钥、Caddy mTLS 证书路径等
```

关键环境变量：

| 变量 | 说明 | 默认值 |
|---|---|---|
| `PORT` | 控制面监听端口 | `8080` |
| `DATABASE_URL` | PostgreSQL 连接串 | `postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable` |
| `REDIS_URL` | Redis 连接串 | `redis://localhost:6379/0` |
| `JWT_PRIVATE_KEY` | JWT 签名密钥（Ed25519） | — |
| `CADDY_ADMIN_URL` | Caddy Admin API 地址 | `https://caddy:2021` |
| `CADDY_MTLS_CERT` | mTLS 客户端证书路径 | `./certs/client.crt` |
| `CADDY_MTLS_KEY` | mTLS 客户端私钥路径 | `./certs/client.key` |
| `CADDY_MTLS_CA` | mTLS CA 证书路径 | `./certs/ca.crt` |

### 5. 启动后端

```bash
go run ./cmd/portunus
```

### 6. 启动前端

```bash
cd web
npm install
npm run dev
```

前端默认运行在 `http://localhost:5173`。

### 7. 生成 Caddy mTLS 证书（开发环境）

```bash
mkdir -p certs
# 生成 CA
openssl req -x509 -newkey rsa:4096 -keyout certs/ca.key -out certs/ca.crt -days 365 -nodes -subj "/CN=Portunus Dev CA"

# 生成服务端证书
openssl req -newkey rsa:4096 -keyout certs/server.key -out certs/server.csr -nodes -subj "/CN=caddy"
openssl x509 -req -in certs/server.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/server.crt -days 365

# 生成客户端证书
openssl req -newkey rsa:4096 -keyout certs/client.key -out certs/client.csr -nodes -subj "/CN=portunus-control"
openssl x509 -req -in certs/client.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/client.crt -days 365
```

## 项目结构

```
portunus/
├── cmd/
│   └── portunus/           # 程序入口
│       └── main.go
├── internal/                # 内部包（不可外部引用）
│   ├── api/                 # 接入层
│   │   ├── handler/         # HTTP Handler
│   │   ├── middleware/      # Gin 中间件
│   │   └── router.go        # 路由注册
│   ├── service/             # 业务逻辑层
│   │   ├── domain.go
│   │   ├── project.go
│   │   ├── proxy.go
│   │   └── ...
│   ├── repository/          # 数据访问层（Ent 生成）
│   │   ├── ent/schema/      # Ent schema 定义（Go 代码）
│   │   ├── ent/generated/   # Ent 自动生成的 Go 代码
│   │   └── ent/migrate/     # Atlas 迁移文件
│   ├── caddy/               # Caddy Admin API 客户端
│   │   └── client.go
│   ├── mcp/                 # MCP Server（AI 工具）
│   │   ├── server.go
│   │   └── tools/           # 14 个 MCP 工具定义
│   ├── ai/                  # Eino AI Agent
│   │   └── agent.go
│   ├── model/               # 领域模型
│   ├── config/              # 配置加载
│   └── scheduler/           # 定时任务
├── web/                     # 前端
│   ├── src/
│   │   ├── pages/          # 页面
│   │   ├── components/     # 通用组件
│   │   ├── stores/         # Zustand 状态
│   │   ├── api/            # API 客户端（自动生成）
│   │   └── hooks/          # React Hooks
│   ├── package.json
│   └── vite.config.ts
├── db/
│   └── migrations/          # Atlas 迁移文件（由 Ent 生成）
├── docs/                    # 项目文档
│   ├── caddy-multi-tenant-architecture.md
│   ├── caddy-tech-stack-selection.md
│   ├── caddy-system-architecture.md
│   ├── caddy-state-transitions.md
│   ├── product-requirements.md
│   ├── test-strategy.md
│   └── openapi.yaml
├── deployments/
│   └── docker-compose.yml
├── .env.example
├── Makefile
├── go.mod
└── go.sum
```

## 开发工作流

### Make 命令

```bash
make dev          # 启动后端（热重载）
make dev-web      # 启动前端
make test         # 运行所有测试
make test-unit    # 单元测试
make test-integration  # 集成测试（需要 Docker）
make lint         # 代码检查
make generate     # Ent 代码生成
make migrate-apply # 执行数据库迁移（基于 Atlas）
make build        # 编译生产二进制
make docker-build # 构建 Docker 镜像
```

### 代码生成

修改 Ent schema（`ent/schema/*.go`）后重新生成类型安全的查询代码：

```bash
go run ent/entc.go
# 或
make generate
```

修改 OpenAPI 规范后重新生成前端 API 客户端：

```bash
npx openapi-typescript-codegen --input docs/openapi.yaml --output web/src/api/generated
```

### 数据库迁移

迁移文件由 Ent + Atlas 自动生成并管理：

```bash
# 生成迁移差异（基于 schema 变更）
go run ent/entc.go --migrate-diff

# 执行迁移
make migrate-apply
```

## 测试

详见 [测试策略文档](docs/test-strategy.md)。

```bash
# 单元测试（覆盖率）
go test -cover ./internal/...

# 集成测试（需要 Docker 依赖）
go test -tags=integration ./test/integration/...

# 前端测试
cd web && npm test
```

## 文档

| 文档 | 说明 |
|---|---|
| [架构设计](docs/caddy-multi-tenant-architecture.md) | 多租户架构、数据模型、MCP 工具、安全策略 |
| [技术栈选型](docs/caddy-tech-stack-selection.md) | Go/Gin/Ent/Eino 选型论证 |
| [系统架构](docs/caddy-system-architecture.md) | 前后端分层、代码示例、部署方案 |
| [状态流转](docs/caddy-state-transitions.md) | 9 类实体的状态机和转换守卫 |
| [产品需求](docs/product-requirements.md) | PRD、用户画像、功能需求 |
| [安全威胁模型](docs/security-threat-model.md) | 威胁建模、安全控制矩阵 |
| [测试策略](docs/test-strategy.md) | 测试金字塔、测试矩阵 |
| [API 规范](docs/openapi.yaml) | OpenAPI 3.1 接口定义 |

## 贡献

1. 创建功能分支：`git checkout -b feat/your-feature`
2. 提交信息格式：`<type>: <简要描述>`（如 `feat: 添加域名批量导入`）
3. 提交 PR 前确保：`make lint && make test`
4. 代码审查通过后合并

## 许可证

待定。
