package company_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	postgresInfra "github.com/leyl1ne/UserService/internal/infrastructure/postgres"
	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/logger/zl"
	"github.com/leyl1ne/UserService/internal/repository/postgres"
	authservice "github.com/leyl1ne/UserService/internal/service/auth"
	companyservice "github.com/leyl1ne/UserService/internal/service/company"
	"github.com/leyl1ne/UserService/internal/testutils"
	authandler "github.com/leyl1ne/UserService/internal/transport/http/handler/auth"
	companyhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/company"
	"github.com/leyl1ne/UserService/internal/transport/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCompanyTest(t *testing.T) (*companyhandler.CompanyHandler, *authandler.AuthHandler, jwt.Config, *postgresInfra.Postgres) {
	db := testutils.SetupPostgres(t)
	repo := postgres.NewRepository(db.Pool())

	hasher, err := password.NewHasher(password.Config{Cost: 10})
	require.NoError(t, err)

	jwtConfig := jwt.Config{
		Secret:        "test-secret-key-for-jwt-signing-32b",
		AccessTokeTTL: 15 * time.Minute,
	}
	jwtProvider := jwt.NewJWTGenerator(jwtConfig)

	log := setupLogger()

	authSvc := authservice.NewService(repo, hasher, jwtProvider, 24*time.Hour)
	companySvc := companyservice.NewService(repo)

	authHandler := authandler.NewAuthHandler(log, authSvc)
	companyHandler := companyhandler.NewCompanyHandler(log, companySvc)

	return companyHandler, authHandler, jwtConfig, db
}

func setupLogger() logger.Logger {
	return zl.NewZerologLogger("debug", io.Discard)
}

func setupAuthenticatedGin(jwtConfig jwt.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtProvider := jwt.NewJWTGenerator(jwtConfig)
	router.Use(middleware.AuthMiddleware(setupLogger(), jwtProvider))

	return router
}

func parseErrorResponse(t *testing.T, body []byte) map[string]interface{} {
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
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

func registerTestUser(t *testing.T, handler *authandler.AuthHandler, email, password, role string) (userID uuid.UUID, jwtToken string) {
	r := gin.New()
	r.POST("/auth/register", handler.Register())

	body, _ := json.Marshal(map[string]interface{}{
		"email":    email,
		"password": password,
		"role":     role,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var resp authandler.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	jwtProvider := jwt.NewJWTGenerator(jwt.Config{
		Secret:        "test-secret-key-for-jwt-signing-32b",
		AccessTokeTTL: 15 * time.Minute,
	})
	payload, err := jwtProvider.Validate(resp.AccessToken)
	require.NoError(t, err)

	userID, err = uuid.Parse(payload.UserID)
	require.NoError(t, err)

	return userID, resp.AccessToken
}

// ============================================
// CREATE COMPANY TESTS
// ============================================

func TestCompanyHandler_CreateCompany_Integration(t *testing.T) {
	companyHandler, authHandler, jwtConfig, _ := setupCompanyTest(t)

	_, token := registerTestUser(t, authHandler, "companycreate@test.com", "MySecureP@ss123", "SELLER")

	tests := []struct {
		name           string
		token          string
		body           map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:  "success - with description",
			token: token,
			body: map[string]interface{}{
				"name":        "Acme Corporation",
				"description": "A leading provider of widgets",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp companyhandler.CompanyResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEqual(t, uuid.Nil, resp.ID)
				assert.Equal(t, "Acme Corporation", resp.Name)
				assert.NotNil(t, resp.Description)
				assert.Equal(t, "A leading provider of widgets", *resp.Description)
				assert.NotEmpty(t, resp.CreatedAt)
			},
		},
		{
			name:  "success - without description",
			token: token,
			body: map[string]interface{}{
				"name": "Minimal Corp",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp companyhandler.CompanyResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Minimal Corp", resp.Name)
				assert.Nil(t, resp.Description)
			},
		},
		{
			name:  "validation error - empty name",
			token: token,
			body: map[string]interface{}{
				"name":        "",
				"description": "Some description",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "validation failed", resp["error"])
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "field is required", fields["name"])
			},
		},
		{
			name:  "validation error - name too long",
			token: token,
			body: map[string]interface{}{
				"name": strings.Repeat("a", 256),
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "too long", fields["name"])
			},
		},
		{
			name:  "validation error - description too long",
			token: token,
			body: map[string]interface{}{
				"name":        "Valid Name",
				"description": strings.Repeat("b", 1001),
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "too long", fields["description"])
			},
		},
		{
			name:  "validation error - missing name",
			token: token,
			body: map[string]interface{}{
				"description": "Only description",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &resp))
				fields := resp["fields"].(map[string]interface{})
				assert.Equal(t, "field is required", fields["name"])
			},
		},
		{
			name:           "validation error - empty body",
			token:          token,
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
			body:           map[string]interface{}{"name": "Unauthorized Corp"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "unauthorized - invalid token",
			token:          "invalid-token",
			body:           map[string]interface{}{"name": "Bad Token Corp"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupAuthenticatedGin(jwtConfig)
			router.POST("/companies", companyHandler.CreateCompany())

			var reqBody []byte
			if tt.body != nil {
				var err error
				reqBody, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewReader(reqBody))
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
// GET COMPANY TESTS
// ============================================

func TestCompanyHandler_GetCompany_Integration(t *testing.T) {
	companyHandler, authHandler, jwtConfig, repo := setupCompanyTest(t)

	_, token := registerTestUser(t, authHandler, "companyget@test.com", "MySecureP@ss123", "SELLER")

	testCompanyName := "Test Get Company"
	testCompanyDesc := "A test company for get endpoint"
	// Insert a company directly into DB
	companyID := uuid.New()
	ctx := t.Context()
	_, err := repo.Pool().Exec(ctx,
		"INSERT INTO companies (id, name, description) VALUES ($1, $2, $3)",
		companyID, testCompanyName, testCompanyDesc)
	require.NoError(t, err)

	noDescCompanyName := "No Description Corp"
	// Insert another company without description
	companyID2 := uuid.New()
	_, err = repo.Pool().Exec(ctx,
		"INSERT INTO companies (id, name) VALUES ($1, $2)",
		companyID2, noDescCompanyName)
	require.NoError(t, err)

	tests := []struct {
		name           string
		token          string
		companyID      string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "success - company with description",
			token:          token,
			companyID:      companyID.String(),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp companyhandler.CompanyResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, companyID, resp.ID)
				assert.Equal(t, testCompanyName, resp.Name)
				assert.NotNil(t, resp.Description)
				assert.Equal(t, testCompanyDesc, *resp.Description)
				assert.NotEmpty(t, resp.CreatedAt)
			},
		},
		{
			name:           "success - company without description",
			token:          token,
			companyID:      companyID2.String(),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp companyhandler.CompanyResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, companyID2, resp.ID)
				assert.Equal(t, noDescCompanyName, resp.Name)
				assert.Nil(t, resp.Description)
			},
		},
		{
			name:           "not found - non-existent company",
			token:          token,
			companyID:      uuid.NewString(),
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "company not found", errResp["error"].(map[string]interface{})["message"])
			},
		},
		{
			name:           "bad request - invalid company id",
			token:          token,
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
		{
			name:           "unauthorized - expired token",
			token:          generateJWTToken(t, jwt.Config{Secret: jwtConfig.Secret, AccessTokeTTL: -time.Second}, uuid.NewString(), "SELLER", ""),
			companyID:      companyID.String(),
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				errResp := parseErrorResponse(t, body)
				assert.Equal(t, "token expired", errResp["error"].(map[string]interface{})["message"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupAuthenticatedGin(jwtConfig)
			router.GET("/companies/:id", companyHandler.GetCompany())

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/companies/%s", tt.companyID), nil)
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
// END-TO-END COMPANY FLOW TEST
// ============================================

func TestCompanyHandler_FullFlow_Integration(t *testing.T) {
	companyHandler, authHandler, jwtConfig, _ := setupCompanyTest(t)

	_, token := registerTestUser(t, authHandler, "companyflow@test.com", "MySecureP@ss123", "SELLER")

	router := setupAuthenticatedGin(jwtConfig)
	router.POST("/companies", companyHandler.CreateCompany())
	router.GET("/companies/:id", companyHandler.GetCompany())

	var createdCompanyID uuid.UUID

	// Step 1: Create company
	t.Run("step 1: create company", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"name":        "Flow Test Corp",
			"description": "Testing the full company flow",
		})
		req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp companyhandler.CompanyResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		createdCompanyID = resp.ID
		assert.Equal(t, "Flow Test Corp", resp.Name)
		assert.NotNil(t, resp.Description)
	})

	// Step 2: Get created company
	t.Run("step 2: get created company", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/companies/%s", createdCompanyID.String()), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp companyhandler.CompanyResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, createdCompanyID, resp.ID)
		assert.Equal(t, "Flow Test Corp", resp.Name)
	})

	// Step 3: Get non-existent company
	t.Run("step 3: get non-existent company returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/companies/%s", uuid.NewString()), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
