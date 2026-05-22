package company

import (
	"time"

	"github.com/google/uuid"
)

type Company struct {
	ID          uuid.UUID
	Name        string
	Description *string
	CreatedAt   time.Time
}

func NewCompany(name string, description *string) (*Company, error) {
	if name == "" {
		return nil, ErrCompanyNameRequired
	}

	return &Company{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}, nil
}
