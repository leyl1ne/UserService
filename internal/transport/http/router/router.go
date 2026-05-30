package router

import (
	"github.com/gin-gonic/gin"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/jwt"
	"github.com/leyl1ne/UserService/internal/logger"
	authandler "github.com/leyl1ne/UserService/internal/transport/http/handler/auth"
	companyhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/company"
	docshandler "github.com/leyl1ne/UserService/internal/transport/http/handler/docs"
	healthhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/health"
	userhandler "github.com/leyl1ne/UserService/internal/transport/http/handler/user"
	"github.com/leyl1ne/UserService/internal/transport/http/middleware"
)

type Handlers struct {
	AuthHandler    *authandler.AuthHandler
	UserHandler    *userhandler.UserHandler
	CompanyHandler *companyhandler.CompanyHandler
	HealthHandler  *healthhandler.HealthHandler
	DocsHandler    *docshandler.DocsHandler
}

type TokenProvider interface {
	Validate(tokenString string) (jwt.Payload, error)
}

func SetupRouter(
	log logger.Logger,
	handlers Handlers,
	tokenProvider TokenProvider,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.LoggerMiddleware(log))

	// Health check
	router.GET("/health", handlers.HealthHandler.Health())

	// Documentation for api
	router.GET("/docs", handlers.DocsHandler.Redirect())
	router.GET("/swagger", handlers.DocsHandler.UI())
	router.GET("/swagger/api.yaml", handlers.DocsHandler.Spec())

	// Auth routes (public)
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", handlers.AuthHandler.Register())
		authGroup.POST("/login", handlers.AuthHandler.Login())
		authGroup.POST("/refresh", handlers.AuthHandler.Refresh())
		authGroup.POST("/logout", handlers.AuthHandler.Logout())
	}

	// Authenticated routes
	authorized := router.Group("")
	authorized.Use(middleware.AuthMiddleware(log, tokenProvider))
	{
		// User routes
		usersGroup := authorized.Group("/users")
		{
			usersGroup.GET("/me", handlers.UserHandler.GetCurrentUser())
			usersGroup.PATCH("/me", handlers.UserHandler.UpdateCurrentUser())
			usersGroup.GET("/:id", handlers.UserHandler.GetUser())
		}

		// Company routes
		companiesGroup := authorized.Group("/companies")
		{
			companiesGroup.POST("", handlers.CompanyHandler.CreateCompany())
			companiesGroup.GET("/:id", handlers.CompanyHandler.GetCompany())
			companiesGroup.GET("/:id/users", handlers.UserHandler.ListUsersByCompany())
		}
	}

	return router
}
