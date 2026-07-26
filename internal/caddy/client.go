// Package caddy 提供 Caddy Admin API 客户端，支持 mTLS 双向认证。
// 日常配置变更通过 /id/<name>/... 端点 + If-Match 头实现细粒度并发控制，
// POST /load 仅用于灾难恢复场景。
package caddy

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Echo-Note/portunus/internal/config"
)

// Client Caddy Admin API 客户端，封装 HTTP 调用和 mTLS 认证。
type Client struct {
	httpClient *http.Client
	adminURL   string
	retryMax   int
}

// New 创建 Caddy Admin API 客户端。
// 加载 mTLS 证书并配置 HTTP 传输层。
func New(cfg config.CaddyConfig) (*Client, error) {
	// 加载 CA 证书池
	caCert, err := os.ReadFile(cfg.MTLSCA)
	if err != nil {
		return nil, fmt.Errorf("caddy client: 读取 CA 证书失败 %s: %w", cfg.MTLSCA, err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("caddy client: 解析 CA 证书失败")
	}

	// 加载客户端证书
	clientCert, err := tls.LoadX509KeyPair(cfg.MTLSCert, cfg.MTLSKey)
	if err != nil {
		return nil, fmt.Errorf("caddy client: 加载客户端证书失败: %w", err)
	}

	// 配置 mTLS 传输层
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{clientCert},
			MinVersion:   tls.VersionTLS12,
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.RequestTimeout,
		},
		adminURL: cfg.AdminURL,
		retryMax: cfg.RetryMax,
	}, nil
}

// NewInsecure 创建不验证 TLS 的客户端（仅用于开发环境）。
func NewInsecure(adminURL string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec // 仅开发环境使用
				},
			},
			Timeout: timeout,
		},
		adminURL: adminURL,
		retryMax: 3,
	}
}

// GetID 发送 GET /id/<name> 请求，返回响应体。
func (c *Client) GetID(ctx context.Context, name string) ([]byte, string, error) {
	url := fmt.Sprintf("%s/id/%s", c.adminURL, name)
	return c.doRequest(ctx, http.MethodGet, url, nil, "")
}

// PostID 发送 POST /id/<name> 请求，创建新的配置节点。
// etag 为空表示不使用乐观并发控制。
func (c *Client) PostID(ctx context.Context, name string, body any, etag string) ([]byte, error) {
	url := fmt.Sprintf("%s/id/%s", c.adminURL, name)
	return c.doRequestWithBody(ctx, http.MethodPost, url, body, etag)
}

// PatchID 发送 PATCH /id/<name> 请求，部分更新配置节点。
// 必须提供 etag 用于乐观并发控制。
func (c *Client) PatchID(ctx context.Context, name string, body any, etag string) ([]byte, error) {
	url := fmt.Sprintf("%s/id/%s", c.adminURL, name)
	return c.doRequestWithBody(ctx, http.MethodPatch, url, body, etag)
}

// DeleteID 发送 DELETE /id/<name> 请求，删除配置节点。
// 必须提供 etag 用于乐观并发控制。
func (c *Client) DeleteID(ctx context.Context, name string, etag string) error {
	url := fmt.Sprintf("%s/id/%s", c.adminURL, name)
	_, _, err := c.doRequest(ctx, http.MethodDelete, url, nil, etag)
	return err
}

// GetConfig 发送 GET /config/ 请求，获取 Caddy 全量配置。
func (c *Client) GetConfig(ctx context.Context) ([]byte, error) {
	url := fmt.Sprintf("%s/config/", c.adminURL)
	body, _, err := c.doRequest(ctx, http.MethodGet, url, nil, "")
	return body, err
}

// PostLoad 发送 POST /load 请求，全量替换 Caddy 配置。
// 仅用于灾难恢复场景，日常变更请使用 /id/ 端点。
func (c *Client) PostLoad(ctx context.Context, configJSON []byte) error {
	url := fmt.Sprintf("%s/load", c.adminURL)
	_, err := c.doRequestWithBody(ctx, http.MethodPost, url, json.RawMessage(configJSON), "")
	return err
}

// GetUpstreams 获取所有上游健康状态。
// 返回的列表是全局的，控制面需要按 dial address 过滤。
func (c *Client) GetUpstreams(ctx context.Context) ([]byte, error) {
	url := fmt.Sprintf("%s/reverse_proxy/upstreams", c.adminURL)
	body, _, err := c.doRequest(ctx, http.MethodGet, url, nil, "")
	return body, err
}

// doRequestWithBody 发送带 JSON 请求体的 HTTP 请求。
func (c *Client) doRequestWithBody(ctx context.Context, method, url string, body any, etag string) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("caddy client: 序列化请求体失败: %w", err)
	}
	respBody, _, err := c.doRequest(ctx, method, url, bytes.NewReader(jsonBody), etag)
	return respBody, err
}

// doRequest 发送 HTTP 请求并处理重试与错误。
// 返回响应体字节和 Etag 头。
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader, etag string) ([]byte, string, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryMax; attempt++ {
		if attempt > 0 {
			slog.WarnContext(ctx, "caddy client: 重试请求",
				"method", method,
				"url", url,
				"attempt", attempt,
				"err", lastErr,
			)
			// 指数退避：1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, "", fmt.Errorf("caddy client: 创建请求失败: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if etag != "" {
			req.Header.Set("If-Match", etag)
		}

		// 如果 body 是 bytes.Reader 或 strings.Reader，需要重置
		if seeker, ok := body.(io.Seeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return nil, "", fmt.Errorf("caddy client: 重置请求体失败: %w", err)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("caddy client: 请求执行失败: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		if cerr := resp.Body.Close(); cerr != nil {
			slog.WarnContext(ctx, "caddy client: 关闭响应体失败", "err", cerr)
		}
		if err != nil {
			lastErr = fmt.Errorf("caddy client: 读取响应失败: %w", err)
			continue
		}

		// 2xx 视为成功
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, resp.Header.Get("Etag"), nil
		}

		// 412 Precondition Failed 不重试（etag 不匹配，说明并发冲突）
		if resp.StatusCode == http.StatusPreconditionFailed {
			return nil, "", fmt.Errorf("%w: etag 不匹配", ErrPreconditionFailed)
		}

		lastErr = fmt.Errorf("caddy client: 非预期响应 %d: %s", resp.StatusCode, string(respBody))

		// 4xx（除 412 外）不重试
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 412 {
			return nil, "", lastErr
		}
	}

	return nil, "", fmt.Errorf("caddy client: 重试 %d 次后仍失败: %w", c.retryMax, lastErr)
}
