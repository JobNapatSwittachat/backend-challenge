// Package service implements the application use cases on top of the ports.
// It contains no HTTP, gRPC, or MongoDB specific code.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"user-management-api/internal/core/domain"
	"user-management-api/internal/core/port"
)

const (
	minPasswordLength = 8
	maxNameLength     = 100
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

// Register validates input, hashes the password, and persists a new user.
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

// Login verifies credentials and returns a signed JWT plus the user.
func (s *UserService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Do not leak whether the email exists.
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

func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}

// Update changes a user's name and/or email after validating them.
func (s *UserService) Update(ctx context.Context, id string, update domain.UserUpdate) (*domain.User, error) {
	if update.Name == nil && update.Email == nil {
		return nil, fmt.Errorf("%w: at least one of name or email is required", domain.ErrValidation)
	}
	if update.Name != nil {
		trimmed := strings.TrimSpace(*update.Name)
		if err := validateName(trimmed); err != nil {
			return nil, err
		}
		update.Name = &trimmed
	}
	if update.Email != nil {
		normalized := normalizeEmail(*update.Email)
		if err := validateEmail(normalized); err != nil {
			return nil, err
		}
		update.Email = &normalized
	}
	return s.repo.Update(ctx, id, update)
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return fmt.Errorf("%w: name must be at most %d characters", domain.ErrValidation, maxNameLength)
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("%w: email is required", domain.ErrValidation)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return fmt.Errorf("%w: email format is invalid", domain.ErrValidation)
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("%w: password is required", domain.ErrValidation)
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: password must be at least %d characters", domain.ErrValidation, minPasswordLength)
	}
	return nil
}
