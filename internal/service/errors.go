package service

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrValidation         = errors.New("validation failed")
	ErrUserNotFound       = errors.New("user not found")
	ErrCompanyNotFound    = errors.New("company not found")
)

type ValidationError struct {
	Fields map[string]string
}

func (e ValidationError) Error() string {
	return "business validation failed"
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Fields: map[string]string{
			field: message,
		},
	}
}
