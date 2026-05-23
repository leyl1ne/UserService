package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leyl1ne/UserService/internal/infrastructure/auth/jwt"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/password"
	authmodel "github.com/leyl1ne/UserService/internal/model/auth"
	usermodel "github.com/leyl1ne/UserService/internal/model/user"
	"github.com/leyl1ne/UserService/internal/service"
)

type Service struct {
	repo                Repository
	passwordHasher      PasswordHasher
	accessTokenProvider AccessTokenProvider
	refreshTokenTTL     time.Duration
}

func NewService(repo Repository, passwordHasher PasswordHasher, accessTokenProvider AccessTokenProvider, refreshTokenTTL time.Duration) *Service {
	return &Service{
		repo:                repo,
		passwordHasher:      passwordHasher,
		accessTokenProvider: accessTokenProvider,
		refreshTokenTTL:     refreshTokenTTL,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*TokensOutput, error) {
	const op = "service.auth.Register"

	passwordHash, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("%s: hash password: %w", op, err)
	}

	user, err := usermodel.NewUser(input.Email, passwordHash, input.Role)
	if err != nil {
		if errors.Is(err, usermodel.ErrInvalidUserRole) {
			return nil, service.NewValidationError("role", usermodel.ErrInvalidUserRole.Error())
		}
		return nil, fmt.Errorf("%s: failed create new user: %w", op, err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	refreshToken := authmodel.NewRefreshToken(user.ID, s.refreshTokenTTL)

	err = s.repo.WithTransaction(ctxTimeout, func(txCtx context.Context) error {
		err = s.repo.CreateUser(txCtx, user)
		if err != nil {
			if errors.Is(err, usermodel.ErrEmailAlreadyExists) {
				return service.ErrDuplicateEmail
			}
			return fmt.Errorf("create user: %w", err)
		}

		err = s.repo.SaveRefreshToken(txCtx, refreshToken)
		if err != nil {
			return fmt.Errorf("save refresh token: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: transaction: %w", op, err)
	}

	accessToken, err := s.accessTokenProvider.GenerateAccessToken(jwt.Payload{
		UserID:    user.ID.String(),
		UserRole:  string(user.Role),
		CompanyID: "",
	})
	if err != nil {
		return nil, fmt.Errorf("%s: generate access token: %w", op, err)
	}

	return &TokensOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token.String(),
	}, nil

}

func (s *Service) Login(ctx context.Context, input LoginInput) (*TokensOutput, error) {
	const op = "service.auth.Login"

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user, err := s.repo.GetUserByEmail(ctxTimeout, input.Email)
	if err != nil {
		if errors.Is(err, usermodel.ErrEmailNotFound) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("%s: get user by email: %w", op, err)
	}

	// TODO: узнать по поводу безопасности данного способа обработки ошибки
	if err = s.passwordHasher.Compare(user.PasswordHash, input.Password); err != nil {
		if errors.Is(err, password.ErrInvalidPassword) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("%s: compare password: %w", op, err)
	}

	refreshToken := authmodel.NewRefreshToken(user.ID, s.refreshTokenTTL)

	if err = s.repo.SaveRefreshToken(ctxTimeout, refreshToken); err != nil {
		return nil, fmt.Errorf("%s: save refresh token: %w", op, err)
	}

	companyID := ""
	if user.CompanyID != nil {
		companyID = user.CompanyID.String()
	}

	accessToken, err := s.accessTokenProvider.GenerateAccessToken(jwt.Payload{
		UserID:    user.ID.String(),
		UserRole:  string(user.Role),
		CompanyID: companyID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: generate access token: %w", op, err)
	}

	return &TokensOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token.String(),
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*RefreshOutput, error) {
	const op = "service.auth.Refresh"

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	token, err := s.repo.GetRefreshToken(ctxTimeout, refreshToken)
	if err != nil {
		if errors.Is(err, authmodel.ErrRefreshTokenNotFound) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrInvalidToken)
		}
		return nil, fmt.Errorf("%s: get refresh token: %w", op, err)
	}

	if time.Now().After(token.ExpiresAt) {
		_ = s.repo.DeleteRefreshToken(ctxTimeout, refreshToken)
		return nil, fmt.Errorf("%s: %w", op, service.ErrTokenExpired)
	}

	user, err := s.repo.GetUserByID(ctxTimeout, token.UserID)
	if err != nil {
		if errors.Is(err, usermodel.ErrUserNotFound) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrInvalidToken)
		}
		return nil, fmt.Errorf("%s: get user by id: %w", op, err)
	}

	companyID := ""
	if user.CompanyID != nil {
		companyID = user.CompanyID.String()
	}

	accessToken, err := s.accessTokenProvider.GenerateAccessToken(jwt.Payload{
		UserID:    token.UserID.String(),
		UserRole:  string(user.Role),
		CompanyID: companyID,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: generate access token: %w", op, err)
	}

	return &RefreshOutput{
		AccessToken: accessToken,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	const op = "service.auth.Logout"

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.repo.DeleteRefreshToken(ctxTimeout, refreshToken); err != nil {
		if errors.Is(err, authmodel.ErrRefreshTokenNotFound) {
			return nil
		}
		return fmt.Errorf("%s: delete refresh token: %w", op, err)
	}

	return nil
}
