package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/service"
	"github.com/leyl1ne/UserService/internal/service/auth"
	"github.com/leyl1ne/UserService/internal/transport/http/middleware"
	"github.com/leyl1ne/UserService/internal/transport/http/response"
)

type AuthService interface {
	Register(ctx context.Context, input auth.RegisterInput) (*auth.TokensOutput, error)
	Login(ctx context.Context, input auth.LoginInput) (*auth.TokensOutput, error)
	Refresh(ctx context.Context, refreshToken string) (*auth.RefreshOutput, error)
	Logout(ctx context.Context, refreshToken string) error
}

type AuthHandler struct {
	log         logger.Logger
	authService AuthService
}

func NewAuthHandler(log logger.Logger, authService AuthService) *AuthHandler {
	return &AuthHandler{
		log:         log,
		authService: authService,
	}
}

func (h *AuthHandler) Register() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.auth.Register"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteBindError(c, err)
			log.Warn("invalid request", logger.Err(err))
			return
		}

		tokensOutput, err := h.authService.Register(c.Request.Context(), auth.RegisterInput{
			Email:    req.Email,
			Password: req.Password,
			Role:     req.Role,
		})
		if err != nil {
			if errors.Is(err, service.ErrValidation) {
				response.WriteError(c, http.StatusUnprocessableEntity, "validation error")
				log.Warn("validation error", logger.Err(err))
				return
			}

			if errors.Is(err, service.ErrDuplicateEmail) {
				response.WriteError(c, http.StatusConflict, "email already exists")
				log.Warn("duplicate email", logger.Err(err))
				return
			}
			response.WriteInternalServerError(c)
			log.Error("internal server error", logger.Err(err))
			return
		}

		c.JSON(http.StatusCreated, AuthResponse{
			AccessToken:  tokensOutput.AccessToken,
			RefreshToken: tokensOutput.RefreshToken,
		})

	}
}

func (h *AuthHandler) Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.auth.Login"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteBindError(c, err)
			log.Warn("invalid request", logger.Err(err))
			return
		}

		tokensOutput, err := h.authService.Login(c.Request.Context(), auth.LoginInput{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				response.WriteError(c, http.StatusUnauthorized, "invalid credentials")
				log.Warn("invalid credentials", logger.Err(err))
				return
			}
			response.WriteInternalServerError(c)
			log.Error("internal service error", logger.Err(err))
			return
		}

		c.JSON(http.StatusOK, AuthResponse{
			AccessToken:  tokensOutput.AccessToken,
			RefreshToken: tokensOutput.RefreshToken,
		})
	}
}

func (h *AuthHandler) Refresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.auth.Refresh"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		var req RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteBindError(c, err)
			log.Warn("invalid request", logger.Err(err))
			return
		}

		refreshOutput, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
		if err != nil {
			if errors.Is(err, service.ErrInvalidToken) {
				response.WriteError(c, http.StatusUnauthorized, "invalid token")
				log.Warn("invalid token", logger.Err(err))
				return
			}

			if errors.Is(err, service.ErrTokenExpired) {
				response.WriteError(c, http.StatusUnauthorized, "token expired")
				log.Warn("token expired", logger.Err(err))
				return
			}

			response.WriteInternalServerError(c)
			log.Error("internal server error", logger.Err(err))
			return
		}

		c.JSON(http.StatusOK, RefreshResponse{
			AccessToken: refreshOutput.AccessToken,
		})
	}
}

func (h *AuthHandler) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.auth.Logout"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		var req LogoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteBindError(c, err)
			log.Warn("invalid request", logger.Err(err))
			return
		}

		err := h.authService.Logout(c.Request.Context(), req.RefreshToken)
		if err != nil {
			response.WriteInternalServerError(c)
			log.Error("internal server error", logger.Err(err))
			return
		}

		c.Status(http.StatusNoContent)
	}
}
