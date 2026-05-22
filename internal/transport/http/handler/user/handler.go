package user

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/service"
	userservice "github.com/leyl1ne/UserService/internal/service/user"
	"github.com/leyl1ne/UserService/internal/transport/http/middleware"
	"github.com/leyl1ne/UserService/internal/transport/http/response"
)

type UserService interface {
	GetUserByID(ctx context.Context, userID uuid.UUID) (*userservice.UserOutput, error)
	UpdateUser(ctx context.Context, input userservice.UpdateUserInput) (*userservice.UserOutput, error)
	ListUsersByCompany(ctx context.Context, companyID uuid.UUID) ([]userservice.UserOutput, error)
}

type UserHandler struct {
	log         logger.Logger
	userService UserService
}

func NewUserHandler(log logger.Logger, userService UserService) *UserHandler {
	return &UserHandler{
		log:         log,
		userService: userService,
	}

}

func (h *UserHandler) GetCurrentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.user.GetCurrentUser"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		userCtx, ok := middleware.GetUser(c)
		if !ok {
			response.WriteError(c, http.StatusUnauthorized, "unauthorized")
			log.Warn("unauthorized request")
			return
		}

		userID, err := uuid.Parse(userCtx.UserID)
		if err != nil {
			response.WriteError(c, http.StatusUnauthorized, "invalid user id")
			log.Warn("invalid user id", logger.Err(err))
			return
		}

		user, err := h.userService.GetUserByID(c.Request.Context(), userID)
		if err != nil {

			switch {
			case errors.Is(err, service.ErrUserNotFound):
				response.WriteError(c, http.StatusUnauthorized, "unauthorized")
				log.Warn("authenticated user not found", logger.Err(err))
				return
			default:
				response.WriteInternalServerError(c)
				log.Error("internal server error", logger.Err(err))
				return
			}
		}

		c.JSON(http.StatusOK, toUserResponse(user))
	}
}

func (h *UserHandler) GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.user.GetUser"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		userID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, "invalid user id")
			log.Warn("invalid user id", logger.Err(err))
			return
		}

		user, err := h.userService.GetUserByID(c.Request.Context(), userID)
		if err != nil {

			switch {
			case errors.Is(err, service.ErrUserNotFound):
				response.WriteError(c, http.StatusNotFound, "user not found")
				log.Warn("authenticated user not found", logger.Err(err))
				return
			default:
				response.WriteInternalServerError(c)
				log.Error("internal server error", logger.Err(err))
				return
			}
		}

		c.JSON(http.StatusOK, toUserResponse(user))
	}
}

func (h *UserHandler) UpdateCurrentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.user.UpdateCurrentUser"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		userCtx, ok := middleware.GetUser(c)
		if !ok {
			response.WriteError(c, http.StatusUnauthorized, "unauthorized")
			log.Warn("unauthorized request")
			return
		}

		userID, err := uuid.Parse(userCtx.UserID)
		if err != nil {
			response.WriteError(c, http.StatusUnauthorized, "invalid user id")
			log.Warn("invalid user id", logger.Err(err))
			return
		}

		var req UpdateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteBindError(c, err)
			log.Warn("invalid request", logger.Err(err))
			return
		}

		user, err := h.userService.UpdateUser(c.Request.Context(), userservice.UpdateUserInput{
			UserID: userID,
			Email:  req.Email,
		})
		if err != nil {

			switch {
			case errors.Is(err, service.ErrDuplicateEmail):
				response.WriteError(c, http.StatusConflict, "email already exists")
				log.Warn("email already exists", logger.Err(err))
				return
			case errors.Is(err, service.ErrUserNotFound):
				response.WriteError(c, http.StatusUnauthorized, "unauthorized")
				log.Warn("authenticated user not found", logger.Err(err))
				return
			default:
				response.WriteInternalServerError(c)
				log.Error("internal server error", logger.Err(err))
				return
			}
		}

		c.JSON(http.StatusOK, toUserResponse(user))
	}
}

func (h *UserHandler) ListUsersByCompany() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.user.ListUsersByCompany"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		companyID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, "invalid company id")
			log.Warn("invalid company id", logger.Err(err))
			return
		}

		users, err := h.userService.ListUsersByCompany(c.Request.Context(), companyID)
		if err != nil {

			switch {
			case errors.Is(err, service.ErrCompanyNotFound):
				response.WriteError(c, http.StatusNotFound, "company not found")
				log.Warn("company not found", logger.Err(err))
				return
			default:
				response.WriteInternalServerError(c)
				log.Error("internal server error", logger.Err(err))
				return
			}
		}

		result := make([]UserResponse, 0, len(users))
		for _, u := range users {
			result = append(result, toUserResponse(&u))
		}

		c.JSON(http.StatusOK, result)
	}
}
