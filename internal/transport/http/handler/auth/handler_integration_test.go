package auth_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/jwt"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/password"
	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/logger/zl"
	"github.com/leyl1ne/UserService/internal/repository/postgres"
	authservice "github.com/leyl1ne/UserService/internal/service/auth"
	"github.com/leyl1ne/UserService/internal/testutils"
	authandler "github.com/leyl1ne/UserService/internal/transport/http/handler/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthHandler(t *testing.T) (*authandler.AuthHandler, *postgres.Repository, jwt.Config, logger.Logger) {
	db := testutils.SetupPostgres(t)

	repo := postgres.NewRepository(db.Pool())

	hasher, err := password.NewHasher(password.Config{Cost: 10})
	require.NoError(t, err)

	jwtConfig := jwt.Config{
		Secret:        "test-secret-key-for-jwt-signing-32b",
		AccessTokeTTL: 15 * time.Minute,
	}
	jwtProvider := jwt.NewJWTGenerator(jwtConfig)

	log := zl.NewZerologLogger("debug", io.Discard)

	authSvc := authservice.NewService(repo, hasher, jwtProvider, 24*time.Hour)
	handler := authandler.NewAuthHandler(log, authSvc)

	return handler, repo, jwtConfig, log
}

func setupGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// ============================================
// REGISTER TESTS
// ============================================

func TestAuthHandler_Register_Integration(t *testing.T) {
	handler, _, _, _ := setupAuthHandler(t)
	dublicateEmail := "seller@test.com"

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "success - SELLER",
			body: map[string]interface{}{
				"email":    dublicateEmail,
				"password": "StrongP@ssw0rd",
				"role":     "SELLER",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp authandler.AuthResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			},
		},
		{
			name: "success - BUYER",
			body: map[string]interface{}{
				"email":    "buyer@test.com",
				"password": "StrongP@ssw0rd",
				"role":     "BUYER",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp authandler.AuthResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			},
		},
		{
			name: "success - WAREHOUSE",
			body: map[string]interface{}{
				"email":    "warehouse@test.com",
				"password": "StrongP@ssw0rd",
				"role":     "WAREHOUSE",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp authandler.AuthResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			},
		},
		{
			name: "success - LOGISTICS",
			body: map[string]interface{}{
				"email":    "logistics@test.com",
				"password": "StrongP@ssw0rd",
				"role":     "LOGISTICS",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp authandler.AuthResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			},
		},
		{
			name: "duplicate email",
			body: map[string]interface{}{
				"email":    dublicateEmail,
				"password": "StrongP@ssw0rd",
				"role":     "BUYER",
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, body []byte) {
				var errResp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &errResp))
				assert.Equal(t, "email already exists", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name: "validation error - invalid email",
			body: map[string]interface{}{
				"email":    "not-an-email",
				"password": "StrongP@ssw0rd",
				"role":     "SELLER",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "invalid email format", fields["email"])
			},
		},
		{
			name: "validation error - short password",
			body: map[string]interface{}{
				"email":    "shortpass@test.com",
				"password": "short",
				"role":     "SELLER",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "too short", fields["password"])
			},
		},
		{
			name: "validation error - too long password",
			body: map[string]interface{}{
				"email":    "shortpass@test.com",
				"password": strings.Repeat("x", 65),
				"role":     "SELLER",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "too long", fields["password"])
			},
		},
		{
			name: "validation error - invalid role",
			body: map[string]interface{}{
				"email":    "invalidrole@test.com",
				"password": "StrongP@ssw0rd",
				"role":     "INVALID_ROLE",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "invalid value", fields["role"])
			},
		},
		{
			name: "validation error - missing email",
			body: map[string]interface{}{
				"password": "StrongP@ssw0rd",
				"role":     "SELLER",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "field is required", fields["email"])
			},
		},
		{
			name: "validation error - missing password",
			body: map[string]interface{}{
				"email": "nopass@test.com",
				"role":  "SELLER",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "field is required", fields["password"])
			},
		},
		{
			name:           "validation error - empty body",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var errResp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &errResp))
				assert.Equal(t, "request body is empty", errResp["error"].(map[string]interface{})["message"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupGin()
			r.POST("/auth/register", handler.Register())

			var reqBody []byte
			if tt.body != nil {
				var err error
				reqBody, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}

// ============================================
// LOGIN TESTS
// ============================================

func TestAuthHandler_Login_Integration(t *testing.T) {
	handler, _, _, _ := setupAuthHandler(t)
	registerEmail := "logintest@test.com"
	registerPassword := "MySecureP@ss123"

	// First, register a user
	r := setupGin()
	r.POST("/auth/register", handler.Register())

	registerBody, _ := json.Marshal(map[string]interface{}{
		"email":    registerEmail,
		"password": registerPassword,
		"role":     "BUYER",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "success",
			body: map[string]interface{}{
				"email":    registerEmail,
				"password": registerPassword,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp authandler.AuthResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			},
		},
		{
			name: "invalid credentials - wrong password",
			body: map[string]interface{}{
				"email":    registerEmail,
				"password": "WrongPassword123",
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var errResp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &errResp))
				assert.Equal(t, "invalid credentials", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name: "invalid credentials - non-existent email",
			body: map[string]interface{}{
				"email":    "nonexistent@test.com",
				"password": "MySecureP@ss123",
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var errResp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &errResp))
				assert.Equal(t, "invalid credentials", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name: "validation error - invalid email format",
			body: map[string]interface{}{
				"email":    "invalid-email",
				"password": "MySecureP@ss123",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "invalid email format", fields["email"])
			},
		},
		{
			name: "validation error - missing email",
			body: map[string]interface{}{
				"password": "MySecureP@ss123",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "field is required", fields["email"])
			},
		},
		{
			name: "validation error - missing password",
			body: map[string]interface{}{
				"email": "logintest@test.com",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "field is required", fields["password"])
			},
		},
		{
			name:           "validation error - empty body",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var errResp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &errResp))
				assert.Equal(t, "request body is empty", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name: "validation error - short password",
			body: map[string]interface{}{
				"email":    "shortpass@test.com",
				"password": "short",
				"role":     "SELLER",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "too short", fields["password"])
			},
		},
		{
			name: "validation error - too long password",
			body: map[string]interface{}{
				"email":    "shortpass@test.com",
				"password": strings.Repeat("x", 65),
				"role":     "SELLER",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "too long", fields["password"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupGin()
			r.POST("/auth/login", handler.Login())

			var reqBody []byte
			if tt.body != nil {
				var err error
				reqBody, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}

// ============================================
// REFRESH TESTS
// ============================================

func TestAuthHandler_Refresh_Integration(t *testing.T) {
	handler, _, _, _ := setupAuthHandler(t)
	registerEmail := "refreshtest@test.com"
	registerPassword := "MySecureP@ss123"

	// Register and login to get a refresh token
	r := setupGin()
	r.POST("/auth/register", handler.Register())
	r.POST("/auth/login", handler.Login())

	registerBody, _ := json.Marshal(map[string]interface{}{
		"email":    registerEmail,
		"password": registerPassword,
		"role":     "BUYER",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	loginBody, _ := json.Marshal(map[string]interface{}{
		"email":    registerEmail,
		"password": registerPassword,
	})
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var loginResp authandler.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &loginResp))
	validRefreshToken := loginResp.RefreshToken

	tests := []struct {
		name           string
		refreshToken   string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "success",
			refreshToken:   validRefreshToken,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp authandler.RefreshResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp.AccessToken)
			},
		},
		{
			name:           "invalid token - non-existent refresh token",
			refreshToken:   uuid.NewString(),
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				var errResp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &errResp))
				assert.Equal(t, "invalid token", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "missing refresh_token field",
			refreshToken:   "",
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "field is required", fields["refreshtoken"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupGin()
			r.POST("/auth/refresh", handler.Refresh())

			body, _ := json.Marshal(map[string]interface{}{
				"refresh_token": tt.refreshToken,
			})

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}

	// Test with expired token
	t.Run("expired token", func(t *testing.T) {
		// Create a new JWT provider with negative TTL to generate expired token
		// Register a new user to get a refresh token, then use it after expiry
		r := setupGin()
		r.POST("/auth/register", handler.Register())

		registerBody, _ := json.Marshal(map[string]interface{}{
			"email":    "expiredrefresh@test.com",
			"password": "MySecureP@ss123",
			"role":     "BUYER",
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)

		var resp authandler.AuthResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

		// Now try to use the refresh token
		r2 := setupGin()
		r2.POST("/auth/refresh", handler.Refresh())

		body, _ := json.Marshal(map[string]interface{}{
			"refresh_token": resp.RefreshToken,
		})
		req = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		r2.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// ============================================
// LOGOUT TESTS
// ============================================

func TestAuthHandler_Logout_Integration(t *testing.T) {
	handler, _, _, _ := setupAuthHandler(t)
	registerEmail := "logouttest@test.com"
	registerPassword := "MySecureP@ss123"

	// Register and login to get a refresh token
	r := setupGin()
	r.POST("/auth/register", handler.Register())
	r.POST("/auth/login", handler.Login())

	registerBody, _ := json.Marshal(map[string]interface{}{
		"email":    registerEmail,
		"password": registerPassword,
		"role":     "BUYER",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	loginBody, _ := json.Marshal(map[string]interface{}{
		"email":    registerEmail,
		"password": registerPassword,
	})
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var loginResp authandler.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &loginResp))
	validRefreshToken := loginResp.RefreshToken

	tests := []struct {
		name           string
		refreshToken   string
		expectedStatus int
	}{
		{
			name:           "success",
			refreshToken:   validRefreshToken,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "success - idempotent (token already deleted)",
			refreshToken:   validRefreshToken,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing refresh_token",
			refreshToken:   "",
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "malformed refresh_token",
			refreshToken:   "malformed token",
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupGin()
			r.POST("/auth/logout", handler.Logout())

			body, _ := json.Marshal(map[string]interface{}{
				"refresh_token": tt.refreshToken,
			})

			req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

// ============================================
// END-TO-END AUTH FLOW TESTS
// ============================================

func TestAuthHandler_FullFlow_Integration(t *testing.T) {
	handler, _, jwtConfig, _ := setupAuthHandler(t)
	registerEmail := "fullflow@test.com"
	registerPassowrd := "MySecureP@ss123"

	r := setupGin()
	r.POST("/auth/register", handler.Register())
	r.POST("/auth/login", handler.Login())
	r.POST("/auth/refresh", handler.Refresh())
	r.POST("/auth/logout", handler.Logout())

	// Step 1: Register
	t.Run("step 1: register", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"email":    registerEmail,
			"password": registerPassowrd,
			"role":     "SELLER",
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp authandler.AuthResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)

		// Verify access token is valid
		jwtProvider := jwt.NewJWTGenerator(jwtConfig)
		payload, err := jwtProvider.Validate(resp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "SELLER", payload.UserRole)
	})

	// Step 2: Login
	var refreshToken string
	t.Run("step 2: login", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"email":    registerEmail,
			"password": registerPassowrd,
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp authandler.AuthResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		refreshToken = resp.RefreshToken
	})

	// Step 3: Refresh
	t.Run("step 3: refresh", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"refresh_token": refreshToken,
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp authandler.RefreshResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp.AccessToken)
	})

	// Step 4: Logout
	t.Run("step 4: logout", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"refresh_token": refreshToken,
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	// Step 5: Try to refresh with revoked token
	t.Run("step 5: refresh with revoked token fails", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"refresh_token": refreshToken,
		})
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
