package service

import (
	"context"
	"errors"
	"fmt"

	"user-management-api/internal/core/domain"
)

// Login verifies credentials and returns a signed token plus the user.
//
// An unknown email and a wrong password both return ErrInvalidCredentials, so
// the response cannot be used to discover which accounts exist.
func (s *UserService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", nil, domain.ErrInvalidCredentials
		}
		return "", nil, err
	}

	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(user.ID)
	if err != nil {
		return "", nil, fmt.Errorf("generating token: %w", err)
	}
	return token, user, nil
}
