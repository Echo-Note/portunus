// Package config 提供结构化日志配置。
package config

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogger 初始化全局 slog Logger。
// 支持 JSON 和 Text 两种输出格式，通过 LOG_FORMAT 环境变量控制。
// 日志级别通过 LOG_LEVEL 环境变量控制：debug / info / warn / error。
func InitLogger() {
	level := logLevel(getEnv("LOG_LEVEL", "info"))
	format := getEnv("LOG_FORMAT", "json")

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler
	if strings.ToLower(format) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))

	slog.Info("日志系统初始化完成",
		"level", level.String(),
		"format", format,
	)
}

// logLevel 将字符串转换为 slog.Level。
func logLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}