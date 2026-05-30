package user_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/jwt"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/password"
	postgresInfra "github.com/leyl1ne/UserService/internal/infrastructure/postgres"
	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/logger/zl"
	"github.com/leyl1ne/UserService/internal/repository/postgres"
	authservice "github.com/leyl1ne/UserService/internal/service/auth"
	companyservice "github.com/leyl1ne/UserService/internal/service/company"
	userservice "github.com/leyl1ne/UserService/internal/service/user"
	"github.com/leyl1ne/UserService/internal/testutils"
	authandler "github.com/leyl1ne/UserService/internal/transport/http/handler/auth"
	companyhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/company"
	userhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/user"
	"github.com/leyl1ne/UserService/internal/transport/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserTest(t *testing.T) (*userhandler.UserHandler, *authandler.AuthHandler, jwt.Config, *postgresInfra.Postgres) {
	db := testutils.SetupPostgres(t)
	repo := postgres.NewRepository(db.Pool())

	hasher, err := password.NewHasher(password.Config{Cost: 10})
	require.NoError(t, err)

	jwtConfig := jwt.Config{
		Secret:        "test-secret-key-for-jwt-signing-32b",
		AccessTokeTTL: 15 * time.Minute,
	}
	jwtProvider := jwt.NewJWTGenerator(jwtConfig)

	log := setupLogger(t)

	authSvc := authservice.NewService(repo, hasher, jwtProvider, 24*time.Hour)
	userSvc := userservice.NewService(repo)
	companySvc := companyservice.NewService(repo)

	authHandler := authandler.NewAuthHandler(log, authSvc)
	userHandler := userhandler.NewUserHandler(log, userSvc)
	companyHandler := companyhandler.NewCompanyHandler(log, companySvc)

	// Register auth handler to create test users
	_ = companyHandler
	_ = authSvc

	return userHandler, authHandler, jwtConfig, db
}
func setupLogger(t *testing.T) logger.Logger {
	log, err := zl.NewZerologLogger(logger.Config{
		Level:  "debug",
		Format: "",
		Output: "discard",
	})
	require.NoError(t, err)
	return log
}

func generateJWTToken(t *testing.T, config jwt.Config, userID, role, companyID string) string {
	jwtProvider := jwt.NewJWTGenerator(config)
	token, err := jwtProvider.GenerateAccessToken(jwt.Payload{
		UserID:    userID,
		UserRole:  role,
		CompanyID: companyID,
	})
	require.NoError(t, err)
	return token
}

func setupAuthenticatedGin(t *testing.T, jwtConfig jwt.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtProvider := jwt.NewJWTGenerator(jwtConfig)
	router.Use(middleware.AuthMiddleware(setupLogger(t), jwtProvider))

	return router
}

func parseErrorResponse(t *testing.T, body []byte) map[string]interface{} {
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

// Helper to register a user and return their ID + JWT token
func registerTestUser(t *testing.T, handler *authandler.AuthHandler, user authandler.RegisterRequest) (userID uuid.UUID, jwtToken string, refreshToken string) {
	r := gin.New()
	r.POST("/auth/register", handler.Register())

	body, _ := json.Marshal(map[string]interface{}{
		"email":    user.Email,
		"password": user.Password,
		"role":     user.Role,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp authandler.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	// Parse userID from access token
	jwtProvider := jwt.NewJWTGenerator(jwt.Config{
		Secret:        "test-secret-key-for-jwt-signing-32b",
		AccessTokeTTL: 15 * time.Minute,
	})
	payload, err := jwtProvider.Validate(resp.AccessToken)
	require.NoError(t, err)

	userID, err = uuid.Parse(payload.UserID)
	require.NoError(t, err)

	return userID, resp.AccessToken, resp.RefreshToken
}

// ============================================
// GET CURRENT USER TESTS
// ============================================

func TestUserHandler_GetCurrentUser_Integration(t *testing.T) {
	userHandler, authHandler, jwtConfig, _ := setupUserTest(t)

	user1 := authandler.RegisterRequest{
		Email:    "currentuser1@test.com",
		Password: "MySecureP@ss123",
		Role:     "SELLER",
	}

	user2 := authandler.RegisterRequest{
		Email:    "currentuser2@test.com",
		Password: "MySecureP@ss123",
		Role:     "BUYER",
	}

	// Register test users
	userID1, token1, _ := registerTestUser(t, authHandler, user1)
	_, token2, _ := registerTestUser(t, authHandler, user2)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "success - user 1",
			token:          token1,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp userhandler.UserResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, userID1, resp.ID)
				assert.Equal(t, user1.Email, resp.Email)
				assert.Equal(t, user1.Role, resp.Role)
			},
		},
		{
			name:           "success - user 2",
			token:          token2,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp userhandler.UserResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, user2.Email, resp.Email)
				assert.Equal(t, user2.Role, resp.Role)
			},
		},
		{
			name:           "unauthorized - missing token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "authorization header is required", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "unauthorized - invalid token format",
			token:          "not-a-valid-token",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "invalid token", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "unauthorized - expired token",
			token:          generateJWTToken(t, jwt.Config{Secret: jwtConfig.Secret, AccessTokeTTL: -1 * time.Second}, userID1.String(), "SELLER", ""),
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "token expired", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "unauthorized - wrong secret",
			token:          generateJWTToken(t, jwt.Config{Secret: "wrong-secret-key-for-jwt-signing!", AccessTokeTTL: 15 * time.Minute}, userID1.String(), "SELLER", ""),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupAuthenticatedGin(t, jwtConfig)
			router.GET("/users/me", userHandler.GetCurrentUser())

			req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}

// ============================================
// GET USER BY ID TESTS
// ============================================

func TestUserHandler_GetUser_Integration(t *testing.T) {
	userHandler, authHandler, jwtConfig, _ := setupUserTest(t)
	user1 := authandler.RegisterRequest{
		Email:    "currentuser1@test.com",
		Password: "MySecureP@ss123",
		Role:     "SELLER",
	}

	user2 := authandler.RegisterRequest{
		Email:    "currentuser2@test.com",
		Password: "MySecureP@ss123",
		Role:     "BUYER",
	}

	// Register test users
	userID1, token1, _ := registerTestUser(t, authHandler, user1)
	userID2, _, _ := registerTestUser(t, authHandler, user2)

	tests := []struct {
		name           string
		token          string
		userID         string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "success - get existing user",
			token:          token1,
			userID:         userID2.String(),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp userhandler.UserResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, userID2, resp.ID)
				assert.Equal(t, user2.Email, resp.Email)
				assert.Equal(t, user2.Role, resp.Role)
			},
		},
		{
			name:           "success - get self",
			token:          token1,
			userID:         userID1.String(),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp userhandler.UserResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, userID1, resp.ID)
				assert.Equal(t, user1.Email, resp.Email)
			},
		},
		{
			name:           "not found - non-existent user",
			token:          token1,
			userID:         uuid.NewString(),
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "user not found", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "bad request - invalid user id",
			token:          token1,
			userID:         "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "invalid user id", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "unauthorized - missing token",
			token:          "",
			userID:         userID1.String(),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupAuthenticatedGin(t, jwtConfig)
			router.GET("/users/:id", userHandler.GetUser())

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%s", tt.userID), nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}

// ============================================
// UPDATE CURRENT USER TESTS
// ============================================

func TestUserHandler_UpdateCurrentUser_Integration(t *testing.T) {
	userHandler, authHandler, jwtConfig, _ := setupUserTest(t)

	user1 := authandler.RegisterRequest{
		Email:    "currentuser1@test.com",
		Password: "MySecureP@ss123",
		Role:     "SELLER",
	}

	user2 := authandler.RegisterRequest{
		Email:    "currentuser2@test.com",
		Password: "MySecureP@ss123",
		Role:     "BUYER",
	}

	// Register test users
	userID1, token1, _ := registerTestUser(t, authHandler, user1)
	_, token2, _ := registerTestUser(t, authHandler, user2)

	tests := []struct {
		name           string
		token          string
		body           map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "success - update email",
			token:          token1,
			body:           map[string]interface{}{"email": "updated1@test.com"},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp userhandler.UserResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, userID1, resp.ID)
				assert.Equal(t, "updated1@test.com", resp.Email)
				assert.Equal(t, user1.Role, resp.Role)
			},
		},
		{
			name:           "success - update to same email",
			token:          token2,
			body:           map[string]interface{}{"email": "updateuser2@test.com"},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp userhandler.UserResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "updateuser2@test.com", resp.Email)
			},
		},
		{
			name:           "conflict - duplicate email",
			token:          token1,
			body:           map[string]interface{}{"email": "updateuser2@test.com"},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "email already exists", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "validation error - invalid email",
			token:          token1,
			body:           map[string]interface{}{"email": "not-an-email"},
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
			name:           "validation error - missing email",
			token:          token1,
			body:           map[string]interface{}{},
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
			name:           "validation error - empty body",
			token:          token1,
			body:           nil,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "request body is empty", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "unauthorized - missing token",
			token:          "",
			body:           map[string]interface{}{"email": "unauth@test.com"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupAuthenticatedGin(t, jwtConfig)
			router.PATCH("/users/me", userHandler.UpdateCurrentUser())

			var reqBody []byte
			if tt.body != nil {
				var err error
				reqBody, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(reqBody))
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}

// ============================================
// LIST USERS BY COMPANY TESTS
// ============================================

func TestUserHandler_ListUsersByCompany_Integration(t *testing.T) {
	userHandler, authHandler, jwtConfig, repo := setupUserTest(t)

	user1 := authandler.RegisterRequest{
		Email:    "currentuser1@test.com",
		Password: "MySecureP@ss123",
		Role:     "SELLER",
	}

	user2 := authandler.RegisterRequest{
		Email:    "currentuser2@test.com",
		Password: "MySecureP@ss123",
		Role:     "BUYER",
	}

	// Register users and bind to company
	userID1, token1, _ := registerTestUser(t, authHandler, user1)
	userID2, _, _ := registerTestUser(t, authHandler, user2)

	// Create company via DB directly
	companyID := uuid.New()
	ctx := t.Context()
	_, err := repo.Pool().Exec(ctx, "INSERT INTO companies (id, name, description) VALUES ($1, $2, $3)", companyID, "Test Company", nil)
	require.NoError(t, err)

	// Bind users to company
	_, err = repo.Pool().Exec(ctx, "UPDATE users SET company_id = $1 WHERE id IN ($2, $3)", companyID, userID1, userID2)
	require.NoError(t, err)

	tests := []struct {
		name           string
		token          string
		companyID      string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "success - list company users",
			token:          token1,
			companyID:      companyID.String(),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp []userhandler.UserResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Len(t, resp, 2)
				emails := make([]string, len(resp))
				for i, u := range resp {
					emails[i] = u.Email
				}
				assert.Contains(t, emails, user1.Email)
				assert.Contains(t, emails, user2.Email)
			},
		},
		{
			name:           "company not found",
			token:          token1,
			companyID:      uuid.NewString(),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "bad request - invalid company id",
			token:          token1,
			companyID:      "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "invalid company id", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "unauthorized - missing token",
			token:          "",
			companyID:      companyID.String(),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupAuthenticatedGin(t, jwtConfig)
			router.GET("/companies/:id/users", userHandler.ListUsersByCompany())

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/companies/%s/users", tt.companyID), nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.Bytes())
			}
		})
	}
}
