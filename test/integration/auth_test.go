package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Echo-Note/portunus/internal/api/dto"
)

// ── 认证链路测试 ──

// TestAuth_RegisterLogin 测试完整的注册 → 激活 → 登录 → 获取信息 → 刷新 → 登出链路。
func TestAuth_RegisterLogin(t *testing.T) {
	email := "auth-register-login@example.com"
	password := "test123456"

	// 1. 注册
	regBody := `{"email":"` + email + `","password":"` + password + `"}`
	resp := doRequest(t, "POST", "/api/v1/auth/register", regBody)
	assert.Equal(t, http.StatusCreated, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, email, data["email"])
	assert.NotEmpty(t, data["user_id"])

	// 2. 未激活用户登录应失败
	loginBody := `{"email":"` + email + `","password":"` + password + `"}`
	resp = doRequest(t, "POST", "/api/v1/auth/login", loginBody)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ = parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)

	// 3. 激活用户
	registerAndActivateUser(t, email, password)

	// 4. 登录
	accessToken, refreshToken := loginUser(t, email, password)

	// 5. 获取用户信息
	resp = doAuthedRequest(t, "GET", "/api/v1/me", "", accessToken)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data = parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, email, data["email"])

	// 6. 刷新令牌
	refreshBody := `{"refresh_token":"` + refreshToken + `"}`
	resp = doRequest(t, "POST", "/api/v1/auth/refresh", refreshBody)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data = parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.NotEmpty(t, data["access_token"])
	// 新的 access_token 应该与旧的不同
	newAccessToken := data["access_token"].(string)
	assert.NotEqual(t, accessToken, newAccessToken)

	// 7. 使用新 token 获取用户信息（验证刷新后的 token 有效）
	resp = doAuthedRequest(t, "GET", "/api/v1/me", "", newAccessToken)
	assert.Equal(t, http.StatusOK, resp.Code)

	// 8. 退出登录
	resp = doAuthedRequest(t, "POST", "/api/v1/auth/logout", "", accessToken)
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data = parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, "已退出登录", data["message"])
}

// TestAuth_WrongPassword 测试错误密码登录返回 401。
func TestAuth_WrongPassword(t *testing.T) {
	email := "auth-wrong-pass@example.com"
	password := "test123456"

	registerAndActivateUser(t, email, password)

	// 用错误密码登录
	loginBody := `{"email":"` + email + `","password":"wrongpassword"}`
	resp := doRequest(t, "POST", "/api/v1/auth/login", loginBody)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
}

// TestAuth_UnactivatedUser 测试未激活用户登录返回 401。
func TestAuth_UnactivatedUser(t *testing.T) {
	email := "auth-unactivated@example.com"
	password := "test123456"

	// 注册但不激活
	regBody := `{"email":"` + email + `","password":"` + password + `"}`
	resp := doRequest(t, "POST", "/api/v1/auth/register", regBody)
	assert.Equal(t, http.StatusCreated, resp.Code)

	// 登录应失败
	loginBody := `{"email":"` + email + `","password":"` + password + `"}`
	resp = doRequest(t, "POST", "/api/v1/auth/login", loginBody)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
}

// TestAuth_InvalidToken 测试无效 token 返回 401。
func TestAuth_InvalidToken(t *testing.T) {
	// 使用垃圾 token
	resp := doAuthedRequest(t, "GET", "/api/v1/me", "", "garbage-token")
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
}

// TestAuth_NoToken 测试无 Authorization 头返回 401。
func TestAuth_NoToken(t *testing.T) {
	resp := doRequest(t, "GET", "/api/v1/me", "")
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
}

// TestAuth_DuplicateRegistration 测试重复注册返回 409。
func TestAuth_DuplicateRegistration(t *testing.T) {
	email := "auth-dup-reg@example.com"
	password := "test123456"

	regBody := `{"email":"` + email + `","password":"` + password + `"}`

	// 首次注册
	resp := doRequest(t, "POST", "/api/v1/auth/register", regBody)
	assert.Equal(t, http.StatusCreated, resp.Code)

	// 重复注册
	resp = doRequest(t, "POST", "/api/v1/auth/register", regBody)
	assert.Equal(t, http.StatusConflict, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeConflict, code)
}

// TestAuth_RefreshTokenFlow 测试使用 refresh_token 刷新 access_token 的完整流程。
func TestAuth_RefreshTokenFlow(t *testing.T) {
	email := "auth-refresh-flow@example.com"
	password := "test123456"

	registerAndActivateUser(t, email, password)
	accessToken, refreshToken := loginUser(t, email, password)

	// 使用 refresh_token 获取新的 token pair
	refreshBody := `{"refresh_token":"` + refreshToken + `"}`
	resp := doRequest(t, "POST", "/api/v1/auth/refresh", refreshBody)
	assert.Equal(t, http.StatusOK, resp.Code)

	var refreshResp struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &refreshResp))
	assert.Equal(t, 0, refreshResp.Code)
	assert.NotEmpty(t, refreshResp.Data.AccessToken)
	assert.NotEqual(t, accessToken, refreshResp.Data.AccessToken)
	assert.Equal(t, "Bearer", refreshResp.Data.TokenType)

	// 使用新 access_token 验证有效性
	resp = doAuthedRequest(t, "GET", "/api/v1/me", "", refreshResp.Data.AccessToken)
	assert.Equal(t, http.StatusOK, resp.Code)
}

// TestAuth_InvalidRefreshToken 测试无效的 refresh_token 返回 401。
func TestAuth_InvalidRefreshToken(t *testing.T) {
	refreshBody := `{"refresh_token":"invalid-refresh-token"}`
	resp := doRequest(t, "POST", "/api/v1/auth/refresh", refreshBody)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	code, _ := parseEnvelope(t, resp)
	assert.Equal(t, dto.CodeUnauthorized, code)
}

// TestAuth_RegisterValidation 测试注册参数校验。
func TestAuth_RegisterValidation(t *testing.T) {
	// 缺少 email
	resp := doRequest(t, "POST", "/api/v1/auth/register", `{"password":"test123456"}`)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// 密码太短
	resp = doRequest(t, "POST", "/api/v1/auth/register", `{"email":"test@example.com","password":"123"}`)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// 无效 email 格式
	resp = doRequest(t, "POST", "/api/v1/auth/register", `{"email":"not-an-email","password":"test123456"}`)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

// TestAuth_LoginValidation 测试登录参数校验。
func TestAuth_LoginValidation(t *testing.T) {
	// 缺少 email
	resp := doRequest(t, "POST", "/api/v1/auth/login", `{"password":"test123456"}`)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// 缺少 password
	resp = doRequest(t, "POST", "/api/v1/auth/login", `{"email":"test@example.com"}`)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

// TestHealth_Check 测试健康检查端点。
func TestHealth_Check(t *testing.T) {
	// /health
	resp := doRequest(t, "GET", "/health", "")
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data := parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, "ok", data["status"])

	// /ready
	resp = doRequest(t, "GET", "/ready", "")
	assert.Equal(t, http.StatusOK, resp.Code)
	code, data = parseEnvelope(t, resp)
	assert.Equal(t, 0, code)
	assert.Equal(t, "ready", data["status"])
}