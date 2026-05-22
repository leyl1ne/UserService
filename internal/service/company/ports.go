package company

import (
	"context"
	"time"

	"github.com/google/uuid"
	companymodel "github.com/leyl1ne/UserService/internal/model/company"
)

type Repository interface {
	CreateCompany(ctx context.Context, c *companymodel.Company) error
	GetCompanyByID(ctx context.Context, id uuid.UUID) (*companymodel.Company, error)
}

type CreateCompanyInput struct {
	Name        string
	Description *string
}

type CompanyOutput struct {
	ID          uuid.UUID
	Name        string
	Description *string
	CreatedAt   time.Time
}

func toCompanyOutput(c *companymodel.Company) *CompanyOutput {
	return &CompanyOutput{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		CreatedAt:   c.CreatedAt,
	}
}
