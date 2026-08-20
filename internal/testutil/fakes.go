// Package testutil provides the fakes shared by the adapter tests, so the
// HTTP, gRPC, and middleware suites all exercise the same stand-ins for the
// core ports instead of each keeping its own copy.
package testutil

import (
	"context"
	"strings"
	"time"

	"user-management-api/internal/core/domain"
	"user-management-api/internal/core/port"
)

// User is the canonical fixture returned by the fakes.
var User = &domain.User{
	ID:        "abc123",
	Name:      "Alice",
	Email:     "alice@example.com",
	CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
}

// AuthHeader is an Authorization header value that TokenService accepts.
const AuthHeader = "Bearer token-for-user-1"

// TokenService is a fake port.TokenService whose tokens are the user ID with
// a fixed prefix, so tests can assert on them without cryptography.
type TokenService struct{}

var _ port.TokenService = TokenService{}

func (TokenService) Generate(userID string) (string, error) { return "token-for-" + userID, nil }

func (TokenService) Verify(token string) (string, error) {
	id, ok := strings.CutPrefix(token, "token-for-")
	if !ok || id == "" {
		return "", domain.ErrInvalidToken
	}
	return id, nil
}

// UserService is a fake port.UserService. Each field overrides one use case;
// calling a use case whose field is nil panics, which keeps tests honest
// about what they actually exercise.
type UserService struct {
	RegisterFn func(name, email, password string) (*domain.User, error)
	LoginFn    func(email, password string) (string, *domain.User, error)
	GetByIDFn  func(id string) (*domain.User, error)
	ListFn     func() ([]domain.User, error)
	UpdateFn   func(id string, update domain.UserUpdate) (*domain.User, error)
	DeleteFn   func(id string) error
	CountFn    func() (int64, error)
}

var _ port.UserService = (*UserService)(nil)

func (s *UserService) Register(_ context.Context, name, email, password string) (*domain.User, error) {
	return s.RegisterFn(name, email, password)
}

func (s *UserService) Login(_ context.Context, email, password string) (string, *domain.User, error) {
	return s.LoginFn(email, password)
}

func (s *UserService) GetByID(_ context.Context, id string) (*domain.User, error) {
	return s.GetByIDFn(id)
}

func (s *UserService) List(_ context.Context) ([]domain.User, error) { return s.ListFn() }

func (s *UserService) Update(_ context.Context, id string, update domain.UserUpdate) (*domain.User, error) {
	return s.UpdateFn(id, update)
}

func (s *UserService) Delete(_ context.Context, id string) error { return s.DeleteFn(id) }

func (s *UserService) Count(_ context.Context) (int64, error) { return s.CountFn() }
