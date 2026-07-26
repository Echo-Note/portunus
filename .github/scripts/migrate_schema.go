// migrate_schema 使用 Ent 自动迁移创建数据库 Schema。
// 此脚本由 CI 流水线在单元测试和集成测试之前运行。
//
// 用法：go run -mod=mod .github/scripts/migrate_schema.go
package main

import (
	"context"
	"log"
	"os"

	"github.com/Echo-Note/portunus/internal/config"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable"
	}

	cfg := config.DatabaseConfig{
		URL:          dsn,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}

	client, err := config.NewEntClient(ctx, cfg)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer client.Close()

	// 自动迁移（创建所有表）
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("Schema 迁移失败: %v", err)
	}

	log.Println("数据库 Schema 迁移完成")
}