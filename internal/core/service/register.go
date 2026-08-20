package service

import (
	"context"
	"fmt"
	"strings"

	"user-management-api/internal/core/domain"
)

// Register validates the input, hashes the password, and persists a new user.
// The plaintext password never leaves this call.
func (s *UserService) Register(ctx context.Context, name, email, password string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)

	if err := validateName(name); err != nil {
		return nil, err
	}
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	return s.repo.Create(ctx, &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
	})
}
