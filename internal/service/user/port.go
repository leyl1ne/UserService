package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	companymodel "github.com/leyl1ne/UserService/internal/model/company"
	usermodel "github.com/leyl1ne/UserService/internal/model/user"
)

//go:generate go run github.com/vektra/mockery/v2@latest --name=Repository
type Repository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*usermodel.User, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, email string) error
	GetCompanyByID(ctx context.Context, id uuid.UUID) (*companymodel.Company, error)
	ListUsersByCompanyID(ctx context.Context, companyID uuid.UUID) ([]usermodel.User, error)
}

type UpdateUserInput struct {
	UserID uuid.UUID
	Email  string
}

type UserOutput struct {
	ID        uuid.UUID
	Email     string
	Role      string
	CompanyID *uuid.UUID
	CreatedAt time.Time
}

func toUserOutput(u *usermodel.User) *UserOutput {
	return &UserOutput{
		ID:        u.ID,
		Email:     u.Email,
		Role:      string(u.Role),
		CompanyID: u.CompanyID,
		CreatedAt: u.CreatedAt,
	}
}

func toListUserOutput(listUser []usermodel.User) []UserOutput {
	result := make([]UserOutput, 0, len(listUser))
	for i := range listUser {
		result = append(result, UserOutput{
			ID:        listUser[i].ID,
			Email:     listUser[i].Email,
			Role:      string(listUser[i].Role),
			CompanyID: listUser[i].CompanyID,
			CreatedAt: listUser[i].CreatedAt,
		})
	}

	return result
}
