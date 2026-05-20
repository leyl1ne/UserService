package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	authmodel "github.com/leyl1ne/UserService/internal/model/auth"
)

func (r *Repository) SaveRefreshToken(ctx context.Context, t *authmodel.RefreshToken) error {
	const op = "repository.postgres.SaveRefreshToken"

	const query = `
		INSERT INTO refresh_tokens (
			token,
			user_id,
			expires_at,
			created_at
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.querier(ctx).Exec(
		ctx,
		query,
		t.Token,
		t.UserID,
		t.ExpiresAt,
		t.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	return nil
}

func (r *Repository) GetRefreshToken(ctx context.Context, token string) (*authmodel.RefreshToken, error) {
	const op = "repository.postgres.GetRefreshToken"

	const query = `
		SELECT token, user_id, expires_at, created_at
		FROM refresh_tokens
		WHERE token = $1
	`

	var t authmodel.RefreshToken

	err := r.querier(ctx).QueryRow(ctx, query, token).Scan(
		&t.Token,
		&t.UserID,
		&t.ExpiresAt,
		&t.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: token not found", op)
		}

		return nil, fmt.Errorf("%s: scan: %w", op, err)
	}

	return &t, nil
}

func (r *Repository) DeleteRefreshToken(ctx context.Context, token string) error {
	const op = "repository.postgres.DeleteRefreshToken"

	const query = `
		DELETE FROM refresh_tokens
		WHERE token = $1
	`

	_, err := r.querier(ctx).Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	return nil
}
