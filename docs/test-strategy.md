# 测试策略文档

> 版本: v1.0 | 日期: 2026-07-25
>
> 关联文档: [系统架构](caddy-system-architecture.md) | [状态流转](caddy-state-transitions.md) | [架构设计](caddy-multi-tenant-architecture.md)

## 1. 测试金字塔

```
           ╱╲
          ╱E2╲          ← E2E 测试（5%）
         ╱  E  ╲           关键用户路径，Playwright
        ╱──────╲
       ╱        ╲
      ╱ 集成测试 ╲       ← 集成测试（20%）
     ╱            ╲        testcontainers + Docker
    ╱──────────────╲
   ╱                ╲
  ╱    单元测试       ╲    ← 单元测试（75%）
 ╱                    ╲     go test, 纯函数, mock 依赖
╱______________________╲
```

| 层级 | 占比 | 工具 | 运行时机 |
|---|:---:|---|---|
| 单元测试 | 75% | Go `testing`、`testify`、`gomock` | 每次 `go test` |
| 集成测试 | 20% | `testcontainers-go`、Docker Compose | CI 流水线 |
| E2E 测试 | 5% | Playwright（前端）、`httpexpect`（API） | 发版前 |

### 覆盖率目标

| 包 | 覆盖率目标 | 说明 |
|---|:---:|---|
| `internal/service/` | ≥ 85% | 核心业务逻辑 |
| `internal/caddy/` | ≥ 75% | Caddy API 客户端 |
| `ent/generated/` | — | Ent 自动生成的代码不考核覆盖率 |
| `internal/repository/` (自定义封装) | ≥ 80% | Repository 层自定义封装逻辑 |
| `internal/api/handler/` | ≥ 80% | HTTP Handler |
| `internal/api/middleware/` | ≥ 85% | 认证/鉴权中间件 |
| `internal/mcp/` | ≥ 75% | MCP 工具 |
| 总体 | ≥ 80% | — |

---

## 2. 单元测试

### 2.1 测试规范

- **文件命名**：`xxx_test.go`，与被测文件同目录
- **函数命名**：`Test<Type>_<Method>_<Scenario>`，如 `TestDomainService_Create_Success`
- **表驱动测试**：优先使用表驱动模式
- **Mock 依赖**：使用 `gomock` 生成接口 Mock
- **断言**：使用 `testify/assert` 和 `testify/require`

### 2.2 单元测试模板

```go
func TestDomainService_Create(t *testing.T) {
    tests := []struct {
        name      string
        input     *dto.CreateDomainRequest
        mockSetup func(m *mocks.MockDomainRepo, c *mocks.MockCaddyClient)
        wantErr   bool
        errMsg    string
    }{
        {
            name: "成功创建域名",
            input: &dto.CreateDomainRequest{
                DomainName: "example.com",
                Upstreams:   []string{"10.0.0.1:8080"},
            },
            mockSetup: func(m *mocks.MockDomainRepo, c *mocks.MockCaddyClient) {
                m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
                c.EXPECT().CreateRoute(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
            },
            wantErr: false,
        },
        {
            name: "域名已存在",
            input: &dto.CreateDomainRequest{DomainName: "exists.com"},
            mockSetup: func(m *mocks.MockDomainRepo, c *mocks.MockCaddyClient) {
                m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(ErrDomainExists)
            },
            wantErr: true,
            errMsg: "domain already exists",
        },
        {
            name: "Caddy API 失败时回滚",
            input: &dto.CreateDomainRequest{DomainName: "new.com"},
            mockSetup: func(m *mocks.MockDomainRepo, c *mocks.MockCaddyClient) {
                m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
                c.EXPECT().CreateRoute(gomock.Any(), gomock.Any(), gomock.Any()).Return(ErrCaddyUnavailable)
                m.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil) // 回滚
            },
            wantErr: true,
            errMsg: "caddy api failed",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ... 执行测试
        })
    }
}
```

### 2.3 必测清单

| 模块 | 测试重点 | 场景数 |
|---|---|:---:|
| `DomainService` | 创建/更新/删除/共享，含 Caddy 失败回滚 | ≥ 12 |
| `ProjectService` | 创建/冻结/删除级联，配额校验 | ≥ 8 |
| `ProxyService` | 上游增删改、负载均衡策略切换 | ≥ 10 |
| `AuthService` | 注册/登录/刷新/邀请、OAuth 流程 | ≥ 10 |
| `MemberService` | 角色变更/移除/邀请接受、owner 转移 | ≥ 8 |
| `CaddyClient` | CreateRoute/DeleteRoute/UpdateRoute、Etag 冲突、超时 | ≥ 15 |
| `ExecuteTransition` | 合法/非法转换、乐观锁冲突、外部 IO 失败补偿 | ≥ 12 |
| 中间件 | Auth/ProjectContext/RequireRole/OwnershipCheck | ≥ 10 |

---

## 3. 集成测试

### 3.1 测试环境

使用 `testcontainers-go` 启动真实的 PostgreSQL、Redis、Caddy 实例：

```go
// test/integration/setup_test.go
func TestMain(m *testing.M) {
    ctx := context.Background()

    // 创建隔离网络
    network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
        NetworkRequest: testcontainers.NetworkRequest{Name: "portunus-test"},
    })
    if err != nil {
        log.Fatalf("create network: %v", err)
    }
    defer func() {
        if err := network.Remove(ctx); err != nil {
            log.Printf("remove network: %v", err)
        }
    }()

    // 启动 PostgreSQL 容器
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("portunus_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).WithStartupTimeout(30*time.Second)),
    )
    if err != nil {
        log.Fatalf("start postgres container: %v", err)
    }
    defer func() {
        if err := pgContainer.Terminate(ctx); err != nil {
            log.Printf("terminate postgres: %v", err)
        }
    }()

    // 启动 Caddy 容器（配置 mTLS）
    caddyContainer, err := caddy.RunContainer(ctx, ...)
    if err != nil {
        log.Fatalf("start caddy container: %v", err)
    }
    defer func() {
        if err := caddyContainer.Terminate(ctx); err != nil {
            log.Printf("terminate caddy: %v", err)
        }
    }()

    // 执行迁移
    if err := migrateUp(pgContainer); err != nil {
        log.Fatalf("migrate up: %v", err)
    }

    os.Exit(m.Run())
}
```

### 3.2 集成测试场景

| # | 场景 | 涉及组件 | 验证点 |
|---|---|---|---|
| I1 | 创建域名 → Caddy 配置生效 → 查询验证 | Service + Caddy + DB | DB 状态与 Caddy 运行态一致 |
| I2 | 更新代理 → Caddy PATCH → 上游健康检查 | Service + Caddy | Caddy @id 路径正确 |
| I3 | 删除域名 → Caddy 清理 → 映射表清理 | Service + Caddy + DB | 无残留 @id |
| I4 | 并发更新 → Etag 冲突 → 412 返回 | 两个并发请求 | 乐观锁生效 |
| I5 | 项目删除 → 级联清理所有域名 | Service + Caddy | 所有 @id 被清理 |
| I6 | 共享域名 → 目标项目可见 → 权限隔离 | Service + DB | 跨项目 @id 访问控制 |
| I7 | MCP 工具调用 → Caddy 配置变更 | MCP Server + Caddy | 工具 Schema 校验 |
| I8 | 配置漂移检测 → 定时任务 → 告警 | Scheduler + Caddy | checksum 比对正确 |
| I9 | 用户注册 → 邮箱验证 → 登录 → Token 刷新 | Auth 全链路 | JWT 生命周期正确 |
| I10 | API Token 创建 → 鉴权 → 撤销 | Auth + Middleware | Token 哈希验证 |
| I11 | 邀请成员 → 接受 → 角色变更 | Member + Invitation | 邀请 Token 过期 |
| I12 | Caddy 不可达 → 状态机 → 恢复 | Caddy + State | 补偿机制触发 |

---

## 4. 状态机测试矩阵

### 4.1 状态转换测试矩阵

对每个实体的每个 `(from_state, to_state)` 组合测试：

以下统计不含自转换（self-transition），所有转换均需通过守卫函数校验。

| 实体 | 状态数 | 合法转换数 | 非法转换数 | 总测试数 |
|---|:---:|:---:|:---:|:---:|
| 用户 | 4 | 5 | 7 | 12 |
| 项目 | 6 | 8 | 22 | 30 |
| 域名 | 7 | 15 | 27 | 42 |
| 代理 | 5 | 6 | 14 | 20 |
| 上游 | 4 | 4 | 8 | 12 |
| 共享 | 5 | 5 | 15 | 20 |
| 成员 | 4 | 5 | 7 | 12 |
| 邀请 | 4 | 4 | 8 | 12 |
| 配置同步 | 4 | 4 | 8 | 12 |
| **合计** | — | — | — | **182** |

### 4.2 状态机测试模板

```go
func TestStateTransition_Domain(t *testing.T) {
    legalTransitions := []struct {
        from, to string
    }{
        {"creating", "active"},
        {"creating", "error"},
        {"active", "updating"},
        {"active", "disabled"},
        {"active", "deleting"},
        {"active", "error"},
        {"updating", "active"},
        {"updating", "error"},
        {"error", "active"},
        {"error", "creating"},
        {"error", "deleting"},
        {"disabled", "active"},
        {"disabled", "deleting"},
        {"deleting", "deleted"},
        {"deleting", "error"},
    }

    illegalTransitions := []struct {
        from, to string
    }{
        {"creating", "disabled"},    // 不能直接跳过 active
        {"active", "creating"},       // 不能回到 creating
        {"deleted", "active"},        // 终态不可恢复
        {"deleted", "deleting"},      // 终态不可转换
        {"error", "disabled"},        // error 不能直接到 disabled
        {"error", "updating"},        // error 不能直接到 updating
        // ... 其余非法转换
    }

    for _, tt := range legalTransitions {
        t.Run("legal_"+tt.from+"_to_"+tt.to, func(t *testing.T) {
            err := ExecuteTransition(ctx, &StateTransition{
                EntityType: "domain",
                FromState:  tt.from,
                ToState:    tt.to,
            })
            assert.NoError(t, err)
        })
    }

    for _, tt := range illegalTransitions {
        t.Run("illegal_"+tt.from+"_to_"+tt.to, func(t *testing.T) {
            err := ExecuteTransition(ctx, &StateTransition{
                EntityType: "domain",
                FromState:  tt.from,
                ToState:    tt.to,
            })
            assert.ErrorIs(t, err, ErrIllegalTransition)
        })
    }
}
```

### 4.3 并发状态转换测试

```go
func TestConcurrentTransition_OptimisticLock(t *testing.T) {
    // 两个 goroutine 同时尝试将域名从 active 转为 updating
    var wg sync.WaitGroup
    var successCount, failCount int32

    for i := 0; i < 2; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            err := ExecuteTransition(ctx, &StateTransition{
                EntityType: "domain",
                EntityID:   domainID,
                FromState:  "active",
                ToState:    "updating",
            })
            if err == nil {
                atomic.AddInt32(&successCount, 1)
            } else if errors.Is(err, ErrConcurrentModification) {
                atomic.AddInt32(&failCount, 1)
            }
        }()
    }
    wg.Wait()

    assert.Equal(t, int32(1), successCount)  // 只有一个成功
    assert.Equal(t, int32(1), failCount)    // 另一个被乐观锁拒绝
}
```

---

## 5. E2E 测试

### 5.1 API E2E 测试（`httpexpect`）

覆盖完整的用户旅程：

| # | 用户旅程 | 步骤 |
|---|---|---|
| E1 | 新用户注册到域名上线 | 注册→验证邮箱→创建项目→添加域名→配置代理→验证 Caddy 生效 |
| E2 | 团队协作 | owner 邀请 editor→editor 接受→editor 配置代理→owner 审核 |
| E3 | 域名共享 | 项目 A 共享域名→项目 B 接受→项目 B 可见→撤销共享 |
| E4 | 项目生命周期 | 创建→正常使用→冻结→解冻→删除→验证清理 |
| E5 | AI 辅助运维 | 用户对话→AI 调用 MCP 工具→确认操作→配置生效 |
| E6 | API Token 工作流 | 创建 Token→使用 Token 访问→撤销→访问被拒 |
| E7 | Caddy 故障恢复 | Caddy 不可达→状态标记→恢复→漂移检测→自动修复 |

### 5.2 前端 E2E（Playwright）

```typescript
// e2e/domain-management.spec.ts
test('创建域名并配置反向代理', async ({ page }) => {
    await page.goto('/projects/1/domains');
    await page.click('button:has-text("添加域名")');
    await page.fill('input[name="domainName"]', 'test.example.com');
    await page.fill('input[name="upstream"]', '10.0.0.1:8080');
    await page.click('button:has-text("确认")');

    // 验证域名列表
    await expect(page.locator('text=test.example.com')).toBeVisible();

    // 验证状态为 active
    await expect(page.locator('.status-badge')).toHaveText('运行中');
});
```

---

## 6. 性能测试

### 6.1 基准测试（Go Benchmarks）

```bash
go test -bench=. -benchmem ./internal/service/...
```

关键基准：

| 基准 | 目标 | 说明 |
|---|---|---|
| `BenchmarkDomainService_Create` (in-memory) | < 1ms/op | 仅 Service 层逻辑，DB/Caddy 均 mock |
| `BenchmarkDomainService_Create` (end-to-end) | < 200ms/op | 含真实 DB + Caddy Mock |
| `BenchmarkDomainService_Get` | < 1ms/op | 含缓存命中 |
| `BenchmarkCaddyClient_CreateRoute` | < 50ms/op | 含 mTLS 握手 |
| `BenchmarkExecuteTransition` | < 2ms/op | 纯数据库操作 |
| `BenchmarkMiddleware_AuthChain` | < 0.1ms/op | JWT 验证链 |

基础设施假设：

- 测试实例：4 核 CPU / 8 GB 内存 / SSD 存储；
- PostgreSQL 连接池大小：25；
- Caddy Mock 与测试进程同机部署，网络延迟 < 1ms；
- 单次测试数据集规模：项目 1 个、域名 10 条、代理规则 20 条。

### 6.2 负载测试（k6）

```javascript
// k6/load-test.js
export default function () {
    const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
        email: 'test@example.com',
        password: 'password',
    }), { headers: { 'Content-Type': 'application/json' } });

    const token = res.json('data.access_token');

    // 创建域名
    http.post(`${BASE_URL}/api/v1/projects/1/domains`, JSON.stringify({
        domain_name: `load-test-${__VU}-${__ITER}.com`,
        upstreams: ['10.0.0.1:8080'],
    }), { headers: { Authorization: `Bearer ${token}` } });
}
```

基础设施假设：

- 测试集群：3 台 4 核 8GB 实例（1 台应用 + 1 台 PostgreSQL + 1 台 Caddy）；
- PostgreSQL 规格：`db.r6g.large` 同等配置，连接数上限 500；
- Caddy 以标准模式（非 admin 调试模式）运行，启用 mTLS；
- 应用与数据库、Caddy 之间同 VPC 部署，网络延迟 < 5ms。

分阶段目标：

- **P0（基线）**：单接口 100 RPS 下 P99 < 200ms，错误率 < 0.1%；
- **P1（目标）**：达到下表 RPS 目标，P99 与错误率满足阈值；
- **P2（极限）**：持续加压至 80% CPU，观察降级曲线，无级联故障。

| 场景 | 目标 RPS | P99 延迟 | 错误率 | 阶段 |
|---|:---:|:---:|:---:|:---:|
| 登录 | 500 | < 100ms | < 0.1% | P1 |
| 域名查询（缓存命中） | 5000 | < 20ms | < 0.01% | P1 |
| 域名创建（含 Caddy） | 200 | < 500ms | < 1% | P1 |
| 并发状态转换 | 1000 | < 50ms | < 0.5% | P1 |

---

## 7. 安全测试

### 7.1 OWASP Top 10 检查清单

| # | 类别 | 测试项 | 工具/方法 |
|---|---|---|---|
| S1 | 访问控制 | 横向越权（用户 A 访问项目 B 的域名） | API E2E |
| S2 | 访问控制 | 纵向越权（viewer 调用 admin 接口） | API E2E |
| S3 | 访问控制 | @id 篡改（修改请求中的 @id 值） | 手动 + 自动 |
| S4 | 注入 | SQL 注入（排序参数、搜索参数） | Go 单元/集成测试覆盖（验证 Ent 参数化查询、非法输入被拒绝）+ gosec 静态扫描 + OWASP ZAP 动态扫描 |
| S5 | 注入 | JSON 路径注入（@id 含 `../../`） | 手动 |
| S6 | 认证 | JWT 篡改（修改 payload） | 手动 |
| S7 | 认证 | 过期 Token 使用 | API E2E |
| S8 | 认证 | 重复 Token 刷新 | 并发测试 |
| S9 | 配置 | Caddy Admin API 未授权访问 | `nmap` + `curl` |
| S10 | 配置 | mTLS 证书伪造 | 手动 |
| S11 | 数据 | 敏感信息日志泄露 | 日志审查 |
| S12 | 数据 | API Token 明文存储 | DB 检查 |

### 7.2 @id 注入测试

```go
func TestIDInjection_Prevention(t *testing.T) {
    maliciousIDs := []string{
        "../../../config/apps",
        "tenant_1_domain_../../admin",
        "tenant_1_domain_<script>alert(1)</script>",
        "tenant_1_domain_'; DROP TABLE domains; --",
        "tenant 1 domain 1",  // 空格
        "TENANT_1_DOMAIN_1",  // 大写（如果区分大小写）
    }

    for _, id := range maliciousIDs {
        t.Run("malicious_id_"+id, func(t *testing.T) {
            // 验证 @id 校验正则 ^[a-zA-Z0-9_-]+$ 拒绝
            matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, id)
            assert.False(t, matched, "malicious @id should be rejected: %s", id)
        })
    }
}
```

---

## 8. CI/CD 测试流水线

```yaml
# .github/workflows/ci.yml（示意）
name: CI
on: [push, pull_request]

jobs:
  unit-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: actions/cache@v4
        with:
          path: |
            ~/.cache/go-build
            ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out

  integration-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
      redis:
        image: redis:7
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: actions/cache@v4
        with:
          path: |
            ~/.cache/go-build
            ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-
      - uses: docker/setup-buildx-action@v3
      - run: go test -tags=integration ./test/integration/...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: actions/cache@v4
        with:
          path: |
            ~/.cache/go-build
            ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-
      - run: golangci-lint run
      - run: go vet ./...

  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: cd web && npm ci && npm test && npm run build
```

### 测试门禁

| 门禁 | 标准 | 阻塞合并 |
|---|---|:---:|
| 单元测试通过率 | 100% | ✅ |
| 单元测试覆盖率 | ≥ 80% | ✅ |
| 集成测试通过率 | 100% | ✅ |
| Lint 错误 | 0 | ✅ |
| Lint 警告 | 0 警告，或新代码 0 警告（--new-from-rev） | ✅ |
| E2E 测试通过率 | 100% | ✅（发版前） |

---

## 9. 测试数据管理

### 9.1 测试数据工厂

```go
// test/factory/factory.go
package factory

func NewDomain(projectID string) *model.Domain {
    return &model.Domain{
        ID:        uuid.New().String(),
        ProjectID: projectID,
        DomainName: fmt.Sprintf("test-%s.com", uuid.New().String()[:8]),
        CaddyID:   fmt.Sprintf("tenant_%s_domain_%s", projectID, uuid.New().String()[:8]),
        Status:    "active",
    }
}

func NewUser() *model.User {
    return &model.User{
        ID:       uuid.New().String(),
        Email:    fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8]),
        Status:   "active",
    }
}
```

### 9.2 数据库清理

每个测试用例使用独立的 testcontainers 数据库，或在 repository 层通过 `WithTx` 封装事务并回滚：

```go
func TestWithTx(t *testing.T, fn func(ctx context.Context, repo *repository.Repository) error) {
    ctx := context.Background()

    pg, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("portunus_test_tx"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    if err != nil {
        t.Fatalf("start postgres container: %v", err)
    }
    defer func() {
        if err := pg.Terminate(ctx); err != nil {
            t.Logf("terminate postgres: %v", err)
        }
    }()

    connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatalf("get connection string: %v", err)
    }

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        t.Fatalf("open db: %v", err)
    }
    defer db.Close()

    repo := repository.New(db)
    if err := fn(ctx, repo); err != nil {
        t.Fatalf("test function failed: %v", err)
    }
}

// 或者使用 repository 层的 WithTx 封装：
func TestRepositoryWithTx(t *testing.T) {
    TestWithTx(t, func(ctx context.Context, repo *repository.Repository) error {
        return repo.WithTx(ctx, func(tx *sql.Tx) error {
            // 在独立事务中执行并自动回滚
            return nil
        })
    })
}
```

---

## 10. 测试文件组织

```
test/
├── integration/             # 集成测试
│   ├── setup_test.go        # 测试环境初始化
│   ├── domain_test.go
│   ├── project_test.go
│   ├── caddy_test.go
│   └── mcp_test.go
├── e2e/                      # E2E 测试
│   ├── api/                 # API E2E
│   │   ├── user_journey_test.go
│   │   └── caddy_failover_test.go
│   └── playwright/          # 前端 E2E
│       ├── domain.spec.ts
│       └── auth.spec.ts
├── factory/                  # 测试数据工厂
│   └── factory.go
└── mocks/                    # 生成的 Mock
    └── ...
```

---

## 附录 A：测试命令速查

```bash
# 全量测试
make test

# 仅单元测试
make test-unit

# 仅集成测试（需要 Docker）
make test-integration

# 仅 E2E 测试
make test-e2e

# 性能基准测试
make bench

# 覆盖率报告
make coverage

# 前端测试
cd web && npm test

# 生成 Mock
go generate ./internal/...

# 运行单个测试
go test -run TestDomainService_Create -v ./internal/service/
```
