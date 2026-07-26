// Package config 提供数据库连接管理。
package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/nrgao/portunus/ent/generated"

	// 导入 PostgreSQL 驱动
	_ "github.com/lib/pq"
)

// NewEntClient 创建并配置 Ent Client 连接 PostgreSQL。
// 配置连接池参数后立即执行一次 Ping 验证连通性。
func NewEntClient(ctx context.Context, cfg DatabaseConfig) (*generated.Client, error) {
	// 打开 PostgreSQL 连接
	drv, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 配置连接池
	db := drv.DB()
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// 验证连通性
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("数据库 Ping 失败: %w", err)
	}

	slog.InfoContext(ctx, "数据库连接成功",
		"max_open_conns", cfg.MaxOpenConns,
		"max_idle_conns", cfg.MaxIdleConns,
	)

	client := generated.NewClient(generated.Driver(drv))
	return client, nil
}