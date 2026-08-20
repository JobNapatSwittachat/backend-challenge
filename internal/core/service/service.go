// Package service implements the application use cases on top of the ports.
// It contains no HTTP, gRPC, or MongoDB specific code.
//
// Each use case lives in its own file (register.go, login.go, …); this file
// holds the shared construction and the validation rules live in validate.go.
package service

import (
	"user-management-api/internal/core/port"
)

// UserService implements port.UserService.
type UserService struct {
	repo   port.UserRepository
	hasher port.PasswordHasher
	tokens port.TokenService
}

// compile-time check that UserService satisfies the driving port.
var _ port.UserService = (*UserService)(nil)

func NewUserService(repo port.UserRepository, hasher port.PasswordHasher, tokens port.TokenService) *UserService {
	return &UserService{repo: repo, hasher: hasher, tokens: tokens}
}
