package testutils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	postgreSQL "github.com/leyl1ne/UserService/internal/infrastructure/postgres"
	"github.com/stretchr/testify/require"
	testContainers "github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupPostgres(t *testing.T) *postgreSQL.Postgres {
	ctx := context.Background()

	dbName := "testdb"
	username := "test"
	password := "test"

	pgContainer, err := pgcontainer.Run(ctx,
		"postgres:15-alpine",
		pgcontainer.WithDatabase(dbName),
		pgcontainer.WithUsername(username),
		pgcontainer.WithPassword(password),
		testContainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err.Error())
		}
	})

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")
	t.Log(host, port.Port())
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		username,
		password,
		host,
		port.Port(),
		dbName,
	)

	ctxPing, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		dbTemp, err := pgxpool.New(ctxPing, dsn)
		if err == nil {
			if pingErr := dbTemp.Ping(ctxPing); pingErr == nil {
				dbTemp.Close()
				break
			}
			dbTemp.Close()
		}
		select {
		case <-ctxPing.Done():
			t.Fatal("postgres never became ready")
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
	db, err := postgreSQL.Open(ctx, postgreSQL.Config{
		DSN:             dsn,
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Hour,
	})
	require.NoError(t, err)

	sqlDB := stdlib.OpenDBFromPool(db.Pool())

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	require.NoError(t, err)

	d, err := iofs.New(postgreSQL.MigrationsFS, "migrations")
	require.NoError(t, err)

	m, err := migrate.NewWithInstance("iofs", d, dbName, driver)
	require.NoError(t, err)

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}
