package dto

import (
	"time"

	"user-management-api/internal/core/domain"
)

// UserResponse is the public representation of a user. Fields are mapped
// explicitly, so nothing added to domain.User reaches clients unreviewed.
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

// NewUserResponses always returns a non-nil slice so an empty result
// serializes as [] rather than null.
func NewUserResponses(users []domain.User) []UserResponse {
	responses := make([]UserResponse, 0, len(users))
	for i := range users {
		responses = append(responses, NewUserResponse(&users[i]))
	}
	return responses
}
