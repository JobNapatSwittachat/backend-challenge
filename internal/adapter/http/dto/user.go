// Package dto defines the HTTP wire contract: the request bodies the API
// accepts and the response bodies it returns.
//
// Keeping these types out of the handler package means the JSON shape of the
// API is defined in exactly one place, and the core domain entities are never
// serialized directly — a new field on domain.User cannot leak to clients by
// accident.
package dto

import (
	"encoding/json"
	"net/http"
	"time"

	"user-management-api/internal/core/domain"
)

// --- Requests ---

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateUserRequest carries a partial update. A nil field means "unchanged".
type UpdateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

// --- Responses ---

type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// NewUserResponse maps a domain entity onto the wire contract, field by field.
func NewUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

func NewUserResponses(users []domain.User) []UserResponse {
	responses := make([]UserResponse, 0, len(users))
	for i := range users {
		responses = append(responses, NewUserResponse(&users[i]))
	}
	return responses
}

// Decode reads a JSON request body into dst, rejecting unknown fields so
// typos in client payloads surface as errors instead of being ignored.
func Decode(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
