package user

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailNotFound      = errors.New("email not found")
	ErrInvalidUserRole    = errors.New("invalid user role")
	ErrEmailAlreadyExists = errors.New("email already exists")
)
