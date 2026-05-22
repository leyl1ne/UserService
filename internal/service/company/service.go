package company

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	companymodel "github.com/leyl1ne/UserService/internal/model/company"
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

func (s *Service) CreateCompany(ctx context.Context, input CreateCompanyInput) (*CompanyOutput, error) {
	const op = "service.company.CreateCompany"

	company, err := companymodel.NewCompany(input.Name, input.Description)
	if err != nil {
		if errors.Is(err, companymodel.ErrCompanyNameRequired) {
			return nil, service.NewValidationError("name", "field is required")
		}
		return nil, fmt.Errorf("%s: %w", op, service.ErrValidation)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.repo.CreateCompany(ctxTimeout, company); err != nil {
		return nil, fmt.Errorf("%s: create company: %w", op, err)
	}

	return toCompanyOutput(company), nil
}

func (s *Service) GetCompanyByID(ctx context.Context, companyID uuid.UUID) (*CompanyOutput, error) {
	const op = "service.company.GetCompany"

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	company, err := s.repo.GetCompanyByID(ctxTimeout, companyID)
	if err != nil {
		if errors.Is(err, companymodel.ErrCompanyNotFound) {
			return nil, fmt.Errorf("%s: %w", op, service.ErrCompanyNotFound)
		}
		return nil, fmt.Errorf("%s: get company by id: %w", op, err)
	}

	return toCompanyOutput(company), nil
}
