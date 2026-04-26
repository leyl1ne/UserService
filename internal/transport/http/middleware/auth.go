package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leyl1ne/UserService/internal/infrastructure/auth/jwt"
	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/transport/http/response"
)

type TokenProvider interface {
	Validate(tokenString string) (jwt.Payload, error)
}

func AuthMiddleware(log logger.Logger, tokenProvider TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		log = log.With(logger.Field{Key: "component", Value: "middleware/auth"})

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("authorization header is missing")
			response.WriteErrorAbort(c, http.StatusUnauthorized, "authorization header is required")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		if tokenString == "" || tokenString == authHeader {
			log.Warn("invalid authorization header format")
			response.WriteErrorAbort(c, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		payload, err := tokenProvider.Validate(tokenString)
		if err != nil {
			log.Warn("token validation failed", logger.Field{Key: "error", Value: err.Error()})

			if errors.Is(err, jwt.ErrExpiredToken) {
				response.WriteErrorAbort(c, http.StatusUnauthorized, "token expired")
				return
			}

			response.WriteErrorAbort(c, http.StatusUnauthorized, "invalid token")
			return
		}

		SetUser(c, UserContext{
			UserID:    payload.UserID,
			UserRole:  payload.UserRole,
			CompanyID: payload.CompanyID,
		})

		log.Info("login success")

		c.Next()
	}
}
