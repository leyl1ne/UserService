package company

import (
	"time"

	"github.com/google/uuid"
	companyservice "github.com/leyl1ne/UserService/internal/service/company"
)

type CreateCompanyRequest struct {
	Name        string `json:"name" binding:"required,max=255"`
	Description string `json:"description" binding:"max=1000"`
}

type CompanyResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

func toCompanyResponse(c *companyservice.CompanyOutput) CompanyResponse {
	return CompanyResponse{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		CreatedAt:   c.CreatedAt.UTC().Format(time.RFC3339),
	}
}
