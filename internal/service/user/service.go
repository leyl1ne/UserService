package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	companymodel "github.com/leyl1ne/UserService/internal/model/company"
	usermodel "github.com/leyl1ne/UserService/internal/model/user"
	"github.com/leyl1ne/UserService/internal/service"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*UserOutput, error) {
	const op = "service.user.GetUser"

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	user, err := s.repo.GetUserByID(ctxTimeout, userID)
	if err != nil {
		if errors.Is(err, usermodel.ErrUserNotFound) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrUserNotFound)
		}

		return nil, fmt.Errorf("%s: get user by id :%w", op, err)
	}

	return toUserOutput(user), nil
}

func (s *Service) UpdateUser(ctx context.Context, input UpdateUserInput) (*UserOutput, error) {
	const op = "service.user.UpdateUser"

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.repo.UpdateUser(ctxTimeout, input.UserID, input.Email); err != nil {
		if errors.Is(err, usermodel.ErrEmailAlreadyExists) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrDuplicateEmail)
		}

		if errors.Is(err, usermodel.ErrUserNotFound) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: update user: %w", op, err)
	}

	user, err := s.repo.GetUserByID(ctxTimeout, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("%s: get user by id: %w", op, err)
	}

	return toUserOutput(user), nil
}

func (s *Service) ListUsersByCompany(ctx context.Context, companyID uuid.UUID) ([]UserOutput, error) {
	const op = "service.user.ListUsersByCompany"

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.repo.GetCompanyByID(ctxTimeout, companyID)
	if err != nil {
		if errors.Is(err, companymodel.ErrCompanyNotFound) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrCompanyNotFound)
		}
		return nil, fmt.Errorf("%s: check company exists: %w", op, err)
	}

	users, err := s.repo.ListUsersByCompanyID(ctxTimeout, companyID)
	if err != nil {
		return nil, fmt.Errorf("%s: list users by company id:%w", op, err)
	}

	return toListUserOutput(users), nil
}
