package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/password"
	authmodel "github.com/leyl1ne/UserService/internal/model/auth"
	usermodel "github.com/leyl1ne/UserService/internal/model/user"
	"github.com/leyl1ne/UserService/internal/service"
	"github.com/leyl1ne/UserService/internal/service/auth/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testRefreshTokenTTL = 24 * time.Hour

func Test_Service_Register(t *testing.T) {
	ctx := context.Background()

	type registerInput struct {
		email    string
		password string
		role     string
	}

	type mockSetup struct {
		hashErr       error
		txReturn      error
		runTx         bool // if true, WithTransaction executes the callback
		createUserErr error
		saveTokenErr  error
		accessToken   string
		accessErr     error
	}

	cases := []struct {
		name      string
		input     registerInput
		mockSetup mockSetup
		expectErr error
	}{
		{
			name: "success",
			input: registerInput{
				email:    "test@mail.com",
				password: "secret123",
				role:     "SELLER",
			},
			mockSetup: mockSetup{
				runTx:       true,
				accessToken: "access-token-123",
			},
			expectErr: nil,
		},
		{
			name: "hash password fails",
			input: registerInput{
				email:    "test@mail.com",
				password: "secret123",
				role:     "SELLER",
			},
			mockSetup: mockSetup{
				hashErr: errors.New("bcrypt failed"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
		{
			name: "invalid role — validation error",
			input: registerInput{
				email:    "test@mail.com",
				password: "secret123",
				role:     "ADMIN",
			},
			mockSetup: mockSetup{
				// Hash succeeds, but NewUser("ADMIN") fails
			},
			expectErr: service.NewValidationError("role", usermodel.ErrInvalidUserRole.Error()),
		},
		{
			name: "duplicate email in transaction",
			input: registerInput{
				email:    "dup@mail.com",
				password: "secret123",
				role:     "SELLER",
			},
			mockSetup: mockSetup{
				txReturn: service.ErrDuplicateEmail,
			},
			expectErr: service.ErrDuplicateEmail,
		},
		{
			name: "transaction unexpected error",
			input: registerInput{
				email:    "test@mail.com",
				password: "secret123",
				role:     "SELLER",
			},
			mockSetup: mockSetup{
				txReturn: errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
		{
			name: "generate access token fails",
			input: registerInput{
				email:    "test@mail.com",
				password: "secret123",
				role:     "SELLER",
			},
			mockSetup: mockSetup{
				runTx:     true,
				accessErr: errors.New("jwt signing failed"),
			},
			expectErr: nil, // wrapped, not a sentinel
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)
			mockHasher := mocks.NewPasswordHasher(t)
			mockTokenProvider := mocks.NewAccessTokenProvider(t)

			// Hash password — always called (unless we test pre-hash errors, but there are none)
			if tc.mockSetup.hashErr != nil {
				mockHasher.On("Hash", tc.input.password).Return("", tc.mockSetup.hashErr).Once()
			} else {
				mockHasher.On("Hash", tc.input.password).Return("hashed-pw", nil).Once()
			}

			// WithTransaction
			if tc.mockSetup.hashErr == nil && !errors.Is(tc.expectErr, service.ErrValidation) {
				if tc.mockSetup.runTx {
					mockRepo.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
						Run(func(args mock.Arguments) {
							fn := args.Get(1).(func(context.Context) error)
							_ = fn(context.Background())
						}).
						Return(nil).
						Once()

					// Inner mocks (called inside the transaction callback)
					mockRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*user.User")).
						Return(tc.mockSetup.createUserErr).
						Once()
					mockRepo.On("SaveRefreshToken", mock.Anything, mock.AnythingOfType("*auth.RefreshToken")).
						Return(tc.mockSetup.saveTokenErr).
						Once()

					// GenerateAccessToken
					mockTokenProvider.On("GenerateAccessToken", mock.AnythingOfType("jwt.Payload")).
						Return(tc.mockSetup.accessToken, tc.mockSetup.accessErr).
						Once()
				} else {
					// Error from transaction — just return the error directly
					mockRepo.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
						Return(tc.mockSetup.txReturn).
						Once()
				}
			}

			svc := NewService(mockRepo, mockHasher, mockTokenProvider, testRefreshTokenTTL)

			result, err := svc.Register(ctx, RegisterInput{
				Email:    tc.input.email,
				Password: tc.input.password,
				Role:     tc.input.role,
			})

			if tc.expectErr != nil {
				var valErr service.ValidationError
				if errors.As(tc.expectErr, &valErr) {
					require.Equal(t, map[string]string{
						"role": usermodel.ErrInvalidUserRole.Error(),
					}, valErr.Fields)
				} else {
					require.ErrorIs(t, err, tc.expectErr)
				}
				require.Nil(t, result)
			} else if tc.mockSetup.hashErr != nil || tc.mockSetup.txReturn != nil || tc.mockSetup.accessErr != nil {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.NotEmpty(t, result.AccessToken)
				require.NotEmpty(t, result.RefreshToken)
			}
		})
	}
}

func Test_Service_Login(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	companyID := uuid.New()
	now := time.Now()

	type loginInput struct {
		email    string
		password string
	}

	type mockSetup struct {
		getUser      *usermodel.User
		getUserErr   error
		compareErr   error
		saveTokenErr error
		accessToken  string
		accessErr    error
	}

	cases := []struct {
		name      string
		input     loginInput
		mockSetup mockSetup
		expectErr error
	}{
		{
			name: "success",
			input: loginInput{
				email:    "seller@mail.com",
				password: "secret123",
			},
			mockSetup: mockSetup{
				getUser: &usermodel.User{
					ID:           userID,
					Email:        "seller@mail.com",
					PasswordHash: "hashed-pw",
					Role:         usermodel.Seller,
					CompanyID:    nil,
					CreatedAt:    now,
				},
				accessToken: "access-token-123",
			},
			expectErr: nil,
		},
		{
			name: "success with company id",
			input: loginInput{
				email:    "seller@mail.com",
				password: "secret123",
			},
			mockSetup: mockSetup{
				getUser: &usermodel.User{
					ID:           userID,
					Email:        "seller@mail.com",
					PasswordHash: "hashed-pw",
					Role:         usermodel.Seller,
					CompanyID:    &companyID,
					CreatedAt:    now,
				},
				accessToken: "access-token-456",
			},
			expectErr: nil,
		},
		{
			name: "email not found",
			input: loginInput{
				email:    "unknown@mail.com",
				password: "secret123",
			},
			mockSetup: mockSetup{
				getUserErr: usermodel.ErrEmailNotFound,
			},
			expectErr: service.ErrInvalidCredentials,
		},
		{
			name: "get user by email unexpected error",
			input: loginInput{
				email:    "test@mail.com",
				password: "secret123",
			},
			mockSetup: mockSetup{
				getUserErr: errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped
		},
		{
			name: "password mismatch",
			input: loginInput{
				email:    "seller@mail.com",
				password: "wrong-password",
			},
			mockSetup: mockSetup{
				getUser: &usermodel.User{
					ID:           userID,
					Email:        "seller@mail.com",
					PasswordHash: "hashed-pw",
					Role:         usermodel.Seller,
					CompanyID:    nil,
					CreatedAt:    now,
				},
				compareErr: password.ErrInvalidPassword,
			},
			expectErr: service.ErrInvalidCredentials,
		},
		{
			name: "compare password unexpected error",
			input: loginInput{
				email:    "seller@mail.com",
				password: "secret123",
			},
			mockSetup: mockSetup{
				getUser: &usermodel.User{
					ID:           userID,
					Email:        "seller@mail.com",
					PasswordHash: "hashed-pw",
					Role:         usermodel.Seller,
					CompanyID:    nil,
					CreatedAt:    now,
				},
				compareErr: errors.New("bcrypt internal error"),
			},
			expectErr: nil, // wrapped
		},
		{
			name: "save refresh token fails",
			input: loginInput{
				email:    "seller@mail.com",
				password: "secret123",
			},
			mockSetup: mockSetup{
				getUser: &usermodel.User{
					ID:           userID,
					Email:        "seller@mail.com",
					PasswordHash: "hashed-pw",
					Role:         usermodel.Seller,
					CompanyID:    nil,
					CreatedAt:    now,
				},
				saveTokenErr: errors.New("db write failed"),
			},
			expectErr: nil, // wrapped
		},
		{
			name: "generate access token fails",
			input: loginInput{
				email:    "seller@mail.com",
				password: "secret123",
			},
			mockSetup: mockSetup{
				getUser: &usermodel.User{
					ID:           userID,
					Email:        "seller@mail.com",
					PasswordHash: "hashed-pw",
					Role:         usermodel.Seller,
					CompanyID:    nil,
					CreatedAt:    now,
				},
				accessToken: "",
				accessErr:   errors.New("jwt signing failed"),
			},
			expectErr: nil, // wrapped
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)
			mockHasher := mocks.NewPasswordHasher(t)
			mockTokenProvider := mocks.NewAccessTokenProvider(t)

			mockRepo.On("GetUserByEmail", mock.Anything, tc.input.email).
				Return(tc.mockSetup.getUser, tc.mockSetup.getUserErr).
				Once()

			if tc.mockSetup.getUserErr == nil {
				mockHasher.On("Compare", tc.mockSetup.getUser.PasswordHash, tc.input.password).
					Return(tc.mockSetup.compareErr).
					Once()
			}

			if tc.mockSetup.getUserErr == nil && tc.mockSetup.compareErr == nil {
				mockRepo.On("SaveRefreshToken", mock.Anything, mock.AnythingOfType("*auth.RefreshToken")).
					Return(tc.mockSetup.saveTokenErr).
					Once()
			}

			if tc.mockSetup.getUserErr == nil && tc.mockSetup.compareErr == nil && tc.mockSetup.saveTokenErr == nil {
				mockTokenProvider.On("GenerateAccessToken", mock.AnythingOfType("jwt.Payload")).
					Return(tc.mockSetup.accessToken, tc.mockSetup.accessErr).
					Once()
			}

			svc := NewService(mockRepo, mockHasher, mockTokenProvider, testRefreshTokenTTL)

			result, err := svc.Login(ctx, LoginInput{
				Email:    tc.input.email,
				Password: tc.input.password,
			})

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, result)
			} else if tc.mockSetup.getUserErr != nil || tc.mockSetup.compareErr != nil ||
				tc.mockSetup.saveTokenErr != nil || tc.mockSetup.accessErr != nil {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.NotEmpty(t, result.AccessToken)
				require.NotEmpty(t, result.RefreshToken)
			}
		})
	}
}

func Test_Service_Refresh(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	companyID := uuid.New()
	now := time.Now()

	// Create a valid (non-expired) refresh token model
	validRefreshToken := authmodel.NewRefreshToken(userID, testRefreshTokenTTL)

	// Create an expired refresh token model (negative TTL → ExpiresAt in the past)
	expiredRefreshToken := authmodel.NewRefreshToken(userID, -time.Hour)

	type mockSetup struct {
		getToken    *authmodel.RefreshToken
		getTokenErr error
		deleteErr   error
		getUser     *usermodel.User
		getUserErr  error
		accessToken string
		accessErr   error
	}

	cases := []struct {
		name      string
		mockSetup mockSetup
		expectErr error
	}{
		{
			name: "success",
			mockSetup: mockSetup{
				getToken: validRefreshToken,
				getUser: &usermodel.User{
					ID:        userID,
					Email:     "seller@mail.com",
					Role:      usermodel.Seller,
					CompanyID: nil,
					CreatedAt: now,
				},
				accessToken: "new-access-token",
			},
			expectErr: nil,
		},
		{
			name: "success with company id",
			mockSetup: mockSetup{
				getToken: validRefreshToken,
				getUser: &usermodel.User{
					ID:        userID,
					Email:     "seller@mail.com",
					Role:      usermodel.Seller,
					CompanyID: &companyID,
					CreatedAt: now,
				},
				accessToken: "new-access-token",
			},
			expectErr: nil,
		},
		{
			name: "refresh token not found",
			mockSetup: mockSetup{
				getTokenErr: authmodel.ErrRefreshTokenNotFound,
			},
			expectErr: service.ErrInvalidToken,
		},
		{
			name: "get refresh token unexpected error",
			mockSetup: mockSetup{
				getTokenErr: errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped
		},
		{
			name: "token expired",
			mockSetup: mockSetup{
				getToken:  expiredRefreshToken,
				deleteErr: nil,
			},
			expectErr: service.ErrTokenExpired,
		},
		{
			name: "user not found after valid refresh token",
			mockSetup: mockSetup{
				getToken:   validRefreshToken,
				getUserErr: usermodel.ErrUserNotFound,
			},
			expectErr: service.ErrInvalidToken,
		},
		{
			name: "get user by id unexpected error",
			mockSetup: mockSetup{
				getToken:   validRefreshToken,
				getUserErr: errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped
		},
		{
			name: "generate access token fails",
			mockSetup: mockSetup{
				getToken: validRefreshToken,
				getUser: &usermodel.User{
					ID:        userID,
					Email:     "seller@mail.com",
					Role:      usermodel.Seller,
					CompanyID: nil,
					CreatedAt: now,
				},
				accessErr: errors.New("jwt signing failed"),
			},
			expectErr: nil, // wrapped
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)
			mockHasher := mocks.NewPasswordHasher(t)
			mockTokenProvider := mocks.NewAccessTokenProvider(t)

			// GetRefreshToken
			mockRepo.On("GetRefreshToken", mock.Anything, validRefreshToken.Token.String()).
				Return(tc.mockSetup.getToken, tc.mockSetup.getTokenErr).
				Once()

			// If token is expired, DeleteRefreshToken is called
			if tc.mockSetup.getToken != nil &&
				tc.mockSetup.getTokenErr == nil &&
				now.After(tc.mockSetup.getToken.ExpiresAt) {
				mockRepo.On("DeleteRefreshToken", mock.Anything, validRefreshToken.Token.String()).
					Return(tc.mockSetup.deleteErr).
					Once()
			}

			// GetUserByID — called when token is valid and not expired
			if tc.mockSetup.getToken != nil &&
				tc.mockSetup.getTokenErr == nil &&
				!now.After(tc.mockSetup.getToken.ExpiresAt) {
				mockRepo.On("GetUserByID", mock.Anything, tc.mockSetup.getToken.UserID).
					Return(tc.mockSetup.getUser, tc.mockSetup.getUserErr).
					Once()
			}

			// GenerateAccessToken — called when user is found
			if tc.mockSetup.getToken != nil &&
				tc.mockSetup.getTokenErr == nil &&
				!now.After(tc.mockSetup.getToken.ExpiresAt) &&
				tc.mockSetup.getUser != nil && tc.mockSetup.getUserErr == nil {
				mockTokenProvider.On("GenerateAccessToken", mock.AnythingOfType("jwt.Payload")).
					Return(tc.mockSetup.accessToken, tc.mockSetup.accessErr).
					Once()
			}

			svc := NewService(mockRepo, mockHasher, mockTokenProvider, testRefreshTokenTTL)

			result, err := svc.Refresh(ctx, validRefreshToken.Token.String())

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, result)
			} else if tc.mockSetup.getTokenErr != nil || tc.mockSetup.getUserErr != nil || tc.mockSetup.accessErr != nil {
				require.Error(t, err)
				require.Nil(t, result)
			} else if tc.mockSetup.getToken != nil && now.After(tc.mockSetup.getToken.ExpiresAt) {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.NotEmpty(t, result.AccessToken)
			}
		})
	}
}

func Test_Service_Logout(t *testing.T) {
	ctx := context.Background()
	refreshToken := "some-refresh-token"

	type mockSetup struct {
		deleteErr error
	}

	cases := []struct {
		name      string
		mockSetup mockSetup
		expectErr error // nil means no error expected
	}{
		{
			name: "success",
			mockSetup: mockSetup{
				deleteErr: nil,
			},
			expectErr: nil,
		},
		{
			name: "refresh token not found — returns nil",
			mockSetup: mockSetup{
				deleteErr: authmodel.ErrRefreshTokenNotFound,
			},
			expectErr: nil, // Logout returns nil when token not found
		},
		{
			name: "delete refresh token unexpected error",
			mockSetup: mockSetup{
				deleteErr: errors.New("db connection lost"),
			},
			expectErr: nil, // wrapped, not a sentinel — check error is non-nil
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := mocks.NewRepository(t)
			mockHasher := mocks.NewPasswordHasher(t)
			mockTokenProvider := mocks.NewAccessTokenProvider(t)

			mockRepo.On("DeleteRefreshToken", mock.Anything, refreshToken).
				Return(tc.mockSetup.deleteErr).
				Once()

			svc := NewService(mockRepo, mockHasher, mockTokenProvider, testRefreshTokenTTL)

			err := svc.Logout(ctx, refreshToken)

			if tc.name == "delete refresh token unexpected error" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
