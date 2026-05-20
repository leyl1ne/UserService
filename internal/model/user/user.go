package user

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	Seller    UserRole = "SELLER"
	Buyer     UserRole = "BUYER"
	Warehouse UserRole = "WAREHOUSE"
	Logistics UserRole = "LOGISTICS"
)

func ParseUserRole(role string) (UserRole, error) {
	switch UserRole(strings.ToUpper(role)) {
	case Seller:
		return Seller, nil
	case Buyer:
		return Buyer, nil
	case Warehouse:
		return Warehouse, nil
	case Logistics:
		return Logistics, nil
	default:
		return "", ErrInvalidUserRole
	}

}

type PasswordHash string

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         UserRole
	CompanyID    *uuid.UUID
	CreatedAt    time.Time
}

func NewUser(email, passwordHash, role string) (*User, error) {

	userRole, err := ParseUserRole(role)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         userRole,
		CreatedAt:    time.Now(),
	}, nil
}
