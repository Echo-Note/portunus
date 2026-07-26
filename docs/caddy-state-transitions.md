# 系统状态流转设计文档

> **版本**: v1.0  
> **日期**: 2026-07-24  
> **关联文档**:  
> - `caddy-multi-tenant-architecture.md` v1.3 — 系统架构  
> - `caddy-system-architecture.md` v1.0 — 前后端架构  
> **状态**: 设计评审中

---

## 目录

- [1. 概述](#1-概述)
  - [1.1 文档目的](#11-文档目的)
  - [1.2 实体关系总览](#12-实体关系总览)
  - [1.3 状态流转设计原则](#13-状态流转设计原则)
- [2. 用户状态流转](#2-用户状态流转)
- [3. 项目状态流转](#3-项目状态流转)
- [4. 域名状态流转](#4-域名状态流转)
- [5. 反向代理配置状态流转](#5-反向代理配置状态流转)
- [6. 上游节点状态流转](#6-上游节点状态流转)
- [7. 项目成员状态流转](#7-项目成员状态流转)
- [8. 域名共享状态流转](#8-域名共享状态流转)
- [9. 配置快照状态流转](#9-配置快照状态流转)
- [10. Caddy 配置同步状态流转](#10-caddy-配置同步状态流转)
- [11. 跨实体联动状态流转](#11-跨实体联动状态流转)
- [12. 状态机实现规范](#12-状态机实现规范)
- [附录: 状态码速查表](#附录-状态码速查表)

---

## 1. 概述

### 1.1 文档目的

本文档定义系统中所有核心实体的**完整生命周期状态机**，包括：

- 每个实体的所有合法状态
- 状态之间的转换条件与触发事件
- 状态转换时的副作用（Caddy 配置变更、数据库更新、审计记录）
- 跨实体的联动状态流转（如项目冻结时域名的级联行为）
- 异常场景的状态恢复策略

### 1.2 实体关系总览

```
用户 (User)
  │
  │ N:N (通过 project_members)
  ▼
项目 (Project) ──────── 域名共享 (DomainShare) ───────→ 目标项目
  │                                                    │
  │ 1:N                                                │
  ▼                                                    ▼
域名 (Domain) ←─── 共享来源 ──── 域名 (Domain)
  │
  │ 1:1
  ▼
反向代理 (ProxyConfig)
  │
  │ 1:N
  ▼
上游 (Upstream)

项目 ─── 1:N ──→ 配置快照 (ConfigSnapshot)
项目 ─── 1:N ──→ 审计日志 (AuditLog)
项目 ─── 1:N ──→ 成员邀请 (Invitation)
```

### 1.3 状态流转设计原则

| 原则 | 说明 | 示例 |
|---|---|---|
| **状态可追溯** | 每次状态变更记录到审计日志，含变更前后值 | 项目 `active→suspended` 记录原因和操作者 |
| **Caddy 同步优先** | 状态变更成功的前提是 Caddy 配置已正确更新 | 域名标记为 `active` 前必须确认 Caddy 路由已加载 |
| **失败可回滚** | 状态转换失败时回滚到前一状态 | Caddy 下发失败 → 域名进入 error 状态 |
| **级联有边界** | 父实体状态变更影响子实体，但有明确边界 | 项目冻结 → 域名不可写但保持运行；项目删除 → 域名级联删除 |
| **最终一致** | 数据库状态与 Caddy 运行态可能短暂不一致，但通过同步机制最终对齐 | 启动时 checksum 比对 + 修复 |

---

## 2. 用户状态流转

### 2.1 状态定义

| 状态 | DB 值 | 说明 | API 行为 | Caddy 影响 |
|---|---|---|---|---|
| **待激活** | `pending` | 已注册但未验证邮箱 | 仅 `/auth/verify` 可用 | 无 |
| **活跃** | `active` | 正常使用 | 全部 API 可用 | 可管理项目资源 |
| **冻结** | `suspended` | 管理员冻结或安全事件 | 只读 API，写操作返回 403 | 项目资源保持运行 |
| **已删除** | `deleted` | 软删除，30 天保留期 | 全部 API 返回 401 | 名下项目进入 `deleting` |

### 2.2 状态机

```
                        POST /auth/register
                              │
                              ▼
                    ┌──────────────────┐
                    │     pending      │
                    │  (待激活)         │
                    └────────┬─────────┘
                             │
                             │ 事件: email_verified
                             │   → POST /auth/verify
                             │   → OAuth callback
                             │
                             ▼
               ┌─────────────────────────────┐
        ┌──────│          active              │──────┐
        │      │         (活跃)              │      │
        │      └─────────────────────────────┘      │
        │                  ▲                       │
        │                  │ 事件: appeal_approved  │
        │                  │   → 管理员审批通过     │
        │                  │                       │
        ▼                  │                       ▼
┌──────────────────┐       │          ┌──────────────────┐
│   suspended       │───────┘          │     deleted      │
│   (冻结)          │                  │    (已删除)       │
└────────┬─────────┘                  └───────────────────┘
         │                                     ▲
         │ 事件: retention_expired             │
         │   → 冻结满 30 天                    │ 事件: delete_requested
         │                                     │   → 用户主动注销
         ▼                                     │   → 管理员删除
┌──────────────────┐                           │
│     deleted       │───────────────────────────┘
│    (已删除)       │
└────────┬─────────┘
         │
         │ 事件: purge_completed
         │   → 30 天保留期满
         │   → 物理删除数据（匿名化）
         │
         ▼
    [数据归档]
    · email → anonymized_****
    · password_hash → NULL
    · 名下项目已级联删除
    · 审计日志保留（actor_id 保留，PII 清除）
```

### 2.3 转换条件矩阵

| 从 → 到 | 触发事件 | 前置条件 | 副作用 |
|---|---|---|---|
| `pending → active` | 邮箱验证码验证 / OAuth 回调 | 验证码正确 | 发送欢迎邮件 |
| `active → suspended` | 管理员操作 / 安全事件 / 违规 | 管理员有 `system_admin` 角色 | 撤销所有活跃 JWT（jti → blacklist） |
| `suspended → active` | 申诉通过 | 管理员审批 | 重新允许登录 |
| `active → deleted` | 用户注销 / 管理员删除 | 用户确认或管理员有权限 | 名下所有项目进入 `deleting` 状态；撤销 JWT |
| `suspended → deleted` | 冻结满 30 天未申诉 | 定时任务检查 | 同上 |
| `deleted → [归档]` | 保留期满 30 天 | 定时任务 | 物理删除 PII，保留审计日志 |

### 2.4 异常处理

| 异常场景 | 处理策略 |
|---|---|
| 用户有活跃项目时注销 | 项目状态先转为 `deleting`，异步清理 Caddy 配置后完成用户删除 |
| 冻结时有进行中的 Caddy 配置变更 | 变更被中止，配置回滚到变更前状态 |
| OAuth 关联账号被删除 | 用户状态转为 `suspended`，需重新绑定新账号 |

---

## 3. 项目状态流转

### 3.1 状态定义

| 状态 | DB 值 | 说明 | API 行为 | Caddy 影响 |
|---|---|---|---|---|
| **活跃** | `active` | 正常使用 | 全部读写 API | 正常路由和代理 |
| **冻结** | `suspended` | 暂停使用 | 只读 API，写操作 403 | Caddy 配置保持运行 |
| **删除中** | `deleting` | 异步清理中 | 全部 API 返回 409 | 正在删除 @id 节点 |
| **错误** | `error` | 删除过程中出现异常 | 只读 API | Caddy 配置部分清理 |
| **已删除** | `deleted` | 软删除 | 全部 API 返回 404 | Caddy @id 已清理 |

### 3.2 状态机

```
                    POST /api/v1/projects
                          │
                          ▼
               ┌─────────────────────┐
               │      active          │ ←───── 解冻
               │     (活跃)           │        (管理员审批 / 申诉)
               └──────┬────────┬─────┘
                      │        │
          冻结        │        │        删除
     (管理员/安全)    │        │     (owner/system_admin)
                      ▼        ▼
               ┌─────────────────────┐
               │     suspended        │
               │     (冻结)           │
               └──────┬──────┬───────┘
                      │      │
           解冻       │      │        删除
       (申诉通过)      │      │    (强制删除)
                      │      │
                      │      ▼
                      │  ┌──────────────────┐
                      │  │    deleting        │
                      │  │   (删除中)          │
                      │  │                    │
                      │  │ 异步执行:           │
                      │  │  1. 删除 Caddy @id  │
                      │  │  2. 清理 DB 映射    │
                      │  │  3. 归档审计日志    │
                      │  │  4. 撤销成员 JWT    │
                      │  │  5. 撤销共享        │
                      │  └────────┬───────────┘
                      │           │
                      │           │ 清理完成
                      │           ▼
                      │  ┌──────────────────┐
                      └──│    deleted        │
                         │   (已删除)         │
                         └────────┬───────────┘
                                  │
                                  │ 30 天保留期
                                  │
                                  ▼
                           [物理删除]
```

### 3.3 `deleting` 状态的异步清理流程

```
项目状态变为 deleting
        │
        ▼
┌───────────────────────────────────────────────┐
│  Step 1: 标记项目为 deleting                   │
│  → DB: projects.status = 'deleting'           │
│  → 撤销该项目所有成员的 JWT                      │
│  → 所有 API 返回 409 Conflict                  │
└───────────────────┬───────────────────────────┘
                    ▼
┌───────────────────────────────────────────────┐
│  Step 2: 查询所有 @id 节点                      │
│  → SELECT * FROM caddy_id_mappings            │
│  → WHERE project_id = ?                        │
└───────────────────┬───────────────────────────┘
                    ▼
┌───────────────────────────────────────────────┐
│  Step 3: 逐个删除 Caddy 配置节点               │
│  → FOR EACH caddy_id:                         │
│  │   → GET /id/<caddy_id> (获取 Etag)         │
│  │   → DELETE /id/<caddy_id>                  │
│  │   → If-Match: <etag>                      │
│  │   → 若 412 → 重试（最多 3 次）              │
│  │   → 若成功 → 删除 DB 映射记录               │
└───────────────────┬───────────────────────────┘
                    ▼
┌───────────────────────────────────────────────┐
│  Step 4: 撤销所有共享关系                       │
│  → UPDATE domain_shares SET status = 'revoked'│
│  → WHERE source_project_id = ?                │
│  → 通知被共享的项目（WebSocket 推送）            │
└───────────────────┬───────────────────────────┘
                    ▼
┌───────────────────────────────────────────────┐
│  Step 5: 保存配置快照                          │
│  → 获取 Caddy 当前配置 GET /config/            │
│  → 确认项目相关节点已全部移除                    │
│  → 存储快照到 config_snapshots                 │
└───────────────────┬───────────────────────────┘
                    ▼
┌───────────────────────────────────────────────┐
│  Step 6: 标记项目为 deleted                    │
│  → DB: projects.status = 'deleted'            │
│  → 归档审计日志                                 │
│  → 发送通知给 owner                             │
└───────────────────────────────────────────────┘
```

### 3.3.1 deleting 状态的异常恢复

如果异步清理流程中某一步失败（如 Caddy API 不可达、部分 @id 删除失败），项目状态转为 `error`：

```
项目 deleting → 清理失败 → 项目 error
                            │
                            │ 修复后重试
                            │ (定时任务每 5 分钟检查)
                            │
                            ▼
                        回到 deleting
                        重新执行清理
```

| 失败场景 | 恢复策略 |
|---|---|
| Caddy API 不可达 | 等待 Caddy 恢复后重试，最长等待 30 分钟 |
| 部分 @id 删除失败（412 冲突） | 重新获取 Etag 后逐个重试 |
| 数据库映射清理失败 | 记录错误，下次定时任务重试 |
| 全部重试耗尽（3 次） | 通知 system_admin 人工介入 |

### 3.4 转换条件矩阵

| 从 → 到 | 触发事件 | 前置条件 | 副作用 |
|---|---|---|---|
| `[创建] → active` | `POST /projects` | 用户配额未超 | 初始化 Caddy 配置上下文；创建 owner 成员记录 |
| `active → suspended` | 管理员操作 / 配额超限 / 安全事件 | 管理员有权限 | 撤销成员 JWT；Caddy 配置保持运行；写 API 返回 403 |
| `suspended → active` | 申诉通过 | 管理员审批 | 恢复 API 写权限 |
| `suspended → deleting` | 强制删除 | 管理员有权限 | 启动异步清理 |
| `active → deleting` | owner 删除 | owner 确认 | 启动异步清理 |
| `deleting → deleted` | 异步清理完成 | 所有 @id 已删除 | 归档审计日志 |
| `deleted → [物理删除]` | 保留期满 30 天 | 定时任务 | 物理删除数据 |

### 3.5 `suspended` 状态的详细行为

| 操作类型 | 允许? | 说明 |
|---|:---:|---|
| `GET` 查询项目/域名/代理 | ✅ | 只读操作正常 |
| `GET` 查询上游健康状态 | ✅ | 实时状态可查 |
| `POST/PATCH/DELETE` 域名 | ❌ | 返回 403 + "Project is suspended" |
| `POST` 邀请成员 | ❌ | 返回 403 |
| `POST` 共享域名 | ❌ | 返回 403 |
| 接收共享 | ❌ | 其他项目无法共享给已冻结项目 |
| MCP 工具调用（写操作） | ❌ | 返回 403 |
| MCP 工具调用（读操作） | ✅ | 正常返回 |

---

## 4. 域名状态流转

### 4.1 状态定义

| 状态 | DB 值 | 说明 | Caddy 路由 | Caddy 代理 |
|---|---|---|---|---|
| **创建中** | `creating` | 正在翻译配置并下发 Caddy | 未创建 | 未创建 |
| **活跃** | `active` | 正常运行 | 已加载 | 正常代理 |
| **更新中** | `updating` | 配置正在变更（Caddy PATCH 中） | 旧配置运行中 | 正常代理（旧配置） |
| **错误** | `error` | Caddy 配置加载失败 | 旧配置或无配置 | 取决于回滚结果 |
| **禁用** | `disabled` | 手动禁用，停止代理 | 路由保留但返回 503 | 返回 503 Service Unavailable |
| **删除中** | `deleting` | 正在从 Caddy 删除 | 正在删除 | 正在停止 |
| **已删除** | `deleted` | 软删除 | 已删除 | 已停止 |

### 4.2 状态机

```
    POST /domains
         │
         ▼
┌────────────────┐
│   creating      │
│  (创建中)        │
└───────┬────────┘
        │
        │ 1. 翻译 Caddy JSON
        │ 2. JSON Schema 校验
        │ 3. 获取 Etag
        │ 4. POST /config/.../routes (If-Match)
        │
        ├── 成功 ──→ ┌──────────────────────────────────────┐
        │             │            active                     │
        │             │           (活跃)                      │
        │             │                                      │
        │             │  Caddy: 路由已加载 + 代理正常          │
        │             │  SSL: 自动 HTTPS 已启用               │
        │             └──┬─────┬──────┬──────┬───────────────┘
        │                │     │      │      │
        │     更新配置    │     │      │      │   删除
        │  (PATCH /proxy) │     │      │   (DELETE)
        │                ▼     │      │      │
        │      ┌──────────────┐│      │      │
        │      │   updating   ││      │      │
        │      │  (更新中)     ││      │      │
        │      └──────┬───────┘│      │      │
        │             │        │      │      │
        │    ┌────────┴────┐   │      │      │
        │    │             │   │      │      │
        │  成功          失败│   │ 禁用 │      │
        │    │          (412)│   │      │      │
        │    ▼             │  │      │      │
        │  back to      ┌──▼──┐    │      │
        │   active      │error│    │      │
        │              │(错误)│    │      │
        │              └──┬──┘    │      │
        │           重试成功│      │      │
        │                │      │      │
        │                ▼      ▼      ▼
        │           ┌──────────────────────┐
        │           │      deleting         │
        │           │     (删除中)           │
        │           └────────┬─────────────┘
        │                    │
        │           1. DELETE /id/<route_id>
        │           2. DELETE /id/<proxy_id>
        │           3. 清理 DB 映射
        │           4. 保存快照
        │                    │
        │                    ▼
        │           ┌──────────────────────┐
        └──────────→│      deleted          │
                    │     (已删除)           │
                    └──────────────────────┘

         ┌──────────────────────────────┐
         │  disabled (禁用) 状态:        │
         │  · Caddy 路由保留（不删除）    │
         │  · 但 handle 改为返回 503     │
         │  · 可随时重新启用 → active    │
         └──────────────────────────────┘
```

### 4.3 转换条件矩阵

| 从 → 到 | 触发事件 | Caddy 操作 | 前置条件 |
|---|---|---|---|
| `[创建] → creating` | `POST /domains` | — | 项目 `active`，配额未超 |
| `creating → active` | Caddy 路由创建成功 | `POST /config/.../routes` | Etag 匹配 |
| `creating → error` | Caddy 路由创建失败 | 由控制面执行补偿式删除/回滚 | — |
| `active → updating` | `PATCH /domains/:did/proxy` | 获取 Etag | 项目 `active` |
| `updating → active` | Caddy 配置更新成功 | `PATCH /id/<proxy_id>` + `If-Match` | Etag 匹配 |
| `updating → error` | Caddy 配置更新失败（412 或其他） | 由控制面执行补偿式删除/回滚 | 重试 3 次后 |
| `error → active` | 重试成功 | 重新 `PATCH` | 手动或自动重试 |
| `active → disabled` | 手动禁用 | `PATCH /id/<route_id>/handle` → `static_response 503` | owner/admin |
| `disabled → active` | 手动启用 | `PATCH /id/<route_id>/handle` → 恢复原 handler | owner/admin |
| `active/disabled → deleting` | `DELETE /domains/:did` | — | owner/admin |
| `deleting → deleted` | Caddy 节点删除完成 | `DELETE /id/<route_id>` + `DELETE /id/<proxy_id>` | — |

### 4.4 `error` 状态的处理策略

```
域名进入 error 状态
        │
        ▼
   ┌──────────────────────────────┐
   │  自动重试（最多 3 次）         │
   │  间隔: 1s → 2s → 4s (指数退避) │
   └──────────────┬───────────────┘
                  │
           ┌──────┴──────┐
           │             │
         重试成功       重试失败
           │             │
           ▼             ▼
      back to       保持 error
       active        + 通知 owner
                      (邮件/WebSocket)
                      + 记录审计日志
                      + 标记需要人工介入
```

---

## 5. 反向代理配置状态流转

### 5.1 状态定义

代理配置的状态与域名状态**联动但不完全同步**——一个域名对应一个代理配置，代理配置可以独立更新（如修改负载均衡策略）而不改变域名状态。

| 状态 | 说明 | 与域名状态的关系 |
|---|---|---|
| **活跃** | 代理正常运行 | 域名为 `active` 时代理必然 `active` |
| **更新中** | 代理配置正在变更 | 域名可能仍为 `active`（代理独立更新） |
| **降级** | 部分上游不健康 | 域名 `active`，但代理健康度下降 |
| **不可用** | 所有上游不健康 | 域名 `active`，但 Caddy 返回 502 |

### 5.2 状态机

```
                    域名创建成功
                         │
                         ▼
               ┌─────────────────┐
               │     active       │ ←─────────────┐
               │    (活跃)        │               │
               └──┬───┬──────┬───┘               │
                  │   │      │                   │
      修改配置     │   │      │  上游健康检查      │
     (PATCH proxy) │   │      │  (Caddy 被动检测)  │
                  ▼   │      ▼                   │
          ┌──────────┐ │  ┌──────────┐           │
          │ updating │ │  │ degraded │           │
          │ (更新中)  │ │  │ (降级)    │           │
          └────┬─────┘ │  └──┬───┬───┘           │
               │       │     │   │               │
          ┌────┴────┐  │     │   │               │
          │         │  │     │   │               │
        成功      失败 │     │   │  上游恢复      │
          │         │  │     │   │  (健康检查通过) │
          │         ▼  │     │   │               │
          │    ┌──────┐ │     │   │               │
          │    │error │ │     │   │               │
          │    │(错误)│ │     │   │               │
          │    └──┬───┘ │     │   │               │
          │   重试成功│     │   │               │
          │        │  │     │   │               │
          │        ▼  │     │   │               │
          │        ├──┴─────┴───┴───────────────┘
          │        │  回到 active
          ▼        │
     ┌──────────┐  │     ┌──────────┐
     │  active  │  │     │unavailable│
     │          │  │     │ (不可用)   │
     └──────────┘  │     └─────┬────┘
                   │           │
                   │      所有上游恢复
                   │           │
                   └───────────┘
                   回到 active
```

### 5.3 健康检查驱动的状态转换

| Caddy 被动健康检查结果 | 代理状态转换 | 触发条件 |
|---|---|---|
| 0 个上游不健康 | `active` → 保持 `active` | 正常运行 |
| 1+ 上游不健康（但非全部） | `active` → `degraded` | 部分上游失败次数超阈值 |
| 全部上游不健康 | `active`/`degraded` → `unavailable` | 所有上游 `fail_duration` 内失败 |
| 不健康上游恢复 | `degraded`/`unavailable` → `active` | 健康检查连续通过 `passes` 次 |

> **注意**：代理的 `degraded`/`unavailable` 状态由 Caddy 的被动健康检查自动驱动，控制面通过 `GET /reverse_proxy/upstreams` 轮询感知并更新数据库状态。这不是由用户操作触发的状态转换。

---

## 6. 上游节点状态流转

### 6.1 状态定义

| 状态 | DB 值 | 说明 | Caddy 行为 |
|---|---|---|---|
| **活跃** | `active` | 正常参与负载均衡 | 正常代理流量 |
| **不健康** | `unhealthy` | 被动健康检查标记为不健康 | Caddy 自动移出负载均衡池 |
| **禁用** | `disabled` | 手动禁用 | 手动从 Caddy 配置中移除 |
| **已移除** | `removed` | 从配置中删除 | Caddy @id 已删除 |

### 6.2 状态机

```
    添加上游 (POST /upstreams)
         │
         ▼
┌─────────────────┐
│     active       │ ←─────────────┐
│    (活跃)        │               │
└──┬───┬──────┬───┘               │
   │   │      │                   │
   │   │      │ 手动禁用           │
   │   │      │ (DELETE /upstreams/:uid)│
   │   │      ▼                   │
   │   │  ┌──────────┐            │
   │   │  │ disabled  │            │
   │   │  │ (禁用)    │            │
   │   │  └────┬─────┘            │
   │   │       │ 重新启用          │
   │   │       └──────────────────┘
   │   │        回到 active
   │   │
   │   │ 健康检查失败
   │   │ (Caddy 被动检测)
   │   ▼
   │ ┌──────────────┐
   │ │  unhealthy    │
   │ │  (不健康)     │
   │ └──┬───────────┘
   │    │
   │    │ 健康检查恢复
   │    │ (连续 passes 次通过)
   │    └──────────────┐
   │                   │
   │                   ▼
   └──────────→ back to active
```

### 6.3 上游状态与代理状态的联动

```
上游状态变化                          代理状态联动
────────────                          ────────────
1 个上游 unhealthy (共 3 个)    →     proxy: active → degraded
2 个上游 unhealthy (共 3 个)    →     proxy: degraded (保持)
3 个上游 unhealthy (共 3 个)    →     proxy: degraded → unavailable
1 个上游恢复 healthy            →     proxy: unavailable → degraded
全部恢复 healthy               →     proxy: degraded → active
```

---

## 7. 项目成员状态流转

### 7.1 状态定义

`project_members` 表只记录已加入项目的成员，状态机如下：

| 状态 | DB 值 | 说明 |
|---|---|---|
| **活跃** | `active` | 已接受邀请，正常参与项目 |
| **已移除** | `removed` | 被管理员移除 |
| **已退出** | `left` | 成员主动退出 |

> **成员邀请** 的生命周期由 `invitations` 表单独维护，状态包括 `pending`、`accepted`、`expired`、`rejected`、`revoked`。`invitations.pending` 被接受后，才会在 `project_members` 表中创建 `active` 记录。

### 7.2 状态机

> 下图上半部分（`pending` → `active`/`[邀请失效]`）描述 `invitations` 表的状态；`active` 之后的 `project_members` 状态属于 `project_members` 表。

```
    邀请成员 (POST /members)
    生成邀请链接 (JWT, 24h 过期)
         │
         ▼
┌──────────────────────┐
│      pending          │
│     (待邀请)          │
│  · 邀请邮件已发送      │
│  · 邀请码 24h 过期     │
└───────┬──────┬───────┘
        │      │
   接受邀请  拒绝/过期
        │      │
        ▼      ▼
┌──────────────┐  ┌──────────────┐
│    active     │  │  [邀请失效]   │
│   (活跃)      │  │  删除邀请记录  │
│               │  └──────────────┘
│  · 正常使用    │
│  · 有角色权限  │
└──┬──────┬────┘
   │      │
 退出   移除
(主动)  (管理员)
   │      │
   ▼      ▼
┌──────────────┐  ┌──────────────┐
│    left       │  │   removed     │
│   (已退出)     │  │   (已移除)    │
└──────────────┘  └──────────────┘
  · 撤销 JWT       · 撤销 JWT
  · 可重新加入     · 需重新邀请
```

### 7.3 成员角色变更（非状态变更）

角色变更不改变 `active` 状态，但记录审计日志：

```
成员角色变更流程:
  1. 校验操作者权限（owner/admin）
  2. 校验目标角色合法性（admin 不能设为 owner）
  3. 更新 project_members.role
  4. 撤销旧 JWT（强制重新登录获取新角色）
  5. 记录审计日志（before: editor, after: admin）
  6. 通知被变更成员
```

---

## 8. 域名共享状态流转

### 8.1 状态定义

| 状态 | DB 值 | 说明 | 被共享方可操作 |
|---|---|---|---|
| **待接受** | `pending` | 共享已发起，等待目标项目确认 | 无（未接受） |
| **活跃** | `active` | 共享生效 | 按 permission 级别 |
| **已撤销** | `revoked` | 源项目主动撤销 | 无（已失效） |
| **已过期** | `expired` | 共享到期自动失效 | 无（已失效） |
| **已拒绝** | `rejected` | 目标项目拒绝共享 | 无（未生效） |

### 8.2 状态机

> **API 流程说明**：共享通过 `POST /api/v1/projects/:id/domains/:did/share` 创建，创建后默认状态为 `pending`（待接受）。目标项目需通过 `POST /api/v1/shares/:sid/accept` 确认后转为 `active`。如目标项目启用了"自动接受共享"设置，则创建后直接为 `active`。

```
   源项目发起共享
   POST /domains/:did/share
   (指定 target_project_id + permission)
         │
         ▼
┌──────────────────────┐
│      pending          │
│     (待接受)          │
│  · 通知目标项目        │
│  · @id 归属不变        │
└───────┬───┬───┬──────┘
        │   │   │
   接受  拒绝  撤销
        │   │   │ (源项目主动)
        ▼   ▼   ▼
┌──────────┐ ┌──────┐ ┌──────────┐
│  active   │ │reject│ │ revoked   │
│ (活跃)    │ │(拒绝)│ │ (已撤销)  │
└──┬───┬───┘ └──────┘ └──────────┘
   │   │
 过期  撤销
(到期) (源项目)
   │   │
   ▼   ▼
┌──────────┐
│ expired   │
│(已过期)   │
└──────────┘
```

### 8.3 共享状态与域名状态的联动

| 域名状态变化 | 活跃共享的影响 |
|---|---|
| 域名 `active → disabled` | 被共享方仍可查询配置（只读），但代理实际返回 503 |
| 域名 `active → deleting` | 所有共享自动转为 `revoked`，通知被共享方 |
| 域名 `active → error` | 被共享方查询时看到 `error` 状态 |
| 源项目 `active → suspended` | 共享保持 `active`（只读），但被共享方写操作被拒 |
| 源项目 `suspended → deleting` | 所有共享自动转为 `revoked` |

---

## 9. 配置快照状态流转

### 9.1 状态定义

配置快照是**不可变**的历史记录，状态简单：

| 状态 | 说明 |
|---|---|
| **已创建** | 快照已存储，可用于回滚 |
| **已归档** | 超过保留期，移至冷存储（S3/MinIO） |
| **已删除** | 超过归档期，物理删除 |

### 9.2 生命周期

```
配置变更成功
     │
     ▼
┌──────────────┐
│   created     │
│  (已创建)     │
│  · 版本号递增  │
│  · SHA-256   │
│  · 可回滚     │
└──────┬───────┘
       │ 超过保留期
       │ (max_config_snapshots 条)
       ▼
┌──────────────┐
│  archived     │
│  (已归档)     │
│  · 移至 S3    │
│  · 可恢复     │
│  · 不在快速查询范围
└──────┬───────┘
       │ 超过归档期 (90 天)
       ▼
┌──────────────┐
│   deleted     │
│  (已删除)     │
│  · 物理删除   │
└──────────────┘
```

### 9.3 回滚操作的状态交互

```
用户请求回滚到版本 N
         │
         ▼
   ┌──────────────────────────┐
   │  1. 加载快照版本 N 的 JSON │
   │  2. POST /adapt 预检       │
   │     (如果格式校验通过)      │
   │  3. POST /load 整体替换     │
   │     (Caddy 失败自动回滚)   │
   └────────────┬─────────────┘
                │
         ┌──────┴──────┐
         │             │
       成功          失败
         │             │
         ▼             ▼
   创建新快照      保持当前配置
   (版本 N+1)    记录回滚失败日志
   记录审计日志    通知管理员
```

---

## 10. Caddy 配置同步状态流转

### 10.1 状态定义

控制面与 Caddy 之间的配置同步状态：

| 状态 | 说明 | 数据库 vs Caddy |
|---|---|---|
| **同步中** | 控制面与 Caddy 配置一致 | 一致 |
| **同步中** | 正在推送配置到 Caddy | 可能短暂不一致 |
| **漂移** | 检测到不一致 | 不一致 |
| **恢复中** | 正在从数据库恢复到 Caddy | 恢复中 |
| **不可达** | Caddy Admin API 不可达 | 未知 |

### 10.2 状态机

```
                    系统启动
                       │
                       ▼
              ┌────────────────────┐
              │   checking          │
              │  (检查中)           │
              │  · GET /config/     │
              │  · 比对 checksum    │
              └────┬──────┬────┬───┘
                   │      │    │
              一致  │   不一致  Caddy 不可达
                   │      │    │
                   ▼      ▼    ▼
           ┌──────────┐ ┌────────┐ ┌──────────┐
           │  synced   │ │ drift  │ │unreachable│
           │ (已同步)   │ │(漂移)  │ │(不可达)   │
           └─────┬────┘ └───┬────┘ └────┬─────┘
                 │          │           │
            配置变更       恢复       Caddy 恢复
                 │          │           │
                 ▼          │           │
           ┌──────────┐     │           │
           │ syncing   │     │      ┌───┘
           │ (同步中)  │     │      │
           └────┬─────┘     │      │
                │           │      │
            下发完成        │      │
                │           │      │
                ▼           │      │
           ┌──────────┐     │      │
           │  synced  │←────┘      │
           └──────────┘            │
                              ┌────┘
                              │
                              ▼
                         ┌────────┐
                         │ drift  │──→ 恢复 ──→ synced
                         └────────┘
```

### 10.3 同步触发时机

| 触发事件 | 同步行为 |
|---|---|
| **系统启动** | checksum 比对 → synced 或 drift → restoring |
| **配置变更成功后** | 主动存储快照 + 更新 checksum → synced |
| **定时健康检查**（每 60 秒） | GET /config/ → 比对 → synced 或 drift |
| **Caddy 重启检测** | GET /config/ 失败 → unreachable → 等待恢复 → restoring |
| **手动触发** | POST /api/v1/admin/sync → checking → synced/restoring |

---

## 11. 跨实体联动状态流转

### 11.1 项目冻结的级联效应

```
项目: active → suspended
         │
         ├──→ 域名: 保持 active（Caddy 继续运行）
         │         但 API 写操作被拒绝（返回 403）
         │
         ├──→ 代理: 保持 active / degraded / unavailable（健康检查继续）
         │
         ├──→ 上游: 保持各自状态（Caddy 继续代理）
         │
         ├──→ 共享: 保持 active（被共享方仍可只读访问）
         │         但被共享方写操作被拒绝
         │
         ├──→ 成员: JWT 撤销 → 需重新登录
         │         登录后发现项目 suspended → 只能查看
         │
         └──→ 审计: 记录 "project.suspended" 事件
```

### 11.2 项目删除的级联效应

```
项目: active → deleting
         │
         ├──→ 成员: 撤销所有 JWT → 成员被强制登出
         │
         ├──→ 域名: 逐个执行 deleting → deleted
         │         → Caddy @id 节点逐个删除
         │
         ├──→ 代理: 随域名删除而删除
         │
         ├──→ 上游: 随代理删除而删除
         │
         ├──→ 共享: 所有共享转为 revoked
         │         → 通知被共享项目
         │
         ├──→ 快照: 保存最终配置快照
         │         → 旧快照保留 30 天后归档
         │
         ├──→ 审计: 记录 "project.deleted" + 级联清理日志
         │
         └──→ 项目: deleting → deleted
```

### 11.3 域名删除的级联效应

```
域名: active → deleting
         │
         ├──→ Caddy: DELETE /id/<route_id> + DELETE /id/<proxy_id>
         │
         ├──→ 代理配置: 标记为 deleted
         │
         ├──→ 上游: 全部标记为 removed
         │
         ├──→ 共享: 该域名的所有共享转为 revoked
         │         → 通知被共享项目
         │
         ├──→ DB 映射: 删除 caddy_id_mappings 记录
         │
         ├──→ 快照: 保存配置快照
         │
         ├──→ 审计: 记录 "domain.deleted"
         │
         └──→ 域名: deleting → deleted
```

### 11.4 用户删除的级联效应

```
用户: active → deleted
         │
         ├──→ 名下项目:
         │    · owner 身份的项目 → 进入 deleting
         │    · 非_owner 身份的项目 → 仅移除成员关系
         │
         ├──→ JWT: 所有 JWT 撤销
         │
         ├──→ API Token: 所有 Token 撤销
         │
         ├──→ 审计: 记录 "user.deleted"
         │         → actor_name 保留但标记为 "已删除用户"
         │
         └──→ PII: 30 天后匿名化
```

### 11.5 共享被撤销对被共享方的影响

```
源项目撤销共享
      │
      ▼
共享状态: active → revoked
      │
      ├──→ 被共享项目:
      │    · WebSocket 推送通知
      │    · 该域名从被共享项目的域名列表中移除
      │    · 如有正在进行的配置修改 → 中止并回滚
      │
      ├──→ 被共享项目成员:
      │    · 前端实时收到通知弹窗
      │    · 该域名的配置查看页面显示 "共享已撤销"
      │
      └──→ 审计: 源项目和被共享项目都记录
```

---

## 12. 状态机实现规范

### 12.1 状态转换守卫模式

所有状态转换必须通过守卫函数校验，防止非法转换：

```go
// internal/service/state_machine.go

// 项目状态转换守卫
var projectTransitions = map[string][]string{
    "active":    {"suspended", "deleting"},
    "suspended": {"active", "deleting"},
    "deleting":  {"deleted", "error"},   // 删除失败可进入 error 状态
    "error":     {"deleting"}, // 错误后只可重试删除
    "deleted":   {},  // 终态，不可转换
}

func validateProjectTransition(from, to string) error {
    allowed, ok := projectTransitions[from]
    if !ok {
        return fmt.Errorf("unknown source state: %s", from)
    }
    for _, s := range allowed {
        if s == to {
            return nil
        }
    }
    return fmt.Errorf("invalid transition: %s → %s", from, to)
}

// 域名状态转换守卫
var domainTransitions = map[string][]string{
    "creating":  {"active", "error"},
    "active":    {"updating", "disabled", "deleting", "error"},
    "updating":  {"active", "error"},
    "error":     {"active", "deleting"},  // 错误后可重试或删除，不可回到 creating
    "disabled":  {"active", "deleting"},
    "deleting":  {"deleted", "error"},                // 删除失败进入 error
    "deleted":   {},
}

// 上游状态转换守卫
var upstreamTransitions = map[string][]string{
    "active":    {"unhealthy", "disabled", "removed"},
    "unhealthy": {"active", "disabled", "removed"},
    "disabled":  {"active", "removed"},
    "removed":   {},
}

// 共享状态转换守卫
var shareTransitions = map[string][]string{
    "pending":  {"active", "rejected", "revoked", "expired"},
    "active":   {"revoked", "expired"},
    "revoked":  {},
    "expired":  {},
    "rejected": {},
}
```

### 12.2 状态转换的统一执行模板

```go
// internal/service/state_transition.go

type StateTransition struct {
    EntityType string      // "project" / "domain" / "upstream" / "share"
    EntityID   string
    FromState  string
    ToState    string
    Trigger    string      // "user_action" / "system" / "health_check" / "timer"
    ActorID    string      // 操作者（system 表示系统自动）
    Reason     string      // 转换原因
    SideEffects []SideEffect  // 副作用列表
}

type SideEffect struct {
    Type     string  // "caddy_api" / "db_update" / "redis" / "websocket" / "audit"
    Action   string  // 具体操作
    Priority int     // 执行优先级
}

func ExecuteTransition(ctx context.Context, t *StateTransition) error {
    // 1. 校验转换合法性
    if err := validateTransition(t.EntityType, t.FromState, t.ToState); err != nil {
        return err
    }

    // 2. 先执行外部 IO 副作用（Caddy API），确保 Caddy 配置变更成功
    //    失败时直接返回错误，不修改数据库状态
    caddyEffects := filterSideEffects(t.SideEffects, "caddy_api")
    for _, effect := range caddyEffects {
        if err := executeExternalSideEffect(ctx, &effect); err != nil {
            slog.ErrorContext(ctx, "caddy side effect failed, aborting transition",
                "entity_type", t.EntityType,
                "entity_id", t.EntityID,
                "action", effect.Action,
                "err", err,
            )
            return fmt.Errorf("caddy side effect failed: %w", err)
        }
    }

    // 3. Caddy 成功后开启数据库事务，使用 Repeatable Read 隔离级别
    tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 4. 更新实体状态（乐观锁：WHERE status = $from）
    tableName, ok := entityTableMap[t.EntityType]
    if !ok {
        return fmt.Errorf("unknown entity type: %s", t.EntityType)
    }
    query := fmt.Sprintf(
        "UPDATE %s SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3",
        tableName, // tableName 来自白名单 map，非用户输入
    )
    result, err := tx.Exec(query, t.ToState, t.EntityID, t.FromState)
    if err != nil {
        return err
    }
    if rows, _ := result.RowsAffected(); rows == 0 {
        return ErrConcurrentModification  // 状态已被他人修改
    }

    // 5. 在事务内执行数据库类副作用（如级联更新、映射表清理）
    dbSideEffects := filterSideEffects(t.SideEffects, "db_update")
    for _, effect := range dbSideEffects {
        if err := executeDBSideEffect(ctx, tx, &effect); err != nil {
            return fmt.Errorf("db side effect failed: %w", err)
        }
    }

    // 6. 记录审计日志（事务内，确保与状态变更原子性）
    auditLog(ctx, tx, t)

    // 7. 提交事务
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transition failed: %w", err)
    }

    // 8. 事务提交后执行其他外部 IO 副作用（Redis、WebSocket 推送）
    //    失败时写入持久化工单队列，由后台 worker 串行处理
    externalEffects := filterSideEffects(t.SideEffects, "redis", "websocket")
    for _, effect := range externalEffects {
        if err := executeExternalSideEffect(ctx, &effect); err != nil {
            slog.ErrorContext(ctx, "external side effect failed, enqueue work order",
                "entity_type", t.EntityType,
                "entity_id", t.EntityID,
                "effect_type", effect.Type,
                "action", effect.Action,
                "err", err,
            )
            if err := enqueueWorkOrder(ctx, t, &effect); err != nil {
                slog.ErrorContext(ctx, "failed to enqueue work order", "err", err)
            }
            return fmt.Errorf("external side effect failed: %w", err)
        }
    }

    return nil
}

// 实体类型到表名的白名单映射（防止 SQL 注入）
var entityTableMap = map[string]string{
    "project":  "projects",
    "domain":   "domains",
    "upstream": "upstreams",
    "share":    "domain_shares",
    "member":   "project_members",
}
```

> **事务隔离级别**：状态转换事务使用 `sql.LevelRepeatableRead` 隔离级别，确保事务执行期间其他会话无法修改同一实体的状态，从而保证乐观锁判断的准确性。

> **分布式锁与后台 Worker**：补偿/重试任务由后台 worker 从持久化工单队列中消费。同一实体的工单通过分布式锁（如基于 Redis/数据库的 `SELECT ... FOR UPDATE` 或分布式锁服务）串行执行，避免并发冲突导致状态错乱。

#### 并发冲突处理

同一实体的并发 API 调用可能同时满足守卫条件并尝试更新状态。执行模板使用数据库乐观锁（`UPDATE ... WHERE status = $from`）保证仅一个调用能成功；若 `RowsAffected() == 0`，说明状态已被其他调用修改，接口应返回 `409 Conflict`，客户端需重新查询当前状态后重试。

### 12.3 定时任务驱动的状态转换

| 定时任务 | 频率 | 检查内容 | 状态转换 |
|---|---|---|---|
| 用户保留期清理 | 每日 | `suspended` 超 30 天的用户 | `suspended → deleted` |
| 项目物理删除 | 每日 | `deleted` 超 30 天的项目 | 物理删除 |
| invitations 表邀请过期清理 | 每小时 | `invitations.pending` 超 24h | `pending → expired`（删除记录） |
| 共享过期检查 | 每小时 | 有过期时间的 `active` 共享 | `active → expired` |
| 配置漂移检测 | 每 60 秒 | Caddy checksum vs 数据库 | `synced → drift` |
| 上游健康检查 | 每 30 秒 | `GET /reverse_proxy/upstreams` | 更新上游和代理状态 |
| 快照归档 | 每日 | 超过保留数的快照 | `created → archived` |
| 快照物理删除 | 每日 | 归档超 90 天的快照 | `archived → deleted` |

> **上游健康检查 Caddy 不可达重试策略**：当 `GET /reverse_proxy/upstreams` 失败时，不再以固定 30 秒间隔继续重试，而是采用指数退避（初始 2s，最大 60s，退避倍率 2x）并引入熔断机制：连续失败 5 次后进入 5 分钟熔断期，期间直接标记上游状态为 `unavailable`，熔断结束后重新探测。

### 12.4 定时任务实现方式

定时任务使用 Go 标准库 `time.Ticker` 实现，在控制面进程内运行：

```go
// internal/scheduler/scheduler.go
type Scheduler struct {
    tasks []ScheduledTask
}

type ScheduledTask struct {
    Name     string
    Interval time.Duration
    Handler  func(ctx context.Context) error
}

func (s *Scheduler) Start(ctx context.Context) {
    for _, task := range s.tasks {
        go s.runTask(ctx, task)
    }
}

func (s *Scheduler) runTask(ctx context.Context, task ScheduledTask) {
    ticker := time.NewTicker(task.Interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := task.Handler(ctx); err != nil {
                slog.ErrorContext(ctx, "scheduled task failed",
                    "task", task.Name, "err", err)
            }
        }
    }
}
```

> **注**：生产环境可替换为 `robfig/cron` 库以支持 cron 表达式，或使用 Kubernetes CronJob 实现分布式定时任务。

---

## 附录: 状态码速查表

### 实体状态码

| 实体 | 状态 | DB 值 | 终态? |
|---|---|---|:---:|
| **用户** | 待激活 | `pending` | |
| | 活跃 | `active` | |
| | 冻结 | `suspended` | |
| | 已删除 | `deleted` | ✅ |
| **项目** | 活跃 | `active` | |
| | 冻结 | `suspended` | |
| | 删除中 | `deleting` | |
| | 已删除 | `deleted` | ✅ |
| **域名** | 创建中 | `creating` | |
| | 活跃 | `active` | |
| | 更新中 | `updating` | |
| | 错误 | `error` | |
| | 禁用 | `disabled` | |
| | 删除中 | `deleting` | |
| | 已删除 | `deleted` | ✅ |
| **代理** | 活跃 | `active` | |
| | 更新中 | `updating` | |
| | 降级 | `degraded` | |
| | 不可用 | `unavailable` | |
| **上游** | 活跃 | `active` | |
| | 不健康 | `unhealthy` | |
| | 禁用 | `disabled` | |
| | 已移除 | `removed` | ✅ |
| **成员** | 待邀请 | `pending` | |
| | 活跃 | `active` | |
| | 已移除 | `removed` | ✅ |
| | 已退出 | `left` | ✅ |
| **共享** | 待接受 | `pending` | |
| | 活跃 | `active` | |
| | 已撤销 | `revoked` | ✅ |
| | 已过期 | `expired` | ✅ |
| | 已拒绝 | `rejected` | ✅ |
| **快照** | 已创建 | `created` | |
| | 已归档 | `archived` | |
| | 已删除 | `deleted` | ✅ |
| **配置同步** | 已同步 | `synced` | |
| | 同步中 | `syncing` | |
| | 漂移 | `drift` | |
| | 恢复中 | `restoring` | |
| | 不可达 | `unreachable` | |
