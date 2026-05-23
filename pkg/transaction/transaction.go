package transaction

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey struct{}

type Transaction struct {
	Tx pgx.Tx
}

func Extract(ctx context.Context) *Transaction {
	val := ctx.Value(ctxKey{})
	if val == nil {
		return nil
	}

	tx, ok := val.(*Transaction)
	if !ok {
		return nil
	}

	return tx
}

func SetTransaction(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, ctxKey{}, &Transaction{Tx: tx})
}

func Wrap(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pool.Begin: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	ctx = context.WithValue(ctx, ctxKey{}, &Transaction{Tx: tx})

	if err := fn(ctx); err != nil {
		return fmt.Errorf("fn: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}
