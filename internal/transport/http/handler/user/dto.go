package user

import (
	"time"

	"github.com/google/uuid"
	userservice "github.com/leyl1ne/UserService/internal/service/user"
)

type UserResponse struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	CompanyID *uuid.UUID `json:"company_id,omitempty"`
	CreatedAt string     `json:"created_at"`
}

type UpdateUserRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=64"`
}

func toUserResponse(u *userservice.UserOutput) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		CompanyID: u.CompanyID,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
	}
}
