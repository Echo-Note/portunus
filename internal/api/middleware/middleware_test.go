package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/internal/config"
	"github.com/Echo-Note/portunus/internal/service"
)

// setupMiddlewareTest 创建测试用的 Gin 路由和 UserService。
func setupMiddlewareTest(t *testing.T) (*gin.Engine, *service.UserService) {
	gin.SetMode(gin.TestMode)

	cfg := config.DatabaseConfig{
		URL:             "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
	}
	client, err := config.NewEntClient(t.Context(), cfg)
	require.NoError(t, err)

	jwtCfg := config.JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	}
	privateKeyFile := os.Getenv("JWT_PRIVATE_KEY_FILE")
	if privateKeyFile == "" {
		privateKeyFile = "../../../certs/jwt-private.pem"
	}
	publicKeyFile := os.Getenv("JWT_PUBLIC_KEY_FILE")
	if publicKeyFile == "" {
		publicKeyFile = "../../../certs/jwt-public.pem"
	}
	if data, err := os.ReadFile(privateKeyFile); err == nil {
		jwtCfg.PrivateKey = string(data)
	}
	if data, err := os.ReadFile(publicKeyFile); err == nil {
		jwtCfg.PublicKey = string(data)
	}

	userSvc, err := service.NewUserService(client, jwtCfg)
	require.NoError(t, err)

	router := gin.New()
	router.Use(RequestID())
	router.Use(Logging())

	t.Cleanup(func() { client.Close() })
	return router, userSvc
}

// TestAuthMiddleware_MissingHeader 测试缺少 Authorization 头。
func TestAuthMiddleware_MissingHeader(t *testing.T) {
	router, userSvc := setupMiddlewareTest(t)
	router.GET("/test", AuthMiddleware(userSvc), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_InvalidFormat 测试错误的 Authorization 格式。
func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	router, userSvc := setupMiddlewareTest(t)
	router.GET("/test", AuthMiddleware(userSvc), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_InvalidToken 测试无效的 Token。
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	router, userSvc := setupMiddlewareTest(t)
	router.GET("/test", AuthMiddleware(userSvc), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_ValidToken 测试有效 Token 通过认证。
func TestAuthMiddleware_ValidToken(t *testing.T) {
	router, userSvc := setupMiddlewareTest(t)

	// 注册并激活用户
	ctx := t.Context()
	_, err := userSvc.Register(ctx, service.RegisterInput{
		Email: "auth-test@example.com", Password: "test123456",
	})
	require.NoError(t, err)

	// 激活用户
	userSvc.ActivateUserByEmail(ctx, "auth-test@example.com")

	// 登录获取 token
	pair, err := userSvc.Login(ctx, service.LoginInput{
		Email: "auth-test@example.com", Password: "test123456",
	})
	require.NoError(t, err)

	router.GET("/test", AuthMiddleware(userSvc), func(c *gin.Context) {
		userID, ok := GetUserID(c)
		assert.True(t, ok)
		c.JSON(200, gin.H{"user_id": userID.String()})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRequestID 测试 RequestID 中间件。
func TestRequestID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		id := GetRequestID(c)
		assert.NotEmpty(t, id)
		c.JSON(200, gin.H{"request_id": id})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

// TestRequestID_PreserveExisting 测试保留已有的 RequestID。
func TestRequestID_PreserveExisting(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		id := GetRequestID(c)
		assert.Equal(t, "my-custom-id", id)
		c.JSON(200, gin.H{"request_id": id})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestCORS 测试 CORS 中间件。
func TestCORS(t *testing.T) {
	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORSOptions 测试 OPTIONS 预检请求。
func TestCORSOptions(t *testing.T) {
	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestRateLimit 测试限流中间件。
func TestRateLimit(t *testing.T) {
	router := gin.New()
	router.Use(RateLimit(100, 5))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 在 burst 范围内应全部通过
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "请求 %d 应通过", i+1)
	}

	// 第 6 个请求应被限流
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "burst 超出后应被限流")
}

// TestProjectContext 测试 ProjectContext 中间件。
func TestProjectContext(t *testing.T) {
	router := gin.New()
	router.GET("/projects/:projectID/test", ProjectContext(), func(c *gin.Context) {
		pid := GetProjectIDFromCtx(c)
		assert.NotEqual(t, "", pid.String())
		c.JSON(200, gin.H{"project_id": pid.String()})
	})

	req := httptest.NewRequest("GET", "/projects/00000000-0000-0000-0000-000000000001/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetUserID_NotSet 测试未设置 userID 时返回 false。
func TestGetUserID_NotSet(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		_, ok := GetUserID(c)
		assert.False(t, ok)
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}