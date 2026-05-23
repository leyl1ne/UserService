package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/leyl1ne/UserService/internal/config/app"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/jwt"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/password"
	postgresInfra "github.com/leyl1ne/UserService/internal/infrastructure/postgres"
	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/logger/zl"
	postgresRepo "github.com/leyl1ne/UserService/internal/repository/postgres"
	httpServer "github.com/leyl1ne/UserService/internal/transport/http"
	authandler "github.com/leyl1ne/UserService/internal/transport/http/handler/auth"
	companyhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/company"
	docshandler "github.com/leyl1ne/UserService/internal/transport/http/handler/docs"
	healthhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/health"
	userhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/user"

	authservice "github.com/leyl1ne/UserService/internal/service/auth"
	companyservice "github.com/leyl1ne/UserService/internal/service/company"
	userservice "github.com/leyl1ne/UserService/internal/service/user"
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log := zl.NewZerologLogger(cfg.Logger.Level, os.Stderr)

	log.Info("http server started!", logger.Field{Key: "Addr", Value: cfg.Server.HTTP.Port})
	log.Debug("logger debug mode enabled")

	// можно ли передавать notidy context ?
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	db, err := postgresInfra.Open(ctx, cfg.Postgres)
	if err != nil {
		log.Fatal("failed to connect database", logger.Field{Key: "error", Value: err.Error()})
		os.Exit(1)
	}

	repo := postgresRepo.NewRepository(db.Pool())
	passwordHasher, err := password.NewHasher(cfg.PasswordHasher)
	if err != nil {
		log.Fatal("failed to create password hasher", logger.Field{Key: "error", Value: err.Error()})
		os.Exit(1)
	}
	jwtGenerator := jwt.NewJWTGenerator(cfg.JWTGenerator)

	refreshTokenTTL := 7 * time.Hour
	authService := authservice.NewService(repo, passwordHasher, jwtGenerator, refreshTokenTTL)
	userService := userservice.NewService(repo)
	companyService := companyservice.NewService(repo)

	authHandler := authandler.NewAuthHandler(log, authService)
	userHandler := userhandler.NewUserHandler(log, userService)
	companyHandler := companyhandler.NewCompanyHandler(log, companyService)
	healthHandler := healthhandler.NewHealthHandler(cfg.App.Version, cfg.App.Name, cfg.App.Environment)
	docsHanler, err := docshandler.NewDocsHandler()
	if err != nil {
		log.Fatal("failed to create docs handler", logger.Err(err))
	}

	router := httpServer.SetupRouter(log, httpServer.Handlers{
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
		CompanyHandler: companyHandler,
		HealthHandler:  healthHandler,
		DocsHandler:    docsHanler,
	}, jwtGenerator)

	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTP.Port),
		Handler:      router.Handler(),
		ReadTimeout:  cfg.Server.HTTP.ReadTimeout,
		WriteTimeout: cfg.Server.HTTP.WriteTimeout,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", logger.Err(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulShutdown)
	defer shutdownCancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to stop server gracefully", logger.Err(err))

		if err := s.Close(); err != nil {
			log.Error("forced shutdown failed", logger.Err(err))
		}
		return

	}

	log.Info("server exiting")

	if err := db.Close(); err != nil {
		log.Error("failed to close database client", logger.Field{Key: "error", Value: err.Error()})
	}

}
