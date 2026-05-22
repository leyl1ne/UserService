package company

import "errors"

var (
	ErrCompanyNotFound     = errors.New("company not found")
	ErrCompanyNameRequired = errors.New("company required")
)
