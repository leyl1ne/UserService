package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/jwt"
	authmodel "github.com/leyl1ne/UserService/internal/model/auth"
	usermodel "github.com/leyl1ne/UserService/internal/model/user"
)

//go:generate go run github.com/vektra/mockery/v2@latest --name=Repository
type Repository interface {
	WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
	CreateUser(ctx context.Context, user *usermodel.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*usermodel.User, error)
	GetUserByEmail(ctx context.Context, email string) (*usermodel.User, error)
	SaveRefreshToken(ctx context.Context, token *authmodel.RefreshToken) error
	GetRefreshToken(ctx context.Context, token string) (*authmodel.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, token string) error
}

//go:generate go run github.com/vektra/mockery/v2@latest --name=PasswordHasher
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

//go:generate go run github.com/vektra/mockery/v2@latest --name=AccessTokenProvider
type AccessTokenProvider interface {
	GenerateAccessToken(payload jwt.Payload) (string, error)
}

type RegisterInput struct {
	Email    string
	Password string
	Role     string
}

type LoginInput struct {
	Email    string
	Password string
}

type TokensOutput struct {
	AccessToken  string
	RefreshToken string
}

type RefreshOutput struct {
	AccessToken string
}
