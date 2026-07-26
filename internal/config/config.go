// Package config 提供应用配置加载与管理。
// 所有配置项通过环境变量注入，遵循 12-Factor App 原则。
package config

import (
	"time"
)

// Config 应用顶层配置，聚合所有子配置。
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Caddy     CaddyConfig
	MinIO     MinIOConfig
	SMTP      SMTPConfig
	OAuth     OAuthConfig
	AI        AIConfig
	Observ    ObservabilityConfig
	RateLimit RateLimitConfig
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port    string // 监听端口，默认 8080
	GinMode string // Gin 运行模式：debug / release / test
}

// DatabaseConfig PostgreSQL 数据库连接配置。
type DatabaseConfig struct {
	URL             string        // 数据库连接字符串
	MaxOpenConns    int           // 最大连接数，默认 25
	MaxIdleConns    int           // 最大空闲连接数，默认 5
	ConnMaxLifetime time.Duration // 连接最大存活时间，默认 5m
}

// RedisConfig Redis 连接配置。
type RedisConfig struct {
	URL      string // Redis 连接字符串
	Password string // Redis 密码（可选）
}

// JWTConfig JWT 认证配置。
type JWTConfig struct {
	PrivateKey      string        // RS256 私钥（PEM 格式）
	PublicKey       string        // RS256 公钥（PEM 格式）
	AccessTokenTTL  time.Duration // access_token 有效期，默认 15m
	RefreshTokenTTL time.Duration // refresh_token 有效期，默认 168h (7d)
}

// CaddyConfig Caddy Admin API 客户端配置。
type CaddyConfig struct {
	AdminURL       string        // Caddy Admin API 地址，默认 https://caddy:2021
	MTLSCert       string        // mTLS 客户端证书路径
	MTLSKey        string        // mTLS 客户端密钥路径
	MTLSCA         string        // mTLS CA 证书路径
	RequestTimeout time.Duration // 请求超时，默认 10s
	RetryMax       int           // 最大重试次数，默认 3
}

// MinIOConfig MinIO 对象存储配置。
type MinIOConfig struct {
	Endpoint  string // MinIO 服务地址
	AccessKey string // 访问密钥
	SecretKey string // 秘密密钥
	Bucket    string // 存储桶名称
	UseSSL    bool   // 是否使用 SSL
}

// SMTPConfig 邮件发送配置。
type SMTPConfig struct {
	Host     string // SMTP 服务器地址
	Port     int    // SMTP 端口，默认 587
	Username string // SMTP 认证用户名
	Password string // SMTP 认证密码
	From     string // 发件人邮箱
	FromName string // 发件人显示名称
}

// OAuthConfig OAuth 第三方登录配置。
type OAuthConfig struct {
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

// AIConfig AI Agent 配置。
type AIConfig struct {
	OpenAIAPIKey      string
	OpenAIModel       string
	EinoAgentMaxTurns int
}

// ObservabilityConfig 可观测性配置。
type ObservabilityConfig struct {
	OTELExporterEndpoint  string
	OTELServiceName       string
	PrometheusMetricsPort string
}

// RateLimitConfig 限流配置。
type RateLimitConfig struct {
	RPM   int // 每分钟请求数，默认 60
	Burst int // 突发请求数，默认 10
}
