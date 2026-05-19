package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	companymodel "github.com/leyl1ne/UserService/internal/model/company"
)

func (r *Repository) CreateCompany(ctx context.Context, c *companymodel.Company) error {
	const op = "repository.postgres.CreateCompany"

	const query = `
		INSERT INTO companies (
			id, name, description
		) VALUES ($1, $2, $3)
	`

	_, err := r.querier(ctx).Exec(
		ctx,
		query,
		c.ID,
		c.Name,
		c.Description,
	)

	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	return nil
}

func (r *Repository) GetCompanyByID(
	ctx context.Context,
	id uuid.UUID,
) (*companymodel.Company, error) {
	const op = "repository.postgres.GetCompanyByID"

	const query = `
		SELECT id, name, description, created_at
		FROM companies
		WHERE id = $1
	`

	var c companymodel.Company

	err := r.querier(ctx).QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.Description,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, companymodel.ErrCompanyNotFound)
		}

		return nil, fmt.Errorf("%s: scan: %w", op, err)
	}

	return &c, nil
}
