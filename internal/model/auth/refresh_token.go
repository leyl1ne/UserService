package auth

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	Token     uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

func NewRefreshToken(userID uuid.UUID, ttl time.Duration) *RefreshToken {
	now := time.Now()

	return &RefreshToken{
		Token:     uuid.New(),
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
}
