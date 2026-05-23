package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	usermodel "github.com/leyl1ne/UserService/internal/model/user"
)

func (r *Repository) CreateUser(ctx context.Context, user *usermodel.User) error {
	const op = "repository.postgres.CreateUser"

	const query = `
		INSERT INTO users (
			id, email, password_hash, role, company_id
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.querier(ctx).Exec(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CompanyID,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, usermodel.ErrEmailAlreadyExists)
		}
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	return nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*usermodel.User, error) {
	const op = "repository.postgres.GetUserByID"

	const query = `
		SELECT id, email, password_hash, role, company_id, created_at
		FROM users
		WHERE id = $1
	`

	var u usermodel.User

	err := r.querier(ctx).QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.CompanyID,
		&u.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, usermodel.ErrUserNotFound)
		}

		return nil, fmt.Errorf("%s: scan: %w", op, err)
	}

	return &u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*usermodel.User, error) {
	const op = "repository.postgres.GetUserByEmail"

	const query = `
		SELECT id, email, password_hash, role, company_id, created_at
		FROM users
		WHERE email = $1
	`

	var u usermodel.User

	err := r.querier(ctx).QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.CompanyID,
		&u.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, usermodel.ErrEmailNotFound)
		}

		return nil, fmt.Errorf("%s: scan: %w", op, err)
	}

	return &u, nil
}

func (r *Repository) UpdateUser(ctx context.Context, userID uuid.UUID, email string) error {
	const op = "repository.postgres.UpdateUserEmail"

	const query = `
		UPDATE users
		SET email = $2
		WHERE id = $1
	`

	tag, err := r.querier(ctx).Exec(ctx, query, userID, email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, usermodel.ErrEmailAlreadyExists)
		}
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, usermodel.ErrUserNotFound)
	}

	return nil
}

func (r *Repository) BindUserToCompany(
	ctx context.Context,
	userID uuid.UUID,
	companyID uuid.UUID,
) error {
	const op = "repository.postgres.BindUserToCompany"

	const query = `
		UPDATE users
		SET company_id = $2
		WHERE id = $1
	`

	tag, err := r.querier(ctx).Exec(ctx, query, userID, companyID)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, usermodel.ErrUserNotFound)
	}

	return nil
}

func (r *Repository) ListUsersByCompanyID(
	ctx context.Context,
	companyID uuid.UUID,
) ([]usermodel.User, error) {
	const op = "repository.postgres.ListUsersByCompanyID"

	const query = `
		SELECT id, email, password_hash, role, company_id, created_at
		FROM users
		WHERE company_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.querier(ctx).Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("%s: query: %w", op, err)
	}
	defer rows.Close()

	var result []usermodel.User

	for rows.Next() {
		var u usermodel.User

		err = rows.Scan(
			&u.ID,
			&u.Email,
			&u.PasswordHash,
			&u.Role,
			&u.CompanyID,
			&u.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}

		result = append(result, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows: %w", op, err)
	}

	return result, nil
}
