# Portunus Makefile
# 常用开发命令集合

SHELL := /bin/bash
GOPATH := $(shell go env GOPATH)
DATABASE_URL ?= postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable

# ── 默认目标 ──
.DEFAULT_GOAL := help

# ── 开发 ──
.PHONY: dev dev-web
dev: ## 启动后端（热重载，需先安装 air：go install github.com/air-verse/air@latest）
	air cmd/portunus/main.go || go run cmd/portunus/main.go

dev-web: ## 启动前端开发服务器
	cd web && npm run dev

# ── 构建 ──
.PHONY: build build-web docker-build
build: ## 编译后端二进制
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/portunus cmd/portunus/main.go

build-web: ## 构建前端生产包
	cd web && npm run build

docker-build: ## 构建 Docker 镜像
	docker build -t portunus:latest .

# ── 代码生成 ──
.PHONY: generate mock
generate: ## Ent 代码生成（从 schema 生成 Go 代码）
	go run ent/entc.go

mock: ## 生成 Mock 接口
	go generate ./internal/...

# ── 数据库迁移 ──
.PHONY: migrate-diff migrate-apply migrate-lint migrate-status
migrate-diff: ## 从 schema 变更生成增量迁移（用法: make migrate-diff NAME=add_xxx）
	@test -n "$(NAME)" || (echo "用法: make migrate-diff NAME=add_xxx" && exit 1)
	atlas migrate diff $(NAME) \
		--to "ent://ent/schema" \
		--dev-url "docker://postgres/15/dev?search_path=public" \
		--dir "file://ent/migrate/migrations"

migrate-apply: ## 应用迁移到数据库
	atlas migrate apply \
		--dir "file://ent/migrate/migrations" \
		--url "$(DATABASE_URL)"

migrate-lint: ## CI 检查迁移文件的破坏性变更
	atlas migrate lint \
		--dir "file://ent/migrate/migrations" \
		--dev-url "docker://postgres/15/dev?search_path=public" \
		--latest 1

migrate-status: ## 查看迁移状态
	atlas migrate status \
		--dir "file://ent/migrate/migrations" \
		--url "$(DATABASE_URL)"

# ── 一键全量生成 ──
.PHONY: gen
gen: ## 全量生成：Ent 代码 + 迁移文件
	@echo "1/2 Generating Ent code..."
	go run ent/entc.go
	@echo "2/2 To generate migration diff, run: make migrate-diff NAME=your_change"

# ── 测试 ──
.PHONY: test test-unit test-integration test-e2e bench coverage lint
test: ## 运行所有测试
	go test -race -cover ./internal/...

test-unit: ## 仅单元测试
	go test -race -cover ./internal/...

test-integration: ## 集成测试（需要 Docker 依赖）
	go test -race -tags=integration ./test/integration/...

test-e2e: ## E2E 测试
	go test -race ./test/e2e/api/...
	cd web && npx playwright test

bench: ## 性能基准测试
	go test -bench=. -benchmem ./internal/...

coverage: ## 生成覆盖率报告
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html

lint: ## 代码检查
	golangci-lint run
	go vet ./...
	cd web && npm run lint

# ── 依赖 ──
.PHONY: deps deps-web
deps: ## 安装 Go 依赖
	go mod download

deps-web: ## 安装前端依赖
	cd web && npm install

# ── Docker ──
.PHONY: up down down-clean
up: ## 启动所有依赖服务
	docker compose up -d postgres redis caddy minio

down: ## 停止所有服务
	docker compose down

down-clean: ## 停止并清除数据卷
	docker compose down -v

# ── 证书 ──
.PHONY: certs
certs: ## 生成开发用 mTLS 证书（脚本待创建，暂请参考 README.md 第 7 节手动执行 openssl 命令）
	@echo "scripts/gen-certs.sh 尚未实现，请参考 README.md 第 7 节手动生成证书"
	@echo "或运行：mkdir -p certs && openssl req -x509 -newkey rsa:4096 -keyout certs/ca.key -out certs/ca.crt -days 365 -nodes -subj '/CN=Portunus Dev CA'"
	@exit 1

# ── 清理 ──
.PHONY: clean
clean: ## 清理构建产物
	rm -rf bin/ coverage.out coverage.html

# ── 帮助 ──
.PHONY: help
help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
