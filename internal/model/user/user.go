package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	CompanyID    *uuid.UUID
	CreatedAt    time.Time
}
