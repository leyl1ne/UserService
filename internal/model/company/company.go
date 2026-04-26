package company

import (
	"time"

	"github.com/google/uuid"
)

type Company struct {
	ID          uuid.UUID
	Name        string
	Description *string
	CreatedAt   time.Time
}
