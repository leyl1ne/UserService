package company

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/service"
	companyservice "github.com/leyl1ne/UserService/internal/service/company"
	"github.com/leyl1ne/UserService/internal/transport/http/middleware"
	"github.com/leyl1ne/UserService/internal/transport/http/response"
)

type CompanyService interface {
	CreateCompany(ctx context.Context, input companyservice.CreateCompanyInput) (*companyservice.CompanyOutput, error)
	GetCompanyByID(ctx context.Context, companyID uuid.UUID) (*companyservice.CompanyOutput, error)
}

type CompanyHandler struct {
	log            logger.Logger
	companyService CompanyService
}

func NewCompanyHandler(log logger.Logger, companyService CompanyService) *CompanyHandler {
	return &CompanyHandler{
		log:            log,
		companyService: companyService,
	}
}

func (h *CompanyHandler) CreateCompany() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.company.CreateCompany"

		log := h.log.With(
			logger.Field{Key: "op", Value: op},
			logger.Field{Key: "request_id", Value: middleware.GetRequestID(c)},
		)

		var req CreateCompanyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteBindError(c, err)
			log.Warn("invalid request", logger.Err(err))
			return
		}

		var desc *string
		if req.Description != "" {
			desc = &req.Description
		}

		company, err := h.companyService.CreateCompany(c.Request.Context(), companyservice.CreateCompanyInput{
			Name:        req.Name,
			Description: desc,
		})
		if err != nil {
			var ve service.ValidationError

			switch {
			case errors.As(err, &ve):
				response.WriteServiceValidationError(c, ve)
				log.Warn("invalid request", logger.Err(err))
				return
			default:
				response.WriteInternalServerError(c)
				log.Error("internal server error", logger.Err(err))
				return
			}
		}

		c.JSON(http.StatusCreated, toCompanyResponse(company))
	}
}

func (h *CompanyHandler) GetCompany() gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handler.company.GetCompany"

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

		company, err := h.companyService.GetCompanyByID(c.Request.Context(), companyID)
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

		c.JSON(http.StatusOK, toCompanyResponse(company))
	}
}
