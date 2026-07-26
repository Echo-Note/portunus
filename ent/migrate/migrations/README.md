# Ent 迁移目录

此目录由 Atlas 自动管理。运行 `make migrate-diff NAME=init` 生成初始迁移。

## 常用命令

- `make generate` — 从 ent/schema/*.go 生成 Go 代码
- `make migrate-diff NAME=xxx` — 从 schema 变更生成增量迁移
- `make migrate-apply` — 应用迁移到数据库
- `make migrate-lint` — CI 检查迁移

## 首次使用

1. 确保 PostgreSQL 已运行：`make up`
2. 生成 Ent 代码：`make generate`
3. 生成初始迁移：`make migrate-diff NAME=init`
4. 应用迁移：`make migrate-apply`
