// Package config 提供应用配置加载与管理。
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

// Load 从环境变量加载应用配置。
// 优先使用 os.Getenv 读取，支持 .env 文件（通过 caarlos0/env 自动解析）。
func Load() (*Config, error) {
	cfg := &Config{}

	// 服务配置
	cfg.Server.Port = getEnv("PORT", "8080")
	cfg.Server.GinMode = getEnv("GIN_MODE", "debug")

	// 数据库配置
	cfg.Database.URL = getEnv("DATABASE_URL", "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable")
	cfg.Database.MaxOpenConns = getEnvInt("DATABASE_MAX_OPEN_CONNS", 25)
	cfg.Database.MaxIdleConns = getEnvInt("DATABASE_MAX_IDLE_CONNS", 5)
	cfg.Database.ConnMaxLifetime = getEnvDuration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute)

	// Redis 配置
	cfg.Redis.URL = getEnv("REDIS_URL", "redis://localhost:6379/0")
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")

	// JWT 配置
	cfg.JWT.PrivateKey = os.Getenv("JWT_PRIVATE_KEY")
	cfg.JWT.PublicKey = os.Getenv("JWT_PUBLIC_KEY")
	cfg.JWT.AccessTokenTTL = getEnvDuration("JWT_ACCESS_TOKEN_TTL", 15*time.Minute)
	cfg.JWT.RefreshTokenTTL = getEnvDuration("JWT_REFRESH_TOKEN_TTL", 168*time.Hour)

	// Caddy 配置
	cfg.Caddy.AdminURL = getEnv("CADDY_ADMIN_URL", "https://caddy:2021")
	cfg.Caddy.MTLSCert = getEnv("CADDY_MTLS_CERT", "./certs/client.crt")
	cfg.Caddy.MTLSKey = getEnv("CADDY_MTLS_KEY", "./certs/client.key")
	cfg.Caddy.MTLSCA = getEnv("CADDY_MTLS_CA", "./certs/ca.crt")
	cfg.Caddy.RequestTimeout = getEnvDuration("CADDY_REQUEST_TIMEOUT", 10*time.Second)
	cfg.Caddy.RetryMax = getEnvInt("CADDY_RETRY_MAX", 3)

	// MinIO 配置
	cfg.MinIO.Endpoint = getEnv("MINIO_ENDPOINT", "localhost:9000")
	cfg.MinIO.AccessKey = getEnv("MINIO_ACCESS_KEY", "minioadmin")
	cfg.MinIO.SecretKey = getEnv("MINIO_SECRET_KEY", "minioadmin")
	cfg.MinIO.Bucket = getEnv("MINIO_BUCKET", "portunus")
	cfg.MinIO.UseSSL = getEnvBool("MINIO_USE_SSL", false)

	// SMTP 配置
	cfg.SMTP.Host = os.Getenv("SMTP_HOST")
	cfg.SMTP.Port = getEnvInt("SMTP_PORT", 587)
	cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")
	cfg.SMTP.From = getEnv("SMTP_FROM", "noreply@portunus.local")
	cfg.SMTP.FromName = getEnv("SMTP_FROM_NAME", "Portunus")

	// OAuth 配置
	cfg.OAuth.GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
	cfg.OAuth.GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	cfg.OAuth.GitHubRedirectURL = getEnv("GITHUB_REDIRECT_URL", "http://localhost:8080/api/v1/auth/oauth/github/callback")
	cfg.OAuth.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	cfg.OAuth.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	cfg.OAuth.GoogleRedirectURL = getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/oauth/google/callback")

	// AI 配置
	cfg.AI.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	cfg.AI.OpenAIModel = getEnv("OPENAI_MODEL", "gpt-4o")
	cfg.AI.EinoAgentMaxTurns = getEnvInt("EINO_AGENT_MAX_TURNS", 10)

	// 可观测性配置
	cfg.Observ.OTELExporterEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	cfg.Observ.OTELServiceName = getEnv("OTEL_SERVICE_NAME", "portunus")
	cfg.Observ.PrometheusMetricsPort = getEnv("PROMETHEUS_METRICS_PORT", "9090")

	// 限流配置
	cfg.RateLimit.RPM = getEnvInt("RATE_LIMIT_RPM", 60)
	cfg.RateLimit.Burst = getEnvInt("RATE_LIMIT_BURST", 10)

	// 忽略 caarlos0/env 的返回值，因为我们使用自定义 getEnv 函数
	_ = env.Parse(cfg)

	return cfg, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值。
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt 获取整数类型环境变量，如果不存在或解析失败则返回默认值。
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}

// getEnvBool 获取布尔类型环境变量，如果不存在或解析失败则返回默认值。
func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

// getEnvDuration 获取时长类型环境变量，如果不存在或解析失败则返回默认值。
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}